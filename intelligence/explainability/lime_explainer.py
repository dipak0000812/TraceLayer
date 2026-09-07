"""LIME explainer wrapper for Isolation Forest.

Uses LIME's TabularExplainer to produce local surrogate explanations
for individual anomaly predictions.

LIME is the SECONDARY explainability method (on-demand only):
  - Perturbation-based (slower: ~500ms-2s per instance)
  - Creates local linear surrogate around each prediction
  - Useful as a second opinion on specific high-priority leads
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

import numpy as np

from intelligence.config import FEATURE_NAMES, MAX_TOP_FEATURES

if TYPE_CHECKING:
    from intelligence.models.isolation_forest import IsolationForestModel

logger = logging.getLogger(__name__)


class LimeExplainerWrapper:
    """LIME TabularExplainer wrapper for Isolation Forest explanations."""

    def __init__(self, model: IsolationForestModel):
        self._model = model
        self._explainer = None
        self._training_data: np.ndarray | None = None
        self._initialized = False

    def _ensure_initialized(self) -> bool:
        """Lazily initialize the LIME explainer."""
        if self._initialized:
            return True

        if self._model.sklearn_model is None:
            logger.warning("Cannot initialize LIME: model not loaded.")
            return False

        try:
            from lime.lime_tabular import LimeTabularExplainer

            # LIME needs a background dataset. If we have the scaler's
            # training stats, generate a synthetic background.
            if self._model.scaler is not None and hasattr(self._model.scaler, "mean_"):
                mean = self._model.scaler.mean_
                std = self._model.scaler.scale_
                rng = np.random.RandomState(42)
                # Generate 100 synthetic background samples
                self._training_data = rng.normal(
                    loc=mean, scale=std, size=(100, len(mean))
                )
            else:
                # Fallback: use a small uniform background
                n_features = len(FEATURE_NAMES)
                self._training_data = np.random.RandomState(42).uniform(
                    0, 1, size=(100, n_features)
                )

            self._explainer = LimeTabularExplainer(
                training_data=self._training_data,
                feature_names=FEATURE_NAMES,
                class_names=["Normal", "Anomaly"],
                mode="regression",
                random_state=42,
            )
            self._initialized = True
            logger.info("LIME TabularExplainer initialized successfully.")
            return True

        except Exception as exc:
            logger.warning("LIME initialization failed: %s", exc)
            return False

    def _predict_fn(self, X: np.ndarray) -> np.ndarray:
        """Prediction function for LIME.

        LIME calls this with perturbed samples to build a local
        surrogate model.  We return anomaly scores (higher = more anomalous).
        """
        scores = self._model.score_batch(X)
        return scores

    def explain(self, feature_vector: np.ndarray) -> list[dict]:
        """Explain a single prediction using LIME.

        Parameters
        ----------
        feature_vector : np.ndarray
            1-D feature array of shape (n_features,).

        Returns
        -------
        list[dict]
            Top contributing features with LIME weights.
            Each dict has: {feature, value, direction}.
        """
        if not self._ensure_initialized():
            return self._fallback_explain(feature_vector)

        try:
            explanation = self._explainer.explain_instance(
                data_row=feature_vector,
                predict_fn=self._predict_fn,
                num_features=MAX_TOP_FEATURES,
            )

            # Extract feature attributions
            attributions = []
            for feature_name, weight in explanation.as_list():
                # LIME returns feature names with conditions (e.g., "fee <= 0.01")
                # We extract the base feature name
                base_name = self._extract_feature_name(feature_name)

                attributions.append({
                    "feature": base_name,
                    "value": round(float(weight), 6),
                    "direction": "risk_increasing" if weight > 0 else "risk_decreasing",
                })

            attributions.sort(key=lambda x: abs(x["value"]), reverse=True)
            return attributions[:MAX_TOP_FEATURES]

        except Exception as exc:
            logger.warning("LIME explain failed: %s. Using fallback.", exc)
            return self._fallback_explain(feature_vector)

    @staticmethod
    def _extract_feature_name(lime_feature_str: str) -> str:
        """Extract the base feature name from LIME's conditional string.

        LIME returns strings like "fee <= 0.01" or "3.50 < tx_amount <= 10.0".
        We extract just the feature name.
        """
        for name in FEATURE_NAMES:
            if name in lime_feature_str:
                return name
        # Fallback: return the full string
        return lime_feature_str.split()[0] if lime_feature_str else "unknown"

    def _fallback_explain(self, feature_vector: np.ndarray) -> list[dict]:
        """Simple fallback when LIME is unavailable."""
        attributions = []
        for i, name in enumerate(FEATURE_NAMES):
            val = feature_vector[i] if i < len(feature_vector) else 0.0
            attributions.append({
                "feature": name,
                "value": round(float(val) * 0.01, 6),
                "direction": "risk_increasing" if val > 0 else "risk_decreasing",
            })
        attributions.sort(key=lambda x: abs(x["value"]), reverse=True)
        return attributions[:MAX_TOP_FEATURES]

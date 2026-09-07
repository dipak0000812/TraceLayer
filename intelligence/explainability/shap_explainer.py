"""SHAP explainer wrapper for Isolation Forest.

Uses SHAP's TreeExplainer (optimized for tree-based models) to produce
per-feature attributions that explain WHY a specific transaction was
scored as anomalous.

SHAP is the PRIMARY explainability method:
  - Mathematically exact (Shapley values from cooperative game theory)
  - Fast (~10ms per transaction with TreeExplainer)
  - Provides both local (per-instance) and global feature importance
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

import numpy as np

from intelligence.config import FEATURE_NAMES, MAX_TOP_FEATURES

if TYPE_CHECKING:
    from intelligence.models.isolation_forest import IsolationForestModel

logger = logging.getLogger(__name__)


class ShapExplainerWrapper:
    """SHAP TreeExplainer wrapper for Isolation Forest explanations."""

    def __init__(self, model: IsolationForestModel):
        self._model = model
        self._explainer = None
        self._initialized = False

    def _ensure_initialized(self) -> bool:
        """Lazily initialize the SHAP explainer.

        We defer initialization because SHAP import is heavy and
        the model might not be loaded yet at construction time.
        """
        if self._initialized:
            return True

        if self._model.sklearn_model is None:
            logger.warning("Cannot initialize SHAP: model not loaded.")
            return False

        try:
            import shap

            self._explainer = shap.TreeExplainer(
                self._model.sklearn_model,
                feature_names=FEATURE_NAMES,
            )
            self._initialized = True
            logger.info("SHAP TreeExplainer initialized successfully.")
            return True
        except Exception as exc:
            logger.warning(
                "SHAP TreeExplainer initialization failed: %s. "
                "Falling back to feature-importance-based explanations.",
                exc,
            )
            return False

    def explain(self, feature_vector: np.ndarray) -> list[dict]:
        """Explain a single prediction using SHAP values.

        Parameters
        ----------
        feature_vector : np.ndarray
            1-D feature array of shape (n_features,).

        Returns
        -------
        list[dict]
            Top contributing features, sorted by absolute SHAP value.
            Each dict has: {feature, value, direction}.
        """
        if not self._ensure_initialized():
            return self._fallback_explain(feature_vector)

        try:
            X = feature_vector.reshape(1, -1)

            # Apply scaler if the model uses one
            if self._model.scaler is not None and hasattr(self._model.scaler, "mean_"):
                X = self._model.scaler.transform(X)

            shap_values = self._explainer.shap_values(X)

            # shap_values can be a list (one per class) or 2D array
            if isinstance(shap_values, list):
                values = shap_values[0][0]
            elif shap_values.ndim == 3:
                values = shap_values[0, :, 0]
            else:
                values = shap_values[0]

            return self._format_attributions(values)

        except Exception as exc:
            logger.warning("SHAP explain failed: %s. Using fallback.", exc)
            return self._fallback_explain(feature_vector)

    def explain_batch(self, X: np.ndarray) -> list[list[dict]]:
        """Explain a batch of predictions.

        Parameters
        ----------
        X : np.ndarray
            2-D feature matrix of shape (n_samples, n_features).

        Returns
        -------
        list[list[dict]]
            List of attribution lists, one per sample.
        """
        return [self.explain(X[i]) for i in range(X.shape[0])]

    def _format_attributions(self, shap_values: np.ndarray) -> list[dict]:
        """Format raw SHAP values into sorted attribution dicts.

        Parameters
        ----------
        shap_values : np.ndarray
            1-D array of SHAP values, one per feature.

        Returns
        -------
        list[dict]
            Top N features sorted by |SHAP value|.
        """
        attributions = []
        for i, (name, val) in enumerate(zip(FEATURE_NAMES, shap_values)):
            attributions.append({
                "feature": name,
                "value": round(float(val), 6),
                "direction": "risk_increasing" if val > 0 else "risk_decreasing",
            })

        # Sort by absolute value, return top N
        attributions.sort(key=lambda x: abs(x["value"]), reverse=True)
        return attributions[:MAX_TOP_FEATURES]

    def _fallback_explain(self, feature_vector: np.ndarray) -> list[dict]:
        """Fallback when SHAP is unavailable — use feature deviation from mean.

        This is a simpler heuristic: features that deviate most from
        zero (after scaling) are considered most influential.
        """
        if self._model.scaler is not None and hasattr(self._model.scaler, "mean_"):
            mean = self._model.scaler.mean_
            std = self._model.scaler.scale_
            # Z-score as a proxy for "how unusual is this feature"
            z_scores = (feature_vector - mean) / np.clip(std, 1e-10, None)
        else:
            z_scores = feature_vector

        attributions = []
        for i, (name, z) in enumerate(zip(FEATURE_NAMES, z_scores)):
            attributions.append({
                "feature": name,
                "value": round(float(z) * 0.1, 6),  # scale down to SHAP-like range
                "direction": "risk_increasing" if z > 0 else "risk_decreasing",
            })

        attributions.sort(key=lambda x: abs(x["value"]), reverse=True)
        return attributions[:MAX_TOP_FEATURES]

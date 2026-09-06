"""Isolation Forest wrapper for TraceLayer anomaly detection.

Provides:
  - Loading a pre-trained model from disk
  - Falling back to training on available data if no pre-trained model exists
  - Score normalization to [0, 1]
  - Batch and single-instance scoring
"""

from __future__ import annotations

import logging
from pathlib import Path

import joblib
import numpy as np
from sklearn.ensemble import IsolationForest
from sklearn.preprocessing import StandardScaler

from intelligence.config import DEFAULT_IF_CONFIG, FEATURE_NAMES, IsolationForestConfig

logger = logging.getLogger(__name__)


class IsolationForestModel:
    """Isolation Forest anomaly detector with score normalization.

    The raw ``decision_function`` output from sklearn's Isolation Forest
    is negative for anomalies and positive for inliers.  We normalize to
    [0, 1] where 1 = most anomalous (highest investigative priority).
    """

    def __init__(self, config: IsolationForestConfig | None = None):
        self.config = config or DEFAULT_IF_CONFIG
        self.model: IsolationForest | None = None
        self.scaler: StandardScaler | None = None

        # For score normalization: calibrated on training data
        self._score_min: float = -0.5
        self._score_max: float = 0.5
        self._is_loaded: bool = False

    @property
    def is_loaded(self) -> bool:
        return self._is_loaded

    @property
    def sklearn_model(self) -> IsolationForest | None:
        """Access the underlying sklearn model (needed by SHAP)."""
        return self.model

    def load_or_initialize(self) -> None:
        """Load a pre-trained model or initialize a new one.

        Priority:
          1. Load from pretrained .joblib if it exists
          2. Otherwise, create an untrained model (will score using
             the default decision function which produces reasonable
             but uncalibrated anomaly scores)
        """
        model_path = self.config.model_path

        if model_path.exists():
            self._load_from_disk(model_path)
        else:
            logger.warning(
                "No pre-trained model at %s. Initializing fresh Isolation Forest. "
                "Run 'python -m intelligence.training.train_isolation_forest' to train.",
                model_path,
            )
            self._initialize_fresh()

    def _load_from_disk(self, path: Path) -> None:
        """Load model + scaler + normalization params from a .joblib bundle."""
        logger.info("Loading pre-trained Isolation Forest from %s ...", path)

        bundle = joblib.load(path)
        self.model = bundle["model"]
        self.scaler = bundle.get("scaler")
        self._score_min = bundle.get("score_min", -0.5)
        self._score_max = bundle.get("score_max", 0.5)
        self._is_loaded = True

        logger.info(
            "Loaded Isolation Forest (n_estimators=%d, contamination=%.3f)",
            self.model.n_estimators,
            self.model.contamination,
        )

    def _initialize_fresh(self) -> None:
        """Create an untrained Isolation Forest with configured hyperparams."""
        self.model = IsolationForest(
            n_estimators=self.config.n_estimators,
            max_samples=self.config.max_samples,
            contamination=self.config.contamination,
            max_features=self.config.max_features,
            random_state=self.config.random_state,
            n_jobs=self.config.n_jobs,
        )
        self.scaler = StandardScaler()
        self._is_loaded = True

    def train(self, X: np.ndarray) -> None:
        """Train the Isolation Forest on a feature matrix.

        Parameters
        ----------
        X : np.ndarray
            Training data of shape (n_samples, n_features).
        """
        logger.info("Training Isolation Forest on %d samples ...", X.shape[0])

        # Fit scaler
        self.scaler = StandardScaler()
        X_scaled = self.scaler.fit_transform(X)

        # Fit Isolation Forest
        self.model = IsolationForest(
            n_estimators=self.config.n_estimators,
            max_samples=self.config.max_samples,
            contamination=self.config.contamination,
            max_features=self.config.max_features,
            random_state=self.config.random_state,
            n_jobs=self.config.n_jobs,
        )
        self.model.fit(X_scaled)

        # Calibrate normalization range from training scores
        raw_scores = self.model.decision_function(X_scaled)
        self._score_min = float(np.percentile(raw_scores, 1))
        self._score_max = float(np.percentile(raw_scores, 99))

        self._is_loaded = True
        logger.info(
            "Training complete. Score range: [%.4f, %.4f]",
            self._score_min,
            self._score_max,
        )

    def save(self, path: Path | None = None) -> Path:
        """Save the trained model bundle to disk.

        Parameters
        ----------
        path : Path, optional
            Save location. Defaults to config.model_path.

        Returns
        -------
        Path
            The path where the model was saved.
        """
        save_path = path or self.config.model_path
        save_path.parent.mkdir(parents=True, exist_ok=True)

        bundle = {
            "model": self.model,
            "scaler": self.scaler,
            "score_min": self._score_min,
            "score_max": self._score_max,
            "feature_names": FEATURE_NAMES,
            "config": {
                "n_estimators": self.config.n_estimators,
                "contamination": self.config.contamination,
                "max_features": self.config.max_features,
                "random_state": self.config.random_state,
            },
        }
        joblib.dump(bundle, save_path)
        logger.info("Saved model to %s", save_path)
        return save_path

    def score(self, feature_vector: np.ndarray) -> float:
        """Score a single feature vector.

        Parameters
        ----------
        feature_vector : np.ndarray
            1-D array of shape (n_features,).

        Returns
        -------
        float
            Anomaly score in [0.0, 1.0] where 1.0 = most anomalous.
        """
        if self.model is None:
            return 0.5  # neutral score if model unavailable

        X = feature_vector.reshape(1, -1)

        # Scale if scaler is fitted
        if self.scaler is not None and hasattr(self.scaler, "mean_"):
            X = self.scaler.transform(X)

        raw_score = self.model.decision_function(X)[0]
        return self._normalize_score(raw_score)

    def score_batch(self, X: np.ndarray) -> np.ndarray:
        """Score a batch of feature vectors.

        Parameters
        ----------
        X : np.ndarray
            2-D array of shape (n_samples, n_features).

        Returns
        -------
        np.ndarray
            1-D array of anomaly scores in [0.0, 1.0].
        """
        if self.model is None:
            return np.full(X.shape[0], 0.5)

        if self.scaler is not None and hasattr(self.scaler, "mean_"):
            X = self.scaler.transform(X)

        raw_scores = self.model.decision_function(X)
        return np.array([self._normalize_score(s) for s in raw_scores])

    def _normalize_score(self, raw_score: float) -> float:
        """Normalize raw IF decision_function score to [0, 1].

        sklearn's decision_function returns:
          - Negative values for anomalies (more negative = more anomalous)
          - Positive values for inliers

        We invert and clip to get [0, 1] where 1 = most anomalous.
        """
        score_range = self._score_max - self._score_min
        if score_range <= 0:
            score_range = 1.0

        # Invert: more negative raw_score → higher anomaly score
        normalized = (self._score_max - raw_score) / score_range
        return float(np.clip(normalized, 0.0, 1.0))

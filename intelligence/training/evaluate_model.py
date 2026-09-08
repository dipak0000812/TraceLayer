"""Evaluate an Isolation Forest model against labeled data.

Provides:
  - AUC-ROC, precision, recall, F1 at various thresholds
  - Precision@K (top-K ranked by anomaly score)
  - Confusion matrix summary

Used by:
  - training/train_isolation_forest.py (post-training evaluation)
  - evaluation/harness.py (full evaluation against ground truth)
"""

from __future__ import annotations

import logging

import numpy as np
from sklearn.metrics import (
    average_precision_score,
    classification_report,
    confusion_matrix,
    f1_score,
    precision_score,
    recall_score,
    roc_auc_score,
)

from intelligence.models.isolation_forest import IsolationForestModel

logger = logging.getLogger(__name__)


def evaluate_against_labels(
    model: IsolationForestModel,
    X: np.ndarray,
    y_true: np.ndarray,
    threshold: float = 0.5,
) -> dict[str, float]:
    """Evaluate model scores against binary labels.

    Parameters
    ----------
    model : IsolationForestModel
        Trained model.
    X : np.ndarray
        Feature matrix (n_samples, n_features).
    y_true : np.ndarray
        Binary labels (1=anomaly/illicit, 0=normal/licit).
    threshold : float
        Score threshold for binary classification.

    Returns
    -------
    dict[str, float]
        Dictionary of evaluation metrics.
    """
    # Get anomaly scores (0 to 1, 1 = most anomalous)
    scores = model.score_batch(X)

    # Binary predictions at threshold
    y_pred = (scores >= threshold).astype(int)

    # --- Metrics ---
    metrics: dict[str, float] = {}

    # AUC-ROC (handles the case where all labels might be one class)
    try:
        metrics["auc_roc"] = roc_auc_score(y_true, scores)
    except ValueError:
        metrics["auc_roc"] = 0.0
        logger.warning("AUC-ROC undefined (single class in y_true)")

    # Average Precision (area under precision-recall curve)
    try:
        metrics["average_precision"] = average_precision_score(y_true, scores)
    except ValueError:
        metrics["average_precision"] = 0.0

    # Precision, Recall, F1 at threshold
    metrics["precision"] = precision_score(y_true, y_pred, zero_division=0)
    metrics["recall"] = recall_score(y_true, y_pred, zero_division=0)
    metrics["f1"] = f1_score(y_true, y_pred, zero_division=0)

    # Precision@K
    for k in [10, 20, 50, 100]:
        if len(scores) >= k:
            metrics[f"precision_at_{k}"] = _precision_at_k(y_true, scores, k)

    # Confusion matrix
    tn, fp, fn, tp = confusion_matrix(y_true, y_pred, labels=[0, 1]).ravel()
    metrics["true_positives"] = float(tp)
    metrics["false_positives"] = float(fp)
    metrics["true_negatives"] = float(tn)
    metrics["false_negatives"] = float(fn)

    # Log results
    logger.info("--- Evaluation Results (threshold=%.2f) ---", threshold)
    logger.info("  AUC-ROC:           %.4f", metrics["auc_roc"])
    logger.info("  Average Precision: %.4f", metrics["average_precision"])
    logger.info("  Precision:         %.4f", metrics["precision"])
    logger.info("  Recall:            %.4f", metrics["recall"])
    logger.info("  F1:                %.4f", metrics["f1"])
    logger.info("  TP=%d, FP=%d, TN=%d, FN=%d", tp, fp, tn, fn)

    for k in [10, 20, 50, 100]:
        key = f"precision_at_{k}"
        if key in metrics:
            logger.info("  Precision@%d:      %.4f", k, metrics[key])

    # Full classification report
    logger.info(
        "\n%s",
        classification_report(
            y_true, y_pred, target_names=["Normal", "Anomaly"], zero_division=0
        ),
    )

    return metrics


def _precision_at_k(y_true: np.ndarray, scores: np.ndarray, k: int) -> float:
    """Precision in the top-K scored instances.

    Parameters
    ----------
    y_true : np.ndarray
        Binary labels.
    scores : np.ndarray
        Anomaly scores (higher = more anomalous).
    k : int
        Number of top instances to consider.

    Returns
    -------
    float
        Fraction of top-K that are actual anomalies.
    """
    top_k_idx = np.argsort(scores)[::-1][:k]
    return float(y_true[top_k_idx].sum() / k)

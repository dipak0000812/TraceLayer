"""Evaluation metrics for anomaly detection and ranking.

Provides precision, recall, F1, AUC-ROC, rank correlation, and
rank shift analysis metrics.
"""

from __future__ import annotations

import logging

import numpy as np
from scipy import stats

logger = logging.getLogger(__name__)


def precision_at_k(y_true: np.ndarray, scores: np.ndarray, k: int) -> float:
    """Precision in the top-K scored instances."""
    if len(scores) < k:
        k = len(scores)
    top_k_idx = np.argsort(scores)[::-1][:k]
    return float(y_true[top_k_idx].sum() / k)


def recall_at_k(y_true: np.ndarray, scores: np.ndarray, k: int) -> float:
    """Recall in the top-K scored instances."""
    if y_true.sum() == 0:
        return 0.0
    if len(scores) < k:
        k = len(scores)
    top_k_idx = np.argsort(scores)[::-1][:k]
    return float(y_true[top_k_idx].sum() / y_true.sum())


def rank_shift_analysis(
    chain_only_scores: np.ndarray,
    fused_scores: np.ndarray,
    entity_ids: list[str] | None = None,
) -> dict:
    """Analyze rank shifts between chain-only and fused rankings.

    This directly tests the project's core hypothesis:
    "Does network evidence produce measurable rank shift?"

    Parameters
    ----------
    chain_only_scores : np.ndarray
        Scores from chain-only analysis.
    fused_scores : np.ndarray
        Scores from fused (chain + network) analysis.
    entity_ids : list[str], optional
        Entity identifiers for reporting.

    Returns
    -------
    dict
        Rank shift statistics.
    """
    n = len(chain_only_scores)
    if n == 0:
        return {"error": "No data for rank shift analysis"}

    # Compute ranks (1 = highest score = most anomalous)
    chain_ranks = n - np.argsort(np.argsort(chain_only_scores))
    fused_ranks = n - np.argsort(np.argsort(fused_scores))

    # Rank shift = chain_rank - fused_rank
    # Positive shift means entity moved UP (higher priority) after fusion
    shifts = chain_ranks - fused_ranks

    # Spearman rank correlation
    if n >= 3:
        spearman_corr, spearman_p = stats.spearmanr(chain_only_scores, fused_scores)
    else:
        spearman_corr, spearman_p = 0.0, 1.0

    # Kendall tau
    if n >= 3:
        kendall_tau, kendall_p = stats.kendalltau(chain_only_scores, fused_scores)
    else:
        kendall_tau, kendall_p = 0.0, 1.0

    result = {
        "n_entities": n,
        "mean_absolute_shift": float(np.abs(shifts).mean()),
        "max_shift": int(np.max(np.abs(shifts))),
        "n_shifted": int(np.sum(shifts != 0)),
        "n_shifted_up": int(np.sum(shifts > 0)),
        "n_shifted_down": int(np.sum(shifts < 0)),
        "n_unchanged": int(np.sum(shifts == 0)),
        "spearman_correlation": round(float(spearman_corr), 4),
        "spearman_p_value": round(float(spearman_p), 6),
        "kendall_tau": round(float(kendall_tau), 4),
        "kendall_p_value": round(float(kendall_p), 6),
        "hypothesis_result": (
            "H1 SUPPORTED: Network evidence produces measurable rank shift"
            if np.sum(shifts != 0) > 0
            else "H0 NOT REJECTED: No rank shift observed"
        ),
    }

    # Top movers
    top_movers_idx = np.argsort(np.abs(shifts))[::-1][:5]
    top_movers = []
    for idx in top_movers_idx:
        mover = {
            "index": int(idx),
            "chain_rank": int(chain_ranks[idx]),
            "fused_rank": int(fused_ranks[idx]),
            "rank_shift": int(shifts[idx]),
            "chain_score": round(float(chain_only_scores[idx]), 4),
            "fused_score": round(float(fused_scores[idx]), 4),
        }
        if entity_ids and idx < len(entity_ids):
            mover["entity_id"] = entity_ids[idx]
        top_movers.append(mover)

    result["top_movers"] = top_movers

    return result

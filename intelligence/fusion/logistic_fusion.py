"""Logistic sigmoid fusion — combines chain, network, and mixing scores.

Implements the fusion formula from DATA_CONTRACT.md §5.2:

    fused_score = σ(w_chain · S_chain + w_net · S_net · Q(tx) − w_mixing · S_mixing + bias)

Where:
    σ(x) = 1 / (1 + e^(-x))        — logistic sigmoid
    S_chain  ∈ [0, 1]               — Isolation Forest chain anomaly score
    S_net    ∈ [0, 1]               — Network propagation anomaly score
    Q(tx)    ∈ [0, 1]               — Network evidence quality
    S_mixing ∈ [0, 1]               — Mixing/peeling penalty
    w_chain, w_net, w_mixing, bias  — tunable fusion weights

The key design: network evidence is DISCOUNTED by Q(tx). If Q(tx) = 0
(no network observations), the network contribution vanishes and the
fused score degrades gracefully to chain-only analysis.
"""

from __future__ import annotations

import math

from intelligence.config import DEFAULT_FUSION_WEIGHTS, FusionWeights


class LogisticFusion:
    """Logistic sigmoid evidence fusion."""

    def __init__(self, weights: FusionWeights | None = None):
        self.weights = weights or DEFAULT_FUSION_WEIGHTS

    def fuse(
        self,
        chain_score: float,
        network_score: float,
        mixing_penalty: float,
        q_tx: float,
    ) -> float:
        """Compute the fused investigative priority score.

        Parameters
        ----------
        chain_score : float
            Isolation Forest chain anomaly score ∈ [0, 1].
        network_score : float
            Network propagation anomaly score ∈ [0, 1].
        mixing_penalty : float
            CoinJoin/peeling chain penalty ∈ [0, 1].
        q_tx : float
            Network evidence quality Q(tx) ∈ [0, 1].

        Returns
        -------
        float
            Fused score ∈ [0, 1].  Higher = higher investigative priority.
        """
        logit = (
            self.weights.w_chain * chain_score
            + self.weights.w_net * network_score * q_tx
            - self.weights.w_mixing * mixing_penalty
            + self.weights.bias
        )
        return self._sigmoid(logit)

    def fuse_chain_only(self, chain_score: float) -> float:
        """Compute chain-only score (for rank shift comparison).

        This is what the fused score would be with NO network evidence
        (Q(tx) = 0, mixing_penalty = 0).  Used to compute rank_shift.

        Parameters
        ----------
        chain_score : float
            Chain anomaly score ∈ [0, 1].

        Returns
        -------
        float
            Chain-only fused score ∈ [0, 1].
        """
        logit = self.weights.w_chain * chain_score + self.weights.bias
        return self._sigmoid(logit)

    def decompose(
        self,
        chain_score: float,
        network_score: float,
        mixing_penalty: float,
        q_tx: float,
    ) -> dict[str, float]:
        """Decompose the fusion into individual contributions.

        Useful for the evidence card / comparison toggle in the frontend.

        Returns
        -------
        dict
            Breakdown of each component's contribution to the logit.
        """
        chain_contrib = self.weights.w_chain * chain_score
        net_contrib = self.weights.w_net * network_score * q_tx
        mix_contrib = self.weights.w_mixing * mixing_penalty
        bias = self.weights.bias

        logit = chain_contrib + net_contrib - mix_contrib + bias
        fused = self._sigmoid(logit)
        chain_only = self.fuse_chain_only(chain_score)

        return {
            "chain_contribution": round(chain_contrib, 4),
            "network_contribution": round(net_contrib, 4),
            "mixing_penalty_contribution": round(-mix_contrib, 4),
            "bias": round(bias, 4),
            "logit": round(logit, 4),
            "fused_score": round(fused, 4),
            "chain_only_score": round(chain_only, 4),
            "network_uplift": round(fused - chain_only, 4),
            "weights_used": {
                "w_chain": self.weights.w_chain,
                "w_net": self.weights.w_net,
                "w_mixing": self.weights.w_mixing,
                "bias": self.weights.bias,
            },
        }

    @staticmethod
    def _sigmoid(x: float) -> float:
        """Logistic sigmoid function, numerically stable."""
        if x >= 0:
            return 1.0 / (1.0 + math.exp(-x))
        else:
            exp_x = math.exp(x)
            return exp_x / (1.0 + exp_x)

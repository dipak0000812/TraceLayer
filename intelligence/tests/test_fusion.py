"""Tests for logistic fusion scoring."""

import math

import pytest

from intelligence.config import FusionWeights
from intelligence.fusion.logistic_fusion import LogisticFusion


class TestLogisticFusion:
    def setup_method(self):
        self.fusion = LogisticFusion()

    def test_sigmoid_bounds(self):
        """Fused score must be in [0, 1]."""
        for chain in [0.0, 0.5, 1.0]:
            for net in [0.0, 0.5, 1.0]:
                for mix in [0.0, 0.5, 1.0]:
                    for q in [0.0, 0.5, 1.0]:
                        score = self.fusion.fuse(chain, net, mix, q)
                        assert 0.0 <= score <= 1.0

    def test_zero_network_quality(self):
        """When Q(tx) = 0, network evidence should not contribute."""
        score_with_net = self.fusion.fuse(0.5, 0.9, 0.0, q_tx=0.0)
        score_chain_only = self.fusion.fuse_chain_only(0.5)
        assert abs(score_with_net - score_chain_only) < 1e-10

    def test_network_increases_score(self):
        """High network score with good quality should increase fused score."""
        chain_only = self.fusion.fuse(0.5, 0.0, 0.0, 0.0)
        with_network = self.fusion.fuse(0.5, 0.8, 0.0, 0.9)
        assert with_network > chain_only

    def test_mixing_penalty_decreases_score(self):
        """Mixing penalty should decrease the fused score."""
        no_mixing = self.fusion.fuse(0.5, 0.5, 0.0, 0.5)
        with_mixing = self.fusion.fuse(0.5, 0.5, 0.8, 0.5)
        assert with_mixing < no_mixing

    def test_chain_only(self):
        """Chain-only score with zero chain should be below 0.5 (due to negative bias)."""
        score = self.fusion.fuse_chain_only(0.0)
        assert score < 0.5  # bias is negative

    def test_high_chain_score(self):
        """High chain score should produce high fused score."""
        score = self.fusion.fuse(1.0, 0.0, 0.0, 0.0)
        assert score > 0.5

    def test_custom_weights(self):
        """Custom weights should be respected."""
        custom = LogisticFusion(weights=FusionWeights(w_chain=1.0, w_net=0.0, w_mixing=0.0, bias=0.0))
        score = custom.fuse(0.0, 1.0, 0.0, 1.0)
        assert abs(score - 0.5) < 1e-10  # sigmoid(0) = 0.5

    def test_decompose(self):
        """Decomposition should return valid components."""
        decomp = self.fusion.decompose(0.5, 0.8, 0.1, 0.9)
        assert "chain_contribution" in decomp
        assert "network_contribution" in decomp
        assert "mixing_penalty_contribution" in decomp
        assert "fused_score" in decomp
        assert "chain_only_score" in decomp
        assert "network_uplift" in decomp
        assert "weights_used" in decomp

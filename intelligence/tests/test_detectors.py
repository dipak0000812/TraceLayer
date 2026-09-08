"""Tests for pattern detectors (peeling chain and CoinJoin)."""

import pytest

from intelligence.detectors.coinjoin import CoinJoinDetector
from intelligence.detectors.peeling_chain import PeelingChainDetector
from intelligence.schemas.request import TransactionInput


def _make_tx(**kwargs) -> TransactionInput:
    defaults = {
        "txid": "a" * 64,
        "amount_btc": 1.0,
        "fee_btc": 0.001,
        "input_count": 1,
        "output_count": 2,
        "input_addresses": ["sbc1_addr_1"],
        "output_addresses": ["sbc1_addr_2", "sbc1_addr_3"],
        "input_amounts": [1.001],
        "output_amounts": [0.5, 0.5],
        "script_type": "P2PKH",
        "network_observations": [],
    }
    defaults.update(kwargs)
    return TransactionInput(**defaults)


class TestPeelingChainDetector:
    def setup_method(self):
        self.detector = PeelingChainDetector()

    def test_classic_peel(self):
        """2 outputs with high value disparity → flagged."""
        tx = _make_tx(
            output_count=2,
            output_amounts=[9.5, 0.5],
            output_addresses=["sbc1_change", "sbc1_peel"],
            input_count=1,
        )
        result = self.detector.detect(tx)
        assert result.score > 0.5
        assert result.flagged is True

    def test_equal_outputs_not_peel(self):
        """2 equal outputs → not flagged."""
        tx = _make_tx(
            output_count=2,
            output_amounts=[0.5, 0.5],
        )
        result = self.detector.detect(tx)
        assert result.score < 0.6
        assert result.flagged is False

    def test_many_outputs_not_peel(self):
        """5 outputs → not a peeling chain."""
        tx = _make_tx(
            output_count=5,
            output_amounts=[0.2, 0.2, 0.2, 0.2, 0.2],
            output_addresses=[f"sbc1_addr_{i}" for i in range(5)],
        )
        result = self.detector.detect(tx)
        assert result.flagged is False

    def test_single_output_not_peel(self):
        """1 output → not a peeling chain."""
        tx = _make_tx(
            output_count=1,
            output_amounts=[1.0],
            output_addresses=["sbc1_addr_1"],
        )
        result = self.detector.detect(tx)
        assert result.flagged is False


class TestCoinJoinDetector:
    def setup_method(self):
        self.detector = CoinJoinDetector()

    def test_classic_coinjoin(self):
        """Many inputs + many equal-denomination outputs → flagged."""
        tx = _make_tx(
            input_count=5,
            output_count=5,
            input_addresses=[f"sbc1_in_{i}" for i in range(5)],
            output_addresses=[f"sbc1_out_{i}" for i in range(5)],
            input_amounts=[0.1] * 5,
            output_amounts=[0.09] * 5,  # Equal denominations
            amount_btc=0.45,
            fee_btc=0.05,
        )
        result = self.detector.detect(tx)
        assert result.score > 0.4
        assert result.flagged is True

    def test_normal_tx_not_coinjoin(self):
        """1 input, 2 outputs → not CoinJoin."""
        tx = _make_tx(
            input_count=1,
            output_count=2,
            output_amounts=[0.5, 0.5],
        )
        result = self.detector.detect(tx)
        assert result.flagged is False

    def test_varied_outputs_weak_signal(self):
        """Many inputs/outputs but no equal denominations → lower score."""
        tx = _make_tx(
            input_count=4,
            output_count=4,
            input_addresses=[f"sbc1_in_{i}" for i in range(4)],
            output_addresses=[f"sbc1_out_{i}" for i in range(4)],
            input_amounts=[0.1, 0.2, 0.3, 0.4],
            output_amounts=[0.05, 0.15, 0.35, 0.45],
            amount_btc=1.0,
            fee_btc=0.0,
        )
        result = self.detector.detect(tx)
        # Should have lower score because outputs are not equal
        assert result.score < 0.5

    def test_two_inputs_not_enough(self):
        """2 inputs, 2 outputs → not enough for CoinJoin."""
        tx = _make_tx(
            input_count=2,
            output_count=2,
            input_addresses=["sbc1_in_1", "sbc1_in_2"],
            output_addresses=["sbc1_out_1", "sbc1_out_2"],
        )
        result = self.detector.detect(tx)
        assert result.flagged is False

"""Tests for feature extraction pipeline."""

import numpy as np
import pytest

from intelligence.features.blockchain_features import extract_blockchain_features
from intelligence.features.network_features import compute_q_tx, extract_network_features
from intelligence.features.pipeline import FeaturePipeline
from intelligence.schemas.request import NetworkObservationInput, TransactionInput


def _make_tx(**kwargs) -> TransactionInput:
    """Helper to build a TransactionInput with sensible defaults."""
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


def _make_obs(**kwargs) -> NetworkObservationInput:
    """Helper to build a NetworkObservationInput."""
    defaults = {
        "observation_id": "obs_001",
        "observed_at": "2025-03-22T16:34:24Z",
        "src_ip": "172.16.2.66",
        "geo_country": "BR",
        "asn": "AS65096",
    }
    defaults.update(kwargs)
    return NetworkObservationInput(**defaults)


class TestBlockchainFeatures:
    def test_basic_extraction(self):
        tx = _make_tx()
        features = extract_blockchain_features(tx)
        assert features.tx_amount == 1.0
        assert features.fee == 0.001
        assert features.input_count == 1
        assert features.output_count == 2
        assert features.input_output_ratio == 0.5

    def test_variance_single_output(self):
        tx = _make_tx(output_amounts=[1.0])
        features = extract_blockchain_features(tx)
        assert features.amount_variance == 0.0

    def test_variance_multiple_outputs(self):
        tx = _make_tx(output_amounts=[0.1, 0.9])
        features = extract_blockchain_features(tx)
        assert features.amount_variance > 0

    def test_fee_rate(self):
        tx = _make_tx(amount_btc=10.0, fee_btc=0.1)
        features = extract_blockchain_features(tx)
        assert abs(features.fee_rate - 0.01) < 1e-6


class TestNetworkFeatures:
    def test_no_observations(self):
        tx = _make_tx(network_observations=[])
        features = extract_network_features(tx)
        assert features.network_observation_count == 0
        assert features.unique_peer_count == 0
        assert features.network_quality == 0.0

    def test_single_observation(self):
        tx = _make_tx(network_observations=[_make_obs()])
        features = extract_network_features(tx)
        assert features.network_observation_count == 1
        assert features.unique_peer_count == 1
        assert features.timing_spread_seconds == 0.0

    def test_multiple_observations(self):
        obs = [
            _make_obs(observation_id="obs_1", src_ip="10.0.0.1", observed_at="2025-03-22T16:34:24Z"),
            _make_obs(observation_id="obs_2", src_ip="10.0.0.2", observed_at="2025-03-22T16:34:30Z"),
            _make_obs(observation_id="obs_3", src_ip="10.0.0.3", observed_at="2025-03-22T16:34:36Z"),
        ]
        tx = _make_tx(network_observations=obs)
        features = extract_network_features(tx)
        assert features.network_observation_count == 3
        assert features.unique_peer_count == 3
        assert features.timing_spread_seconds == 12.0

    def test_duplicate_peers(self):
        obs = [
            _make_obs(observation_id="obs_1", src_ip="10.0.0.1", observed_at="2025-03-22T16:34:24Z"),
            _make_obs(observation_id="obs_2", src_ip="10.0.0.1", observed_at="2025-03-22T16:34:30Z"),
        ]
        tx = _make_tx(network_observations=obs)
        features = extract_network_features(tx)
        assert features.network_observation_count == 2
        assert features.unique_peer_count == 1


class TestQualityMetric:
    def test_zero_observations(self):
        assert compute_q_tx([]) == 0.0

    def test_perfect_quality(self):
        """3+ observations, 0 spread → Q = 1.0"""
        obs = [
            _make_obs(observation_id=f"obs_{i}", observed_at="2025-03-22T16:34:24Z")
            for i in range(5)
        ]
        q = compute_q_tx(obs)
        assert q == 1.0

    def test_partial_coverage(self):
        """1 observation → coverage = 1/3"""
        obs = [_make_obs()]
        q = compute_q_tx(obs)
        assert 0 < q < 1.0

    def test_large_spread(self):
        """Spread > MAX_SPREAD → freshness = 0"""
        obs = [
            _make_obs(observation_id="obs_1", observed_at="2025-03-22T16:00:00Z"),
            _make_obs(observation_id="obs_2", observed_at="2025-03-22T16:10:00Z"),
        ]
        q = compute_q_tx(obs)
        assert q == 0.0  # 600s spread > 120s max


class TestFeaturePipeline:
    def test_output_shape(self):
        tx = _make_tx()
        pipeline = FeaturePipeline()
        features = pipeline.extract(tx)
        arr = features.to_array()
        assert arr.shape == (11,)

    def test_batch_extraction(self):
        txs = [_make_tx(txid=f"{'a' * 63}{i}") for i in range(5)]
        pipeline = FeaturePipeline()
        features_list, matrix = pipeline.extract_batch(txs)
        assert len(features_list) == 5
        assert matrix.shape == (5, 11)

    def test_to_dict_keys(self):
        from intelligence.config import FEATURE_NAMES

        tx = _make_tx()
        pipeline = FeaturePipeline()
        features = pipeline.extract(tx)
        d = features.to_dict()
        assert list(d.keys()) == FEATURE_NAMES

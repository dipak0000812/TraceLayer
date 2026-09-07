"""Tests for the FastAPI /intelligence/score endpoint."""

import numpy as np
import pytest
from fastapi.testclient import TestClient

from intelligence.main import app


@pytest.fixture
def client():
    """Create a test client with the app lifespan."""
    with TestClient(app) as c:
        yield c


class TestHealthEndpoint:
    def test_health_returns_200(self, client):
        response = client.get("/health")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "ok"
        assert "model_loaded" in data
        assert "model_version" in data


class TestScoreEndpoint:
    def test_score_single_transaction(self, client):
        payload = {
            "transactions": [
                {
                    "txid": "a" * 64,
                    "amount_btc": 1.25,
                    "fee_btc": 0.001,
                    "input_count": 2,
                    "output_count": 2,
                    "input_addresses": ["sbc1_addr_1", "sbc1_addr_2"],
                    "output_addresses": ["sbc1_addr_3", "sbc1_addr_4"],
                    "input_amounts": [0.7, 0.551],
                    "output_amounts": [0.625, 0.625],
                    "script_type": "P2PKH",
                    "network_observations": [
                        {
                            "observation_id": "obs_001",
                            "observed_at": "2025-03-22T16:34:24Z",
                            "src_ip": "172.16.2.66",
                            "geo_country": "BR",
                            "asn": "AS65096",
                        }
                    ],
                }
            ]
        }
        response = client.post("/intelligence/score", json=payload)
        assert response.status_code == 200
        data = response.json()
        assert "scores" in data
        assert len(data["scores"]) == 1

        score = data["scores"][0]
        assert score["txid"] == "a" * 64
        assert 0.0 <= score["chain_score"] <= 1.0
        assert 0.0 <= score["network_score"] <= 1.0
        assert 0.0 <= score["fused_score"] <= 1.0
        assert 0.0 <= score["network_quality_q"] <= 1.0
        assert isinstance(score["peeling_chain_flag"], bool)
        assert isinstance(score["mixing_flag"], bool)
        assert isinstance(score["top_features"], list)
        assert isinstance(score["explanation"], str)
        assert score["explainability_method"] == "shap"

    def test_score_no_network_observations(self, client):
        payload = {
            "transactions": [
                {
                    "txid": "b" * 64,
                    "amount_btc": 0.5,
                    "fee_btc": 0.0001,
                    "input_count": 1,
                    "output_count": 1,
                    "input_addresses": ["sbc1_addr_1"],
                    "output_addresses": ["sbc1_addr_2"],
                    "input_amounts": [0.5001],
                    "output_amounts": [0.5],
                    "script_type": "P2WPKH",
                    "network_observations": [],
                }
            ]
        }
        response = client.post("/intelligence/score", json=payload)
        assert response.status_code == 200
        score = response.json()["scores"][0]
        assert score["network_quality_q"] == 0.0
        assert score["network_score"] == 0.0

    def test_score_batch(self, client):
        txs = []
        for i in range(3):
            txs.append({
                "txid": f"{i:064x}",
                "amount_btc": 0.1 * (i + 1),
                "fee_btc": 0.0001,
                "input_count": 1,
                "output_count": 1,
                "input_addresses": [f"sbc1_in_{i}"],
                "output_addresses": [f"sbc1_out_{i}"],
                "input_amounts": [0.1 * (i + 1) + 0.0001],
                "output_amounts": [0.1 * (i + 1)],
                "script_type": "P2PKH",
                "network_observations": [],
            })
        payload = {"transactions": txs}
        response = client.post("/intelligence/score", json=payload)
        assert response.status_code == 200
        assert len(response.json()["scores"]) == 3

    def test_empty_transactions_rejected(self, client):
        payload = {"transactions": []}
        response = client.post("/intelligence/score", json=payload)
        assert response.status_code == 422  # Pydantic validation error

    def test_lime_explainer(self, client):
        payload = {
            "transactions": [
                {
                    "txid": "c" * 64,
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
            ],
            "explainer": "lime",
        }
        response = client.post("/intelligence/score", json=payload)
        assert response.status_code == 200
        score = response.json()["scores"][0]
        assert score["explainability_method"] == "lime"

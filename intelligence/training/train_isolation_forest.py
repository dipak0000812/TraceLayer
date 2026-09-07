"""Train an Isolation Forest model on the Elliptic dataset.

Usage:
    cd TraceLayer
    python -m intelligence.training.train_isolation_forest

This script:
  1. Loads the Elliptic dataset (or falls back to synthetic data)
  2. Maps features to TraceLayer's 11-feature schema
  3. Trains an Isolation Forest
  4. Saves the model to intelligence/models/pretrained/
  5. Prints evaluation metrics against labeled data
"""

from __future__ import annotations

import logging
import sys
from pathlib import Path

import numpy as np

# Ensure project root is on sys.path
PROJECT_ROOT = Path(__file__).resolve().parent.parent.parent
if str(PROJECT_ROOT) not in sys.path:
    sys.path.insert(0, str(PROJECT_ROOT))

from intelligence.config import DEFAULT_IF_CONFIG, FEATURE_NAMES
from intelligence.data.elliptic_loader import EllipticDataset
from intelligence.data.feature_mapper import map_elliptic_to_tracelayer
from intelligence.models.isolation_forest import IsolationForestModel
from intelligence.training.evaluate_model import evaluate_against_labels

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)


def train_on_elliptic() -> Path | None:
    """Train Isolation Forest on Elliptic dataset.

    Returns
    -------
    Path | None
        Path to saved model, or None if training failed.
    """
    dataset = EllipticDataset()

    if not dataset.load():
        logger.error(
            "Cannot train: Elliptic dataset not found. "
            "Download from Kaggle and place in intelligence/data/elliptic/"
        )
        return None

    # --- Get temporal split for train/test ---
    split = dataset.get_temporal_split(train_ratio=0.7)
    if split is None:
        logger.error("Failed to split dataset.")
        return None

    X_train_full, X_test_full, y_train, y_test = split

    # --- Map to TraceLayer's 11-feature schema ---
    X_train = map_elliptic_to_tracelayer(X_train_full)
    X_test = map_elliptic_to_tracelayer(X_test_full)

    logger.info(
        "Training set: %d samples, Test set: %d samples, Features: %d",
        len(X_train), len(X_test), len(FEATURE_NAMES),
    )

    # --- Train ---
    model = IsolationForestModel(config=DEFAULT_IF_CONFIG)
    model.train(X_train.values)

    # --- Evaluate on test set ---
    logger.info("=" * 60)
    logger.info("EVALUATION ON TEST SET")
    logger.info("=" * 60)
    evaluate_against_labels(model, X_test.values, y_test.values)

    # --- Save ---
    saved_path = model.save()
    logger.info("Model saved to: %s", saved_path)

    return saved_path


def train_on_synthetic() -> Path | None:
    """Fallback: train on synthetic seed-42 data.

    This produces a weaker model but allows the system to function
    without the Elliptic dataset.

    Returns
    -------
    Path | None
        Path to saved model.
    """
    from intelligence.data.synthetic_loader import SyntheticDataset
    from intelligence.features.pipeline import FeaturePipeline
    from intelligence.schemas.request import (
        NetworkObservationInput,
        TransactionInput,
    )

    logger.warning(
        "Training on synthetic data (fallback). "
        "For better results, use the Elliptic dataset."
    )

    dataset = SyntheticDataset()
    if not dataset.load():
        logger.error("Synthetic dataset not found either!")
        return None

    # Build TransactionInput objects from synthetic data
    correlated = dataset.get_correlated_data()
    pipeline = FeaturePipeline()

    transactions = []
    for tx_data in correlated:
        # Parse JSON-encoded array fields if they're strings
        input_addrs = tx_data.get("input_addresses", [])
        if isinstance(input_addrs, str):
            import json
            input_addrs = json.loads(input_addrs)

        output_addrs = tx_data.get("output_addresses", [])
        if isinstance(output_addrs, str):
            import json
            output_addrs = json.loads(output_addrs)

        input_amounts = tx_data.get("input_amounts", [])
        if isinstance(input_amounts, str):
            import json
            input_amounts = json.loads(input_amounts)

        output_amounts = tx_data.get("output_amounts", [])
        if isinstance(output_amounts, str):
            import json
            output_amounts = json.loads(output_amounts)

        # Build network observations
        net_obs = []
        for obs in tx_data.get("network_observations", []):
            net_obs.append(
                NetworkObservationInput(
                    observation_id=obs.get("observation_id", ""),
                    observed_at=obs.get("timestamp", ""),
                    src_ip=obs.get("src_ip", ""),
                    geo_country=obs.get("geo_country"),
                    asn=obs.get("asn"),
                )
            )

        tx_input = TransactionInput(
            txid=tx_data["txid"],
            amount_btc=float(sum(float(a) for a in output_amounts)) if output_amounts else 0.0,
            fee_btc=float(tx_data.get("fee", 0.0)),
            input_count=len(input_addrs),
            output_count=len(output_addrs),
            input_addresses=input_addrs,
            output_addresses=output_addrs,
            input_amounts=[float(a) for a in input_amounts],
            output_amounts=[float(a) for a in output_amounts],
            script_type=tx_data.get("script_type", "P2PKH"),
            network_observations=net_obs,
        )
        transactions.append(tx_input)

    # Extract features
    _, X_matrix = pipeline.extract_batch(transactions)
    logger.info("Extracted features for %d synthetic transactions", X_matrix.shape[0])

    # Train
    model = IsolationForestModel(config=DEFAULT_IF_CONFIG)
    model.train(X_matrix)

    # Save
    saved_path = model.save()
    logger.info("Model saved to: %s", saved_path)

    return saved_path


def main():
    """Entry point: try Elliptic first, fall back to synthetic."""
    logger.info("=" * 60)
    logger.info("TraceLayer Isolation Forest Training")
    logger.info("=" * 60)

    # Try Elliptic first
    dataset = EllipticDataset()
    if dataset.is_available:
        result = train_on_elliptic()
    else:
        logger.warning("Elliptic dataset not available. Falling back to synthetic data.")
        result = train_on_synthetic()

    if result:
        logger.info("Training complete! Model at: %s", result)
    else:
        logger.error("Training failed.")
        sys.exit(1)


if __name__ == "__main__":
    main()

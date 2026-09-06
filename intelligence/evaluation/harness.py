"""End-to-end evaluation harness.

This is the ONLY module that reads from data/eval/ground_truth.json.
It runs the full scoring pipeline against the seed-42 dataset and
evaluates results against ground truth labels.

Usage:
    cd TraceLayer
    python -m intelligence.evaluation.harness
"""

from __future__ import annotations

import json
import logging
import sys
from pathlib import Path

import numpy as np

# Ensure project root is on sys.path
PROJECT_ROOT = Path(__file__).resolve().parent.parent.parent
if str(PROJECT_ROOT) not in sys.path:
    sys.path.insert(0, str(PROJECT_ROOT))

from intelligence.config import EVAL_DATA_DIR
from intelligence.data.synthetic_loader import SyntheticDataset
from intelligence.detectors.coinjoin import CoinJoinDetector
from intelligence.detectors.peeling_chain import PeelingChainDetector
from intelligence.evaluation.metrics import precision_at_k, rank_shift_analysis
from intelligence.features.pipeline import FeaturePipeline
from intelligence.fusion.logistic_fusion import LogisticFusion
from intelligence.models.isolation_forest import IsolationForestModel
from intelligence.schemas.request import NetworkObservationInput, TransactionInput

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)


def load_ground_truth() -> list[dict]:
    """Load ground truth from data/eval/ground_truth.json.

    THIS IS THE ONLY PLACE IN THE INTELLIGENCE MODULE THAT READS
    FROM data/eval/. The pipeline code NEVER touches this file.
    """
    gt_path = EVAL_DATA_DIR / "ground_truth.json"
    if not gt_path.exists():
        logger.error("Ground truth not found at %s", gt_path)
        return []

    with open(gt_path, "r", encoding="utf-8") as f:
        return json.load(f)


def build_transaction_inputs(dataset: SyntheticDataset) -> list[TransactionInput]:
    """Convert synthetic dataset to TransactionInput objects."""
    correlated = dataset.get_correlated_data()
    transactions = []

    for tx_data in correlated:
        # Parse JSON-encoded fields
        def _parse_json_field(val):
            if isinstance(val, str):
                return json.loads(val)
            return val if val else []

        input_addrs = _parse_json_field(tx_data.get("input_addresses", []))
        output_addrs = _parse_json_field(tx_data.get("output_addresses", []))
        input_amounts = [float(a) for a in _parse_json_field(tx_data.get("input_amounts", []))]
        output_amounts = [float(a) for a in _parse_json_field(tx_data.get("output_amounts", []))]

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
            amount_btc=sum(output_amounts) if output_amounts else 0.0,
            fee_btc=float(tx_data.get("fee", 0.0)),
            input_count=len(input_addrs),
            output_count=len(output_addrs),
            input_addresses=input_addrs,
            output_addresses=output_addrs,
            input_amounts=input_amounts,
            output_amounts=output_amounts,
            script_type=tx_data.get("script_type", "P2PKH"),
            network_observations=net_obs,
        )
        transactions.append(tx_input)

    return transactions


def run_evaluation():
    """Run full evaluation pipeline."""
    logger.info("=" * 60)
    logger.info("TraceLayer Evaluation Harness")
    logger.info("=" * 60)

    # --- Load ground truth ---
    ground_truth = load_ground_truth()
    if not ground_truth:
        logger.error("Cannot proceed without ground truth.")
        return

    # Build txid → anomaly label mapping
    txid_labels: dict[str, bool] = {}
    txid_scenarios: dict[str, str] = {}
    for scenario in ground_truth:
        for txid in scenario.get("txids", []):
            txid_labels[txid] = scenario.get("anomaly_label", False)
            txid_scenarios[txid] = scenario.get("scenario_type", "UNKNOWN")

    logger.info(
        "Ground truth: %d scenarios, %d labeled txids (%d anomalous)",
        len(ground_truth),
        len(txid_labels),
        sum(txid_labels.values()),
    )

    # --- Load synthetic dataset ---
    dataset = SyntheticDataset()
    if not dataset.load():
        logger.error("Cannot load synthetic dataset.")
        return

    transactions = build_transaction_inputs(dataset)
    logger.info("Built %d TransactionInput objects", len(transactions))

    # --- Initialize components ---
    pipeline = FeaturePipeline()
    model = IsolationForestModel()
    model.load_or_initialize()
    fusion = LogisticFusion()
    peeling_detector = PeelingChainDetector()
    coinjoin_detector = CoinJoinDetector()

    # --- Score all transactions ---
    chain_scores = []
    fused_scores = []
    y_true = []
    txids = []

    for tx in transactions:
        features = pipeline.extract(tx)
        feature_vector = features.to_array()

        chain_score = model.score(feature_vector)

        # Network score
        if features.network_observation_count > 0:
            peer_factor = min(1.0, features.unique_peer_count / 10.0)
            timing_factor = max(0.0, 1.0 - features.timing_spread_seconds / 300.0)
            density = min(1.0, features.network_observation_count / 8.0)
            network_score = min(1.0, peer_factor * 0.4 + timing_factor * 0.35 + density * 0.25)
        else:
            network_score = 0.0

        # Pattern detection
        peeling = peeling_detector.detect(tx)
        coinjoin = coinjoin_detector.detect(tx)
        mixing_penalty = max(peeling.score, coinjoin.score)

        # Fusion
        fused = fusion.fuse(chain_score, network_score, mixing_penalty, features.network_quality)
        chain_only = fusion.fuse_chain_only(chain_score)

        chain_scores.append(chain_only)
        fused_scores.append(fused)
        txids.append(tx.txid)
        y_true.append(1 if txid_labels.get(tx.txid, False) else 0)

    chain_scores_arr = np.array(chain_scores)
    fused_scores_arr = np.array(fused_scores)
    y_true_arr = np.array(y_true)

    # --- Rank shift analysis ---
    logger.info("")
    logger.info("=" * 60)
    logger.info("RANK SHIFT ANALYSIS (Core Hypothesis Test)")
    logger.info("=" * 60)

    shift_results = rank_shift_analysis(chain_scores_arr, fused_scores_arr, txids)
    for key, val in shift_results.items():
        if key != "top_movers":
            logger.info("  %s: %s", key, val)

    logger.info("\n  Top movers:")
    for mover in shift_results.get("top_movers", []):
        logger.info(
            "    %s: rank %d → %d (shift=%+d, chain=%.3f, fused=%.3f)",
            mover.get("entity_id", f"idx_{mover['index']}"),
            mover["chain_rank"],
            mover["fused_rank"],
            mover["rank_shift"],
            mover["chain_score"],
            mover["fused_score"],
        )

    # --- Detection accuracy ---
    if y_true_arr.sum() > 0:
        logger.info("")
        logger.info("=" * 60)
        logger.info("DETECTION ACCURACY")
        logger.info("=" * 60)

        for k in [5, 10, 20]:
            if len(fused_scores_arr) >= k:
                p_chain = precision_at_k(y_true_arr, chain_scores_arr, k)
                p_fused = precision_at_k(y_true_arr, fused_scores_arr, k)
                logger.info(
                    "  Precision@%d: chain=%.3f, fused=%.3f (delta=%+.3f)",
                    k, p_chain, p_fused, p_fused - p_chain,
                )

    # --- Pattern detection accuracy ---
    logger.info("")
    logger.info("=" * 60)
    logger.info("PATTERN DETECTION ACCURACY")
    logger.info("=" * 60)

    peeling_tp, peeling_fp, peeling_fn = 0, 0, 0
    coinjoin_tp, coinjoin_fp, coinjoin_fn = 0, 0, 0

    for tx in transactions:
        scenario = txid_scenarios.get(tx.txid, "UNKNOWN")
        peeling = peeling_detector.detect(tx)
        coinjoin = coinjoin_detector.detect(tx)

        if scenario == "PEELING_CHAIN":
            if peeling.flagged:
                peeling_tp += 1
            else:
                peeling_fn += 1
        elif peeling.flagged:
            peeling_fp += 1

        if scenario == "COINJOIN_LIKE":
            if coinjoin.flagged:
                coinjoin_tp += 1
            else:
                coinjoin_fn += 1
        elif coinjoin.flagged:
            coinjoin_fp += 1

    logger.info("  Peeling Chain: TP=%d, FP=%d, FN=%d", peeling_tp, peeling_fp, peeling_fn)
    logger.info("  CoinJoin:      TP=%d, FP=%d, FN=%d", coinjoin_tp, coinjoin_fp, coinjoin_fn)

    logger.info("")
    logger.info("Evaluation complete.")


if __name__ == "__main__":
    run_evaluation()

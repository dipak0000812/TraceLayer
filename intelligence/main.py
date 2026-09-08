"""TraceLayer Intelligence Worker — FastAPI Application.

Stateless scoring microservice that receives transaction + network
observation data from the Go API, extracts features, runs anomaly
detection, pattern detection, explainability, and logistic fusion,
then returns scored results.

Architecture constraints (from ARCHITECTURE.md):
  - This worker holds NO database connection.
  - It communicates only via HTTP on loopback (port 8000).
  - It is stateless: all state comes from the request payload.
  - The pre-trained Isolation Forest model is loaded at startup.
"""

from __future__ import annotations

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException

from intelligence.config import HOST, MODEL_VERSION, PORT
from intelligence.detectors.coinjoin import CoinJoinDetector
from intelligence.detectors.peeling_chain import PeelingChainDetector
from intelligence.explainability.lime_explainer import LimeExplainerWrapper
from intelligence.explainability.shap_explainer import ShapExplainerWrapper
from intelligence.features.pipeline import FeaturePipeline
from intelligence.fusion.logistic_fusion import LogisticFusion
from intelligence.models.isolation_forest import IsolationForestModel
from intelligence.schemas.request import ScoreRequest, TransactionInput
from intelligence.schemas.response import (
    FeatureAttribution,
    HealthResponse,
    ScoreResponse,
    TransactionScore,
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("intelligence")

# ---------------------------------------------------------------------------
# Global components (initialized at startup)
# ---------------------------------------------------------------------------

isolation_forest: IsolationForestModel | None = None
feature_pipeline: FeaturePipeline | None = None
peeling_detector: PeelingChainDetector | None = None
coinjoin_detector: CoinJoinDetector | None = None
shap_explainer: ShapExplainerWrapper | None = None
lime_explainer: LimeExplainerWrapper | None = None
fusion: LogisticFusion | None = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup / shutdown lifecycle.  Load the model once at boot."""
    global isolation_forest, feature_pipeline, peeling_detector
    global coinjoin_detector, shap_explainer, lime_explainer, fusion

    logger.info("Initializing intelligence worker components...")

    # 1. Feature pipeline
    feature_pipeline = FeaturePipeline()

    # 2. Isolation Forest model
    isolation_forest = IsolationForestModel()
    isolation_forest.load_or_initialize()

    # 3. Pattern detectors
    peeling_detector = PeelingChainDetector()
    coinjoin_detector = CoinJoinDetector()

    # 4. Explainability (SHAP needs the trained model)
    shap_explainer = ShapExplainerWrapper(isolation_forest)
    lime_explainer = LimeExplainerWrapper(isolation_forest)

    # 5. Fusion
    fusion = LogisticFusion()

    logger.info("Intelligence worker ready.")
    yield
    logger.info("Intelligence worker shutting down.")


# ---------------------------------------------------------------------------
# FastAPI app
# ---------------------------------------------------------------------------

app = FastAPI(
    title="TraceLayer Intelligence Worker",
    version=MODEL_VERSION,
    description=(
        "Internal scoring microservice for TraceLayer. "
        "Receives transactions from the Go API, returns anomaly scores, "
        "pattern flags, and SHAP/LIME explanations."
    ),
    lifespan=lifespan,
)


@app.get("/health", response_model=HealthResponse)
async def health():
    """Service health check."""
    return HealthResponse(
        status="ok",
        model_loaded=isolation_forest is not None and isolation_forest.is_loaded,
        model_version=MODEL_VERSION,
    )


@app.post("/intelligence/score", response_model=ScoreResponse)
async def score_transactions(request: ScoreRequest):
    """Score a batch of transactions.

    Pipeline:
      1. Extract 11 features per transaction
      2. Isolation Forest anomaly scoring
      3. Pattern detection (peeling chain + CoinJoin)
      4. Explainability (SHAP or LIME)
      5. Logistic fusion → fused_score
    """
    if isolation_forest is None or not isolation_forest.is_loaded:
        raise HTTPException(
            status_code=503,
            detail="Isolation Forest model not loaded. Service not ready.",
        )

    results: list[TransactionScore] = []

    for tx in request.transactions:
        try:
            score = _score_single_transaction(tx, explainer_method=request.explainer)
            results.append(score)
        except Exception as exc:
            logger.error("Error scoring txid=%s: %s", tx.txid, exc, exc_info=True)
            raise HTTPException(
                status_code=500,
                detail=f"Scoring failed for txid={tx.txid}: {exc}",
            )

    return ScoreResponse(scores=results)


# ---------------------------------------------------------------------------
# Internal scoring logic
# ---------------------------------------------------------------------------


def _score_single_transaction(
    tx: TransactionInput,
    explainer_method: str = "shap",
) -> TransactionScore:
    """Run the full scoring pipeline on a single transaction."""

    # --- 1. Feature extraction ---
    features = feature_pipeline.extract(tx)
    feature_vector = features.to_array()

    # --- 2. Isolation Forest anomaly score (chain features only) ---
    chain_score = isolation_forest.score(feature_vector)

    # --- 3. Network score & quality ---
    network_quality_q = features.network_quality
    # Network score: combination of peer diversity and timing anomalies
    network_score = _compute_network_anomaly_score(features)

    # --- 4. Pattern detection ---
    peeling_result = peeling_detector.detect(tx)
    coinjoin_result = coinjoin_detector.detect(tx)

    mixing_penalty = max(peeling_result.score, coinjoin_result.score)

    # Collect flags
    flags: list[str] = []
    if peeling_result.flagged:
        flags.append("PEELING_CHAIN")
    if coinjoin_result.flagged:
        flags.append("COINJOIN_MIXING")

    # Additional anomaly flags based on features
    flags.extend(_derive_anomaly_flags(features))

    # --- 5. Explainability ---
    if explainer_method == "lime":
        attributions = lime_explainer.explain(feature_vector)
        method_used = "lime"
    else:
        attributions = shap_explainer.explain(feature_vector)
        method_used = "shap"

    top_features = [
        FeatureAttribution(
            feature=attr["feature"],
            shap_value=attr["value"],
            direction="risk_increasing" if attr["value"] > 0 else "risk_decreasing",
        )
        for attr in attributions
    ]

    # --- 6. Logistic fusion ---
    fused = fusion.fuse(
        chain_score=chain_score,
        network_score=network_score,
        mixing_penalty=mixing_penalty,
        q_tx=network_quality_q,
    )

    # --- 7. Heuristic association strength ---
    # Normalized measure of how strongly the network evidence associates
    # this transaction with anomalous propagation patterns.
    heuristic_assoc = _compute_association_strength(
        chain_score, network_score, network_quality_q
    )

    # --- 8. Build explanation string ---
    explanation = _build_explanation(
        tx, chain_score, network_score, network_quality_q,
        fused, flags, top_features,
    )

    return TransactionScore(
        txid=tx.txid,
        chain_score=round(chain_score, 4),
        network_score=round(network_score, 4),
        mixing_penalty=round(mixing_penalty, 4),
        network_quality_q=round(network_quality_q, 4),
        fused_score=round(fused, 4),
        heuristic_association_strength=round(heuristic_assoc, 4),
        peeling_chain_flag=peeling_result.flagged,
        mixing_flag=coinjoin_result.flagged,
        flags=flags,
        top_features=top_features,
        explanation=explanation,
        explainability_method=method_used,
    )


def _compute_network_anomaly_score(features) -> float:
    """Derive a network anomaly score from network features.

    Higher score means more suspicious network propagation:
    - Many unique peers with tight timing → unusual broadcast pattern
    - Few peers with large timing spread → normal propagation
    """
    if features.network_observation_count == 0:
        return 0.0

    # Peer diversity factor: more unique peers = more anomalous
    peer_factor = min(1.0, features.unique_peer_count / 10.0)

    # Timing factor: tight timing spread = rapid propagation = more suspicious
    if features.timing_spread_seconds <= 0:
        timing_factor = 1.0
    else:
        timing_factor = max(0.0, 1.0 - features.timing_spread_seconds / 300.0)

    # Observation density
    density = min(1.0, features.network_observation_count / 8.0)

    return min(1.0, (peer_factor * 0.4 + timing_factor * 0.35 + density * 0.25))


def _compute_association_strength(
    chain_score: float, network_score: float, q_tx: float
) -> float:
    """Normalized heuristic association strength ∈ [0, 1].

    Measures how strongly combined evidence associates an entity with
    suspicious activity.  NOT a calibrated probability.
    """
    if q_tx <= 0:
        return chain_score * 0.5

    combined = (chain_score * 0.4 + network_score * q_tx * 0.6)
    return min(1.0, max(0.0, combined))


def _derive_anomaly_flags(features) -> list[str]:
    """Derive additional anomaly flags from feature values."""
    flags: list[str] = []

    # High fan-out: many outputs relative to inputs
    if features.output_count >= 5 and features.input_output_ratio < 0.5:
        flags.append("HIGH_FAN_OUT")

    # High fee ratio: fee > 5% of transaction amount
    if features.fee_rate > 0.05:
        flags.append("HIGH_FEE_RATIO")

    # Rapid dispersion: many peers saw it with tight timing
    if features.unique_peer_count >= 3 and features.timing_spread_seconds < 15:
        flags.append("RAPID_DISPERSION")

    # Low network evidence
    if (
        features.network_observation_count > 0
        and features.network_quality < 0.3
    ):
        flags.append("LOW_NETWORK_QUALITY")

    return flags


def _build_explanation(
    tx: TransactionInput,
    chain_score: float,
    network_score: float,
    q_tx: float,
    fused_score: float,
    flags: list[str],
    top_features: list[FeatureAttribution],
) -> str:
    """Build an analyst-facing natural language explanation."""
    parts: list[str] = []

    # Chain anomaly summary
    if chain_score >= 0.7:
        parts.append(
            f"Transaction shows highly anomalous on-chain structure "
            f"(chain_score={chain_score:.2f})."
        )
    elif chain_score >= 0.4:
        parts.append(
            f"Transaction has moderately unusual on-chain characteristics "
            f"(chain_score={chain_score:.2f})."
        )
    else:
        parts.append(
            f"On-chain structure appears normal (chain_score={chain_score:.2f})."
        )

    # Network evidence summary
    n_obs = len(tx.network_observations)
    if n_obs == 0:
        parts.append("No network observations available.")
    else:
        n_peers = len({obs.src_ip for obs in tx.network_observations})
        n_countries = len(
            {obs.geo_country for obs in tx.network_observations if obs.geo_country}
        )
        parts.append(
            f"Observed by {n_peers} peer(s) across {n_countries} countr(y/ies) "
            f"(network_score={network_score:.2f}, Q={q_tx:.2f})."
        )

    # Flags
    if flags:
        parts.append(f"Flags: {', '.join(flags)}.")

    # Top feature driver
    if top_features:
        top = top_features[0]
        parts.append(
            f"Primary anomaly driver: {top.feature} "
            f"(SHAP={top.shap_value:+.3f}, {top.direction})."
        )

    return " ".join(parts)


# ---------------------------------------------------------------------------
# Entrypoint
# ---------------------------------------------------------------------------

if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "intelligence.main:app",
        host=HOST,
        port=PORT,
        reload=False,
        log_level="info",
    )

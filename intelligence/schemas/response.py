"""Response schemas for POST /intelligence/score.

The Go API server consumes these responses to persist detection_results
and compute rank shifts.
"""

from __future__ import annotations

from pydantic import BaseModel, Field


class FeatureAttribution(BaseModel):
    """A single SHAP/LIME feature attribution for explainability."""

    feature: str = Field(..., description="Feature name")
    shap_value: float = Field(..., description="SHAP value (or LIME weight)")
    direction: str = Field(
        ...,
        description="'risk_increasing' if positive, 'risk_decreasing' if negative",
    )


class TransactionScore(BaseModel):
    """Scoring result for a single transaction."""

    txid: str = Field(..., description="Transaction identifier")

    # --- Core scores (all 0.0 to 1.0) ---
    chain_score: float = Field(
        ..., ge=0.0, le=1.0, description="Isolation Forest chain anomaly score"
    )
    network_score: float = Field(
        ..., ge=0.0, le=1.0, description="Network propagation anomaly score"
    )
    mixing_penalty: float = Field(
        ..., ge=0.0, le=1.0, description="CoinJoin/mixing penalty score"
    )
    network_quality_q: float = Field(
        ..., ge=0.0, le=1.0, description="Network evidence quality Q(tx)"
    )
    fused_score: float = Field(
        ..., ge=0.0, le=1.0, description="Logistic fusion score (final priority)"
    )
    heuristic_association_strength: float = Field(
        ..., ge=0.0, le=1.0, description="Normalized heuristic association metric"
    )

    # --- Pattern flags ---
    peeling_chain_flag: bool = Field(
        False, description="True if peeling chain pattern detected"
    )
    mixing_flag: bool = Field(
        False, description="True if CoinJoin/mixing pattern detected"
    )
    flags: list[str] = Field(
        default_factory=list,
        description="Additional anomaly flags (e.g., RAPID_DISPERSION, HIGH_FAN_OUT)",
    )

    # --- Explainability ---
    top_features: list[FeatureAttribution] = Field(
        default_factory=list,
        description="Top contributing features with SHAP/LIME values",
    )
    explanation: str = Field(
        "", description="Analyst-facing natural language explanation"
    )
    explainability_method: str = Field(
        "shap", description="Method used: 'shap' or 'lime'"
    )


class ScoreResponse(BaseModel):
    """Top-level response from POST /intelligence/score."""

    scores: list[TransactionScore] = Field(
        ..., description="Scoring results, one per input transaction"
    )


class HealthResponse(BaseModel):
    """Response from GET /health."""

    status: str = Field("ok", description="Service status")
    model_loaded: bool = Field(False, description="Whether the IF model is loaded")
    model_version: str = Field("1.0.0", description="Model version string")

"""Request schemas for POST /intelligence/score.

These models define the contract between the Go API server and the
Python intelligence worker.  The Go server sends transaction data
enriched with correlated network observations; the worker extracts
features, scores, and explains each transaction.
"""

from __future__ import annotations

from pydantic import BaseModel, Field


class NetworkObservationInput(BaseModel):
    """A single P2P network observation attached to a transaction.

    Enriched by the Go pipeline's offline GeoIP component — geo_country
    and asn may be None if enrichment was skipped or the IP could not
    be resolved.
    """

    observation_id: str = Field(..., description="Unique observation identifier")
    observed_at: str = Field(
        ..., description="ISO-8601 UTC timestamp of peer announcement"
    )
    src_ip: str = Field(..., description="Relay peer IP address")
    geo_country: str | None = Field(
        None, description="ISO 3166-1 alpha-2 country code (enriched)"
    )
    asn: str | None = Field(None, description="Autonomous System Number (enriched)")


class TransactionInput(BaseModel):
    """A single transaction with its correlated network observations.

    This is the primary unit of work for the intelligence worker.
    """

    txid: str = Field(..., description="64-hex-char transaction identifier")
    amount_btc: float = Field(..., description="Total output value in BTC")
    fee_btc: float = Field(..., description="Transaction fee in BTC")
    input_count: int = Field(..., ge=1, description="Number of inputs")
    output_count: int = Field(..., ge=1, description="Number of outputs")
    input_addresses: list[str] = Field(
        default_factory=list, description="Input (sender) addresses"
    )
    output_addresses: list[str] = Field(
        default_factory=list, description="Output (receiver) addresses"
    )
    input_amounts: list[float] = Field(
        default_factory=list, description="Input amounts in BTC"
    )
    output_amounts: list[float] = Field(
        default_factory=list, description="Output amounts in BTC"
    )
    script_type: str = Field(
        "P2PKH", description="Script type: P2PKH, P2WPKH, P2SH, P2TR"
    )
    network_observations: list[NetworkObservationInput] = Field(
        default_factory=list,
        description="Correlated network observations for this txid",
    )


class ScoreRequest(BaseModel):
    """Top-level request to POST /intelligence/score.

    Contains a batch of transactions to be scored in one call.
    """

    transactions: list[TransactionInput] = Field(
        ..., min_length=1, description="Transactions to score"
    )
    explainer: str = Field(
        "shap",
        description="Explainability method: 'shap' (default, fast) or 'lime' (slower, on-demand)",
    )

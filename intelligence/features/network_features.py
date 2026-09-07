"""Network (P2P layer) feature extraction.

Extracts 4 features from a transaction's correlated network observations:
  8.  network_observation_count  — how many peers observed this tx
  9.  unique_peer_count          — distinct source IPs
  10. timing_spread_seconds      — max(obs.time) - min(obs.time)
  11. network_quality            — Q(tx) from DATA_CONTRACT.md §5.1

Also computes auxiliary metrics used by pattern detectors and explanation
generation (e.g., unique_countries, unique_asns).
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timezone

from intelligence.config import Q_TX_MAX_SPREAD_SECONDS, Q_TX_N_THRESHOLD
from intelligence.schemas.request import NetworkObservationInput, TransactionInput


@dataclass
class NetworkFeatures:
    """Extracted network-level features for a single transaction."""

    network_observation_count: int
    unique_peer_count: int
    timing_spread_seconds: float
    network_quality: float

    # Auxiliary (not fed to Isolation Forest, used for flags/explanations)
    unique_countries: int = 0
    unique_asns: int = 0


def _parse_timestamp(ts_str: str) -> datetime:
    """Parse an ISO-8601 timestamp string to a datetime object."""
    # Handle both 'Z' suffix and '+00:00' formats
    cleaned = ts_str.replace("Z", "+00:00")
    try:
        return datetime.fromisoformat(cleaned)
    except ValueError:
        # Fallback for edge-case formats
        return datetime.strptime(ts_str, "%Y-%m-%dT%H:%M:%S%z")


def compute_q_tx(observations: list[NetworkObservationInput]) -> float:
    """Compute network evidence quality Q(tx).

    From DATA_CONTRACT.md §5.1:

        Q(tx) = 0.0                                              if N = 0
        Q(tx) = min(1.0, N/3) × max(0.0, 1.0 − Δt_spread/120)  if N > 0

    Parameters
    ----------
    observations : list[NetworkObservationInput]
        Network observations correlated to this transaction.

    Returns
    -------
    float
        Quality metric in [0.0, 1.0].
    """
    n = len(observations)
    if n == 0:
        return 0.0

    timestamps = [_parse_timestamp(obs.observed_at) for obs in observations]
    spread = (max(timestamps) - min(timestamps)).total_seconds()

    coverage = min(1.0, n / Q_TX_N_THRESHOLD)
    freshness = max(0.0, 1.0 - spread / Q_TX_MAX_SPREAD_SECONDS)

    return round(coverage * freshness, 6)


def extract_network_features(tx: TransactionInput) -> NetworkFeatures:
    """Extract 4 network features from a TransactionInput's observations.

    Parameters
    ----------
    tx : TransactionInput
        A single transaction with its correlated network observations.

    Returns
    -------
    NetworkFeatures
        The 4 network features + auxiliary metrics.
    """
    obs_list = tx.network_observations

    if not obs_list:
        return NetworkFeatures(
            network_observation_count=0,
            unique_peer_count=0,
            timing_spread_seconds=0.0,
            network_quality=0.0,
            unique_countries=0,
            unique_asns=0,
        )

    # Count
    observation_count = len(obs_list)

    # Unique peers (by source IP)
    unique_ips = {obs.src_ip for obs in obs_list}
    unique_peer_count = len(unique_ips)

    # Timing spread
    timestamps = [_parse_timestamp(obs.observed_at) for obs in obs_list]
    if len(timestamps) >= 2:
        timing_spread = (max(timestamps) - min(timestamps)).total_seconds()
    else:
        timing_spread = 0.0

    # Q(tx)
    quality = compute_q_tx(obs_list)

    # Auxiliary
    countries = {obs.geo_country for obs in obs_list if obs.geo_country}
    asns = {obs.asn for obs in obs_list if obs.asn}

    return NetworkFeatures(
        network_observation_count=observation_count,
        unique_peer_count=unique_peer_count,
        timing_spread_seconds=round(timing_spread, 3),
        network_quality=quality,
        unique_countries=len(countries),
        unique_asns=len(asns),
    )

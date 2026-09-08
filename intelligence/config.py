"""Configuration for the TraceLayer intelligence worker.

All tunable parameters live here.  Values are loaded from environment
variables when available, falling back to the defaults from
DATA_CONTRACT.md and ARCHITECTURE.md.
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field
from pathlib import Path


# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------

# Root of the intelligence package
INTELLIGENCE_ROOT = Path(__file__).resolve().parent

# Pre-trained model artifacts
PRETRAINED_DIR = INTELLIGENCE_ROOT / "models" / "pretrained"

# Elliptic dataset (manually downloaded)
ELLIPTIC_DATA_DIR = INTELLIGENCE_ROOT / "data" / "elliptic"

# Synthetic seed-42 dataset
SYNTHETIC_DATA_DIR = (
    INTELLIGENCE_ROOT.parent / "data" / "synthetic" / "datasets" / "seed-42" / "data"
)

# Evaluation data (anti-leakage: only used by evaluation harness)
EVAL_DATA_DIR = SYNTHETIC_DATA_DIR / "eval"


# ---------------------------------------------------------------------------
# Feature engineering constants
# ---------------------------------------------------------------------------

# The 11 features the Isolation Forest operates on (in order)
FEATURE_NAMES: list[str] = [
    "tx_amount",
    "fee",
    "input_count",
    "output_count",
    "input_output_ratio",
    "amount_variance",
    "fee_rate",
    "network_observation_count",
    "unique_peer_count",
    "timing_spread_seconds",
    "network_quality",
]

# Number of blockchain-only features (first N in the list)
N_BLOCKCHAIN_FEATURES = 7

# Number of network features (last M in the list)
N_NETWORK_FEATURES = 4


# ---------------------------------------------------------------------------
# Network evidence quality Q(tx) parameters
# (from DATA_CONTRACT.md §5.1)
# ---------------------------------------------------------------------------

Q_TX_N_THRESHOLD: int = 3
Q_TX_MAX_SPREAD_SECONDS: float = 120.0


# ---------------------------------------------------------------------------
# Fusion weights
# (from DATA_CONTRACT.md §5.2)
# ---------------------------------------------------------------------------

@dataclass
class FusionWeights:
    """Logistic sigmoid fusion parameters.

    fused = σ(w_chain·S_chain + w_net·S_net·Q(tx) − w_mixing·S_mixing + bias)
    """

    w_chain: float = 2.0
    w_net: float = 1.5
    w_mixing: float = 1.0
    bias: float = -1.0


DEFAULT_FUSION_WEIGHTS = FusionWeights()


# ---------------------------------------------------------------------------
# Isolation Forest hyperparameters
# ---------------------------------------------------------------------------

@dataclass
class IsolationForestConfig:
    """Hyperparameters for the Isolation Forest model."""

    n_estimators: int = 200
    max_samples: str | int = "auto"
    contamination: float = 0.05
    max_features: float = 1.0
    random_state: int = 42
    n_jobs: int = -1

    # Saved model filename
    model_filename: str = "isolation_forest_v1.joblib"

    @property
    def model_path(self) -> Path:
        return PRETRAINED_DIR / self.model_filename


DEFAULT_IF_CONFIG = IsolationForestConfig()


# ---------------------------------------------------------------------------
# Pattern detector thresholds
# ---------------------------------------------------------------------------

@dataclass
class PeelingChainConfig:
    """Thresholds for peeling chain detection."""

    # Minimum ratio of large output to small output
    min_peel_ratio: float = 5.0
    # Minimum chain length to flag
    min_chain_length: int = 3
    # Score threshold to set the flag
    flag_threshold: float = 0.6


@dataclass
class CoinJoinConfig:
    """Thresholds for CoinJoin/mixing detection."""

    # Minimum inputs for CoinJoin suspicion
    min_inputs: int = 3
    # Minimum outputs for CoinJoin suspicion
    min_outputs: int = 3
    # Tolerance for "equal denomination" matching (1%)
    denomination_tolerance: float = 0.01
    # Minimum fraction of outputs that must be equal-denomination
    min_equal_fraction: float = 0.5
    # Score threshold to set the flag
    flag_threshold: float = 0.5


DEFAULT_PEELING_CONFIG = PeelingChainConfig()
DEFAULT_COINJOIN_CONFIG = CoinJoinConfig()


# ---------------------------------------------------------------------------
# Explainability
# ---------------------------------------------------------------------------

# Maximum number of top features to return in explanations
MAX_TOP_FEATURES: int = 5

# Default explainability method
DEFAULT_EXPLAINER: str = os.getenv("TRACELAYER_EXPLAINER", "shap")


# ---------------------------------------------------------------------------
# Server
# ---------------------------------------------------------------------------

HOST: str = os.getenv("INTELLIGENCE_HOST", "0.0.0.0")
PORT: int = int(os.getenv("INTELLIGENCE_PORT", "8000"))
MODEL_VERSION: str = "1.0.0"

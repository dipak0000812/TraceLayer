"""Feature mapper — maps Elliptic dataset features to TraceLayer's 11-feature schema.

The Elliptic dataset has 166 anonymized features (94 local + 72 aggregated).
Since feature names are anonymized, we map by position/semantics where possible
and use the raw Elliptic features directly for model training.

Strategy:
  - For TRAINING on Elliptic: use all 166 Elliptic features (the model learns
    anomaly patterns from the full feature space).
  - For SCORING at runtime: use our 11 TraceLayer features.
  - The model is trained on Elliptic's feature space and then a separate
    "mapped" model is trained on TraceLayer's 11-feature subset.

This module provides the mapping utilities for the subset approach.
"""

from __future__ import annotations

import logging

import numpy as np
import pandas as pd

from intelligence.config import FEATURE_NAMES

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Elliptic-to-TraceLayer feature mapping
#
# Elliptic local features (anonymized, but published research suggests):
#   local_feature_1:  time step
#   local_feature_2:  number of inputs
#   local_feature_3:  number of outputs
#   local_feature_4:  transaction fee
#   local_feature_5:  output value (total)
#   local_feature_6:  aggregated previous output values
#   ...
#
# We approximate our 11 features from whatever is available.
# Network features (8-11) have NO Elliptic equivalent → set to 0.
# ---------------------------------------------------------------------------

# Maps TraceLayer feature name → Elliptic column name (or None if unmappable)
ELLIPTIC_MAPPING: dict[str, str | None] = {
    "tx_amount": "feature_5",         # total output value (approx)
    "fee": "feature_4",               # transaction fee (approx)
    "input_count": "feature_2",       # number of inputs (approx)
    "output_count": "feature_3",      # number of outputs (approx)
    "input_output_ratio": None,       # derived: input_count / output_count
    "amount_variance": "feature_6",   # approximate: aggregated values
    "fee_rate": None,                 # derived: fee / amount
    "network_observation_count": None, # no equivalent in Elliptic
    "unique_peer_count": None,         # no equivalent in Elliptic
    "timing_spread_seconds": None,     # no equivalent in Elliptic
    "network_quality": None,           # no equivalent in Elliptic
}


def map_elliptic_to_tracelayer(elliptic_df: pd.DataFrame) -> pd.DataFrame:
    """Map Elliptic features to the TraceLayer 11-feature schema.

    Parameters
    ----------
    elliptic_df : pd.DataFrame
        Raw Elliptic features (166 columns: local_feature_1..94, agg_feature_1..72).

    Returns
    -------
    pd.DataFrame
        DataFrame with columns matching FEATURE_NAMES (11 columns).
        Network features are filled with 0.0 (Elliptic has no network layer).
    """
    result = pd.DataFrame(index=elliptic_df.index)

    for feature_name in FEATURE_NAMES:
        elliptic_col = ELLIPTIC_MAPPING.get(feature_name)

        if elliptic_col and elliptic_col in elliptic_df.columns:
            result[feature_name] = elliptic_df[elliptic_col].astype(np.float64)
        elif feature_name == "input_output_ratio":
            # Derived feature
            inp = elliptic_df.get("feature_2", pd.Series(1.0, index=elliptic_df.index))
            out = elliptic_df.get("feature_3", pd.Series(1.0, index=elliptic_df.index))
            result[feature_name] = (inp / out.clip(lower=1)).astype(np.float64)
        elif feature_name == "fee_rate":
            # Derived feature
            fee = elliptic_df.get("feature_4", pd.Series(0.0, index=elliptic_df.index))
            amt = elliptic_df.get("feature_5", pd.Series(1.0, index=elliptic_df.index))
            result[feature_name] = (fee / amt.clip(lower=1e-8)).astype(np.float64)
        else:
            # Network features → 0.0 (Elliptic has no network observations)
            result[feature_name] = 0.0

    logger.info(
        "Mapped %d Elliptic rows to TraceLayer schema (%d features). "
        "Network features set to 0.0.",
        len(result),
        len(FEATURE_NAMES),
    )

    return result


def get_training_features(
    elliptic_df: pd.DataFrame, use_full_elliptic: bool = True
) -> pd.DataFrame:
    """Get the feature matrix for training.

    Parameters
    ----------
    elliptic_df : pd.DataFrame
        Raw Elliptic features.
    use_full_elliptic : bool
        If True, use all 166 Elliptic features (better model, but
        requires re-mapping at scoring time).
        If False, use only the mapped 11 TraceLayer features.

    Returns
    -------
    pd.DataFrame
        Training feature matrix.
    """
    if use_full_elliptic:
        # Use all numeric columns from Elliptic
        numeric_cols = elliptic_df.select_dtypes(include=[np.number]).columns.tolist()
        logger.info("Using full Elliptic feature set: %d features", len(numeric_cols))
        return elliptic_df[numeric_cols].astype(np.float64)
    else:
        return map_elliptic_to_tracelayer(elliptic_df)

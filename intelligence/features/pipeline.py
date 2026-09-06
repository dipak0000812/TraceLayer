"""Feature extraction pipeline — orchestrates blockchain + network features
into a single 11-element feature vector for the Isolation Forest.

The pipeline is the single source of truth for feature ordering.  All
downstream consumers (model, explainability, fusion) reference
``config.FEATURE_NAMES`` to interpret the vector.
"""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np

from intelligence.config import FEATURE_NAMES
from intelligence.features.blockchain_features import (
    BlockchainFeatures,
    extract_blockchain_features,
)
from intelligence.features.network_features import (
    NetworkFeatures,
    extract_network_features,
)
from intelligence.schemas.request import TransactionInput


@dataclass
class CombinedFeatures:
    """All 11 features for a single transaction, in canonical order.

    The ``to_array()`` method returns a 1-D numpy array aligned with
    ``config.FEATURE_NAMES``.
    """

    # Blockchain (7)
    tx_amount: float = 0.0
    fee: float = 0.0
    input_count: int = 0
    output_count: int = 0
    input_output_ratio: float = 0.0
    amount_variance: float = 0.0
    fee_rate: float = 0.0

    # Network (4)
    network_observation_count: int = 0
    unique_peer_count: int = 0
    timing_spread_seconds: float = 0.0
    network_quality: float = 0.0

    # Auxiliary (not in feature vector, used for flags/explanations)
    unique_countries: int = 0
    unique_asns: int = 0

    def to_array(self) -> np.ndarray:
        """Return 1-D numpy array in FEATURE_NAMES order."""
        return np.array(
            [
                self.tx_amount,
                self.fee,
                float(self.input_count),
                float(self.output_count),
                self.input_output_ratio,
                self.amount_variance,
                self.fee_rate,
                float(self.network_observation_count),
                float(self.unique_peer_count),
                self.timing_spread_seconds,
                self.network_quality,
            ],
            dtype=np.float64,
        )

    def to_dict(self) -> dict[str, float]:
        """Return feature dict keyed by FEATURE_NAMES."""
        arr = self.to_array()
        return {name: float(arr[i]) for i, name in enumerate(FEATURE_NAMES)}


class FeaturePipeline:
    """Orchestrates extraction of all 11 features from a TransactionInput."""

    def extract(self, tx: TransactionInput) -> CombinedFeatures:
        """Extract the full feature vector for a single transaction.

        Parameters
        ----------
        tx : TransactionInput
            Raw transaction data with network observations.

        Returns
        -------
        CombinedFeatures
            The 11-element feature set.
        """
        chain: BlockchainFeatures = extract_blockchain_features(tx)
        net: NetworkFeatures = extract_network_features(tx)

        return CombinedFeatures(
            # Blockchain
            tx_amount=chain.tx_amount,
            fee=chain.fee,
            input_count=chain.input_count,
            output_count=chain.output_count,
            input_output_ratio=chain.input_output_ratio,
            amount_variance=chain.amount_variance,
            fee_rate=chain.fee_rate,
            # Network
            network_observation_count=net.network_observation_count,
            unique_peer_count=net.unique_peer_count,
            timing_spread_seconds=net.timing_spread_seconds,
            network_quality=net.network_quality,
            # Auxiliary
            unique_countries=net.unique_countries,
            unique_asns=net.unique_asns,
        )

    def extract_batch(
        self, transactions: list[TransactionInput]
    ) -> tuple[list[CombinedFeatures], np.ndarray]:
        """Extract features for a batch of transactions.

        Parameters
        ----------
        transactions : list[TransactionInput]
            Batch of transactions.

        Returns
        -------
        tuple[list[CombinedFeatures], np.ndarray]
            (list of CombinedFeatures, 2-D numpy array of shape (N, 11))
        """
        features_list = [self.extract(tx) for tx in transactions]
        matrix = np.vstack([f.to_array() for f in features_list])
        return features_list, matrix

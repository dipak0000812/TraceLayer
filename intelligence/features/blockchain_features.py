"""Blockchain (on-chain) feature extraction.

Extracts 7 structural features from a single transaction's blockchain data:
  1. tx_amount       — total output value in BTC
  2. fee             — transaction fee in BTC
  3. input_count     — number of inputs
  4. output_count    — number of outputs
  5. input_output_ratio — input_count / output_count
  6. amount_variance — variance of output amounts (dispersion measure)
  7. fee_rate        — fee as a fraction of total amount
"""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np

from intelligence.schemas.request import TransactionInput


@dataclass
class BlockchainFeatures:
    """Extracted blockchain-level features for a single transaction."""

    tx_amount: float
    fee: float
    input_count: int
    output_count: int
    input_output_ratio: float
    amount_variance: float
    fee_rate: float


def extract_blockchain_features(tx: TransactionInput) -> BlockchainFeatures:
    """Extract 7 blockchain features from a TransactionInput.

    Parameters
    ----------
    tx : TransactionInput
        A single transaction with amounts, addresses, and metadata.

    Returns
    -------
    BlockchainFeatures
        The 7 chain-only features.
    """
    # Total output value
    tx_amount = tx.amount_btc if tx.amount_btc > 0 else sum(tx.output_amounts)

    # Fee
    fee = tx.fee_btc
    if fee <= 0 and tx.input_amounts and tx.output_amounts:
        fee = max(0.0, sum(tx.input_amounts) - sum(tx.output_amounts))

    # Counts
    input_count = tx.input_count if tx.input_count > 0 else len(tx.input_addresses)
    output_count = tx.output_count if tx.output_count > 0 else len(tx.output_addresses)

    # I/O ratio
    input_output_ratio = input_count / max(output_count, 1)

    # Output amount variance (measures dispersion of how value is split)
    if len(tx.output_amounts) >= 2:
        amount_variance = float(np.var(tx.output_amounts, ddof=0))
    elif len(tx.output_amounts) == 1:
        amount_variance = 0.0
    else:
        amount_variance = 0.0

    # Fee rate (fee as fraction of total value)
    fee_rate = fee / max(tx_amount, 1e-8)

    return BlockchainFeatures(
        tx_amount=tx_amount,
        fee=fee,
        input_count=input_count,
        output_count=output_count,
        input_output_ratio=round(input_output_ratio, 6),
        amount_variance=round(amount_variance, 8),
        fee_rate=round(fee_rate, 8),
    )

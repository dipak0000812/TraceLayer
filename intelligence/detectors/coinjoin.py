"""CoinJoin / mixing transaction detector.

CoinJoin is a privacy technique where multiple users combine inputs
into a single transaction to break the common-input-ownership heuristic.

Heuristic signals:
  1. Many inputs (≥ 3) AND many outputs (≥ 3)
  2. Multiple outputs have equal or near-equal denominations
  3. Low address overlap between inputs and outputs
  4. Unusual fee structure (coordinator fee pattern)
  5. High input-output count product (complexity measure)

References:
  - Meiklejohn et al., "A Fistful of Bitcoins" (2013)
  - Schnoering & Vazirgiannis, "Heuristics for CoinJoin Detection" (2023)
"""

from __future__ import annotations

import logging
from collections import Counter

from intelligence.config import DEFAULT_COINJOIN_CONFIG, CoinJoinConfig
from intelligence.detectors.base import BaseDetector, DetectionResult
from intelligence.schemas.request import TransactionInput

logger = logging.getLogger(__name__)


class CoinJoinDetector(BaseDetector):
    """Detect CoinJoin/mixing patterns in a single transaction."""

    def __init__(self, config: CoinJoinConfig | None = None):
        self.config = config or DEFAULT_COINJOIN_CONFIG

    @property
    def name(self) -> str:
        return "coinjoin"

    def detect(self, tx: TransactionInput) -> DetectionResult:
        """Analyze a transaction for CoinJoin/mixing characteristics.

        Parameters
        ----------
        tx : TransactionInput
            Transaction to analyze.

        Returns
        -------
        DetectionResult
            CoinJoin score and flag.
        """
        signals: dict[str, float] = {}
        score = 0.0

        input_count = tx.input_count or len(tx.input_addresses)
        output_count = tx.output_count or len(tx.output_addresses)

        # --- Signal 1: High input AND output count ---
        has_many_inputs = input_count >= self.config.min_inputs
        has_many_outputs = output_count >= self.config.min_outputs

        signals["high_input_count"] = min(1.0, input_count / 5.0)
        signals["high_output_count"] = min(1.0, output_count / 5.0)
        signals["both_high"] = 1.0 if (has_many_inputs and has_many_outputs) else 0.0

        if not (has_many_inputs and has_many_outputs):
            # Quick exit: CoinJoin requires multiple participants
            return DetectionResult(
                detector_name=self.name,
                score=0.0,
                flagged=False,
                reason=(
                    f"Insufficient participants for CoinJoin "
                    f"(inputs={input_count}, outputs={output_count})."
                ),
                signals=signals,
            )

        # --- Signal 2: Equal-denomination outputs ---
        equal_denom_fraction = self._compute_equal_denomination_fraction(
            tx.output_amounts
        )
        signals["equal_denomination_fraction"] = equal_denom_fraction

        # --- Signal 3: Address overlap (low = more CoinJoin-like) ---
        input_addrs = set(tx.input_addresses)
        output_addrs = set(tx.output_addresses)
        if input_addrs and output_addrs:
            overlap = len(input_addrs & output_addrs) / max(
                len(input_addrs | output_addrs), 1
            )
            signals["address_overlap"] = overlap
            signals["low_overlap"] = 1.0 - overlap
        else:
            signals["address_overlap"] = 0.0
            signals["low_overlap"] = 1.0

        # --- Signal 4: Complexity (input × output product) ---
        complexity = input_count * output_count
        signals["complexity"] = min(1.0, complexity / 25.0)

        # --- Signal 5: Fee anomaly ---
        # CoinJoin coordinators sometimes take a fixed fee that creates
        # one output slightly different from the equal denominations
        if tx.fee_btc > 0 and tx.amount_btc > 0:
            fee_ratio = tx.fee_btc / tx.amount_btc
            # CoinJoin fees are often slightly higher than typical
            signals["fee_anomaly"] = min(1.0, fee_ratio / 0.02) if fee_ratio > 0.005 else 0.0
        else:
            signals["fee_anomaly"] = 0.0

        # --- Combine signals ---
        weights = {
            "both_high": 0.20,
            "equal_denomination_fraction": 0.35,
            "low_overlap": 0.15,
            "complexity": 0.15,
            "fee_anomaly": 0.15,
        }

        score = sum(signals.get(signal, 0.0) * weight for signal, weight in weights.items())
        score = min(1.0, max(0.0, score))

        flagged = score >= self.config.flag_threshold

        reason = ""
        if flagged:
            reason = (
                f"Transaction exhibits CoinJoin/mixing characteristics: "
                f"{input_count} inputs, {output_count} outputs, "
                f"{equal_denom_fraction:.0%} equal-denomination outputs. "
                f"Consistent with multi-party mixing protocol."
            )
        else:
            reason = "No significant CoinJoin/mixing pattern detected."

        return DetectionResult(
            detector_name=self.name,
            score=round(score, 4),
            flagged=flagged,
            reason=reason,
            signals=signals,
        )

    def _compute_equal_denomination_fraction(
        self, output_amounts: list[float]
    ) -> float:
        """Compute the fraction of outputs that share a common denomination.

        CoinJoin transactions often have multiple outputs of the exact
        same (or very close) value.

        Parameters
        ----------
        output_amounts : list[float]
            Output amounts in BTC.

        Returns
        -------
        float
            Fraction of outputs in the largest equal-denomination group.
        """
        if len(output_amounts) < 2:
            return 0.0

        tolerance = self.config.denomination_tolerance

        # Group amounts that are within `tolerance` fraction of each other.
        # We round each amount to a precision determined by the tolerance,
        # so amounts within ±1% of a reference value land in the same bucket.
        buckets: list[float] = []
        for amt in output_amounts:
            if amt > 0:
                # Round to nearest (amt * tolerance) granularity
                # e.g., 0.05 with tolerance=0.01 → granularity=0.0005
                #        round(0.05 / 0.0005) * 0.0005 = 0.05
                # But 0.15 → granularity=0.0015 → round(0.15/0.0015)*0.0015 = 0.15
                # This groups values relative to their magnitude.
                #
                # Instead, use a fixed precision: round to N significant figures
                # so that 0.05 and 0.15 are clearly different buckets.
                import math
                sig_figs = max(1, -int(math.floor(math.log10(abs(amt) * tolerance + 1e-15))))
                bucket = round(amt, sig_figs)
                buckets.append(bucket)

        if not buckets:
            return 0.0

        # Find the largest group of equal-denomination outputs
        from collections import Counter
        counts = Counter(buckets)
        max_group_size = max(counts.values())

        if max_group_size < 2:
            return 0.0

        return max_group_size / len(output_amounts)

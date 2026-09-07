"""Peeling chain detector.

A peeling chain is a laundering technique where:
  1. A transaction has exactly 2 outputs
  2. One output is much larger than the other (the "change")
  3. The small output ("peel") goes to a destination (e.g., exchange)
  4. This pattern repeats across multiple sequential transactions

Heuristic signals:
  - Exactly 2 outputs
  - Large value disparity between outputs (ratio > 5:1)
  - Low output count relative to input count
  - Output addresses are distinct (no self-spend)

The detector scores a single transaction. The Go pipeline can chain
results across transactions to detect multi-hop peeling chains.
"""

from __future__ import annotations

import logging

from intelligence.config import DEFAULT_PEELING_CONFIG, PeelingChainConfig
from intelligence.detectors.base import BaseDetector, DetectionResult
from intelligence.schemas.request import TransactionInput

logger = logging.getLogger(__name__)


class PeelingChainDetector(BaseDetector):
    """Detect peeling chain patterns in a single transaction."""

    def __init__(self, config: PeelingChainConfig | None = None):
        self.config = config or DEFAULT_PEELING_CONFIG

    @property
    def name(self) -> str:
        return "peeling_chain"

    def detect(self, tx: TransactionInput) -> DetectionResult:
        """Analyze a transaction for peeling chain characteristics.

        Parameters
        ----------
        tx : TransactionInput
            Transaction to analyze.

        Returns
        -------
        DetectionResult
            Peeling chain score and flag.
        """
        signals: dict[str, float] = {}
        score = 0.0

        # --- Signal 1: Exactly 2 outputs (classic peel structure) ---
        # Peeling chains typically have exactly 2 outputs:
        # one small "peel" and one large "change"
        has_two_outputs = tx.output_count == 2 or len(tx.output_amounts) == 2
        signals["has_two_outputs"] = 1.0 if has_two_outputs else 0.0

        if not has_two_outputs:
            # Not a classic peel if output count != 2
            # Still check for near-peel patterns (2-3 outputs)
            if tx.output_count <= 3 and len(tx.output_amounts) >= 2:
                signals["near_peel_structure"] = 0.3
            else:
                return DetectionResult(
                    detector_name=self.name,
                    score=0.0,
                    flagged=False,
                    reason="Output count not consistent with peeling chain.",
                    signals=signals,
                )

        # --- Signal 2: Value disparity between outputs ---
        output_amounts = tx.output_amounts
        if len(output_amounts) >= 2:
            sorted_amounts = sorted(output_amounts, reverse=True)
            largest = sorted_amounts[0]
            smallest = sorted_amounts[-1]

            if smallest > 0:
                peel_ratio = largest / smallest
            else:
                peel_ratio = float("inf")

            signals["peel_ratio"] = min(peel_ratio, 100.0)

            # High ratio = strong peeling signal
            if peel_ratio >= self.config.min_peel_ratio:
                signals["high_disparity"] = 1.0
            else:
                signals["high_disparity"] = min(1.0, peel_ratio / self.config.min_peel_ratio)
        else:
            signals["peel_ratio"] = 0.0
            signals["high_disparity"] = 0.0

        # --- Signal 3: Output address uniqueness ---
        # In a peel, both outputs typically go to different, new addresses
        output_addrs = set(tx.output_addresses)
        input_addrs = set(tx.input_addresses)
        addr_overlap = len(output_addrs & input_addrs)

        signals["output_addr_unique"] = 1.0 if addr_overlap == 0 else 0.0

        # --- Signal 4: Single or few inputs ---
        # Peeling chains typically have 1 input (the change from previous hop)
        signals["single_input"] = 1.0 if tx.input_count == 1 else max(0.0, 1.0 - (tx.input_count - 1) * 0.3)

        # --- Combine signals into a score ---
        # Disparity is a GATE: without significant value disparity,
        # other signals alone cannot flag a peeling chain.
        disparity_weight = signals.get("high_disparity", 0.0)

        structural_score = (
            signals.get("has_two_outputs", 0.0) * 0.35
            + signals.get("output_addr_unique", 0.0) * 0.25
            + signals.get("single_input", 0.0) * 0.40
        )

        # Multiply structural signals by disparity — if disparity is low,
        # the entire score collapses regardless of structure.
        score = structural_score * disparity_weight
        score = min(1.0, max(0.0, score))

        flagged = score >= self.config.flag_threshold

        reason = ""
        if flagged:
            reason = (
                f"Transaction exhibits peeling chain characteristics: "
                f"2 outputs with {signals.get('peel_ratio', 0):.1f}:1 value ratio, "
                f"consistent with value-splitting laundering pattern."
            )
        else:
            reason = "No significant peeling chain pattern detected."

        return DetectionResult(
            detector_name=self.name,
            score=round(score, 4),
            flagged=flagged,
            reason=reason,
            signals=signals,
        )

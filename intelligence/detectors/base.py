"""Abstract base class for pattern detectors.

All structural detectors (peeling chain, CoinJoin, etc.) implement
this interface so the scoring pipeline can call them uniformly.
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field

from intelligence.schemas.request import TransactionInput


@dataclass
class DetectionResult:
    """Result from a single pattern detector."""

    # Name of the detector (e.g., "peeling_chain", "coinjoin")
    detector_name: str = ""

    # Confidence score for this pattern, 0.0 to 1.0
    score: float = 0.0

    # Whether the score exceeds the flag threshold
    flagged: bool = False

    # Human-readable explanation of the detection
    reason: str = ""

    # Sub-signals that contributed to the score
    signals: dict[str, float] = field(default_factory=dict)


class BaseDetector(ABC):
    """Abstract base for all pattern detectors."""

    @property
    @abstractmethod
    def name(self) -> str:
        """Detector name (used in flags and logging)."""
        ...

    @abstractmethod
    def detect(self, tx: TransactionInput) -> DetectionResult:
        """Analyze a transaction for this pattern.

        Parameters
        ----------
        tx : TransactionInput
            A single transaction with its network observations.

        Returns
        -------
        DetectionResult
            Score, flag, and explanation.
        """
        ...

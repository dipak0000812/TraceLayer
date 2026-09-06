"""Synthetic dataset loader.

Loads the TraceLayer seed-42 synthetic dataset for integration testing
and fallback training (if Elliptic is not available).
"""

from __future__ import annotations

import csv
import json
import logging
from pathlib import Path

import numpy as np
import pandas as pd

from intelligence.config import SYNTHETIC_DATA_DIR

logger = logging.getLogger(__name__)


class SyntheticDataset:
    """Load the TraceLayer seed-42 synthetic dataset."""

    def __init__(self, data_dir: Path | None = None):
        self.data_dir = data_dir or SYNTHETIC_DATA_DIR
        self.transactions: list[dict] = []
        self.network_observations: list[dict] = []
        self._loaded = False

    @property
    def is_available(self) -> bool:
        """Check if synthetic data files exist."""
        raw_dir = self.data_dir / "raw"
        return (
            (raw_dir / "transactions.csv").exists()
            or (raw_dir / "transactions.json").exists()
        )

    def load(self) -> bool:
        """Load synthetic transactions and network observations.

        Tries JSON first (easier to parse), falls back to CSV.

        Returns
        -------
        bool
            True if loaded successfully.
        """
        raw_dir = self.data_dir / "raw"

        # --- Transactions ---
        tx_json = raw_dir / "transactions.json"
        tx_csv = raw_dir / "transactions.csv"

        if tx_json.exists():
            with open(tx_json, "r", encoding="utf-8") as f:
                self.transactions = json.load(f)
            logger.info("Loaded %d synthetic transactions (JSON)", len(self.transactions))
        elif tx_csv.exists():
            self.transactions = self._load_csv(tx_csv)
            logger.info("Loaded %d synthetic transactions (CSV)", len(self.transactions))
        else:
            logger.warning("No synthetic transaction files found at %s", raw_dir)
            return False

        # --- Network observations ---
        obs_json = raw_dir / "network_observations.json"
        obs_csv = raw_dir / "network_observations.csv"

        if obs_json.exists():
            with open(obs_json, "r", encoding="utf-8") as f:
                self.network_observations = json.load(f)
            logger.info(
                "Loaded %d synthetic network observations (JSON)",
                len(self.network_observations),
            )
        elif obs_csv.exists():
            self.network_observations = self._load_csv(obs_csv)
            logger.info(
                "Loaded %d synthetic network observations (CSV)",
                len(self.network_observations),
            )

        self._loaded = True
        return True

    def get_correlated_data(self) -> list[dict]:
        """Return transactions with their correlated network observations.

        Returns
        -------
        list[dict]
            Each dict has all transaction fields plus a
            ``network_observations`` key containing a list of observation dicts.
        """
        if not self._loaded:
            return []

        # Index observations by txid
        obs_by_txid: dict[str, list[dict]] = {}
        for obs in self.network_observations:
            txid = obs.get("txid", obs.get("observed_txid", ""))
            obs_by_txid.setdefault(txid, []).append(obs)

        result = []
        for tx in self.transactions:
            txid = tx["txid"]
            correlated = {
                **tx,
                "network_observations": obs_by_txid.get(txid, []),
            }
            result.append(correlated)

        return result

    @staticmethod
    def _load_csv(path: Path) -> list[dict]:
        """Load a CSV file into a list of dicts."""
        rows = []
        with open(path, "r", encoding="utf-8") as f:
            reader = csv.DictReader(f)
            for row in reader:
                rows.append(dict(row))
        return rows

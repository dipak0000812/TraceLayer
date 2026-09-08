"""Elliptic dataset loader.

Loads the three CSV files from the manually-downloaded Elliptic dataset:
  - elliptic_txs_features.csv   (203,769 × 167 columns: txid + 166 features)
  - elliptic_txs_classes.csv    (203,769 × 2:  txid, class)
  - elliptic_txs_edgelist.csv   (234,355 × 2:  txid1, txid2)

Classes: 1 = illicit, 2 = licit, "unknown" = unlabeled
"""

from __future__ import annotations

import logging
from pathlib import Path

import numpy as np
import pandas as pd

from intelligence.config import ELLIPTIC_DATA_DIR

logger = logging.getLogger(__name__)


class EllipticDataset:
    """Load and provide access to the Elliptic Bitcoin dataset."""

    def __init__(self, data_dir: Path | None = None):
        self.data_dir = data_dir or ELLIPTIC_DATA_DIR
        self.features_df: pd.DataFrame | None = None
        self.classes_df: pd.DataFrame | None = None
        self.edges_df: pd.DataFrame | None = None
        self._loaded = False

    @property
    def is_available(self) -> bool:
        """Check if the Elliptic dataset files exist on disk."""
        required_files = [
            self.data_dir / "elliptic_txs_features.csv",
            self.data_dir / "elliptic_txs_classes.csv",
        ]
        return all(f.exists() for f in required_files)

    def load(self) -> bool:
        """Load the dataset from CSV files.

        Returns
        -------
        bool
            True if loaded successfully, False if files not found.
        """
        if not self.is_available:
            logger.warning(
                "Elliptic dataset not found at %s. "
                "Download from https://www.kaggle.com/datasets/ellipticco/elliptic-data-set "
                "and place the CSVs in intelligence/data/elliptic/",
                self.data_dir,
            )
            return False

        logger.info("Loading Elliptic dataset from %s ...", self.data_dir)

        # --- Features ---
        # Column 0 = txid (int), Column 1 = time step, Columns 2-166 = features
        features_path = self.data_dir / "elliptic_txs_features.csv"
        self.features_df = pd.read_csv(features_path, header=None)
        # Column 0 = txid, Columns 1-166 = features
        # The first feature (col 1) is the time step
        n_cols = len(self.features_df.columns)
        self.features_df.columns = (
            ["txid"]
            + [f"feature_{i}" for i in range(1, n_cols)]
        )
        # Rename feature_1 to time_step for convenience
        self.features_df.rename(columns={"feature_1": "time_step"}, inplace=True)
        self.features_df["txid"] = self.features_df["txid"].astype(str)
        logger.info(
            "  Features: %d transactions × %d columns",
            len(self.features_df),
            len(self.features_df.columns),
        )

        # --- Classes ---
        classes_path = self.data_dir / "elliptic_txs_classes.csv"
        self.classes_df = pd.read_csv(classes_path)
        self.classes_df.columns = ["txid", "class"]
        self.classes_df["txid"] = self.classes_df["txid"].astype(str)
        logger.info("  Classes: %d records", len(self.classes_df))

        # --- Edges (optional) ---
        edges_path = self.data_dir / "elliptic_txs_edgelist.csv"
        if edges_path.exists():
            self.edges_df = pd.read_csv(edges_path)
            self.edges_df.columns = ["txid1", "txid2"]
            logger.info("  Edges: %d records", len(self.edges_df))

        self._loaded = True
        return True

    def get_labeled_data(
        self,
    ) -> tuple[pd.DataFrame, pd.Series] | None:
        """Return features and binary labels for labeled transactions only.

        Returns
        -------
        tuple[pd.DataFrame, pd.Series] | None
            (X, y) where y: 1=illicit (anomaly), 0=licit (normal).
            Returns None if dataset is not loaded.
        """
        if not self._loaded:
            return None

        # Merge features with classes
        merged = self.features_df.merge(self.classes_df, on="txid")

        # Keep only labeled rows (drop "unknown")
        labeled = merged[merged["class"].isin(["1", "2", 1, 2])].copy()

        # Convert: 1 (illicit) → 1 (anomaly), 2 (licit) → 0 (normal)
        labeled["label"] = labeled["class"].astype(int).map({1: 1, 2: 0})

        # Feature columns (drop txid, time_step, class, label)
        feature_cols = [
            c for c in labeled.columns
            if c not in ("txid", "time_step", "class", "label")
        ]
        X = labeled[feature_cols].astype(np.float64)
        y = labeled["label"]

        logger.info(
            "Labeled data: %d transactions (%d illicit, %d licit)",
            len(y),
            y.sum(),
            len(y) - y.sum(),
        )

        return X, y

    def get_all_features(self) -> pd.DataFrame | None:
        """Return all features (including unlabeled) for unsupervised training.

        Returns
        -------
        pd.DataFrame | None
            Feature matrix (203K × 166).  None if not loaded.
        """
        if not self._loaded:
            return None

        feature_cols = [
            c for c in self.features_df.columns
            if c not in ("txid", "time_step")
        ]
        return self.features_df[feature_cols].astype(np.float64)

    def get_temporal_split(
        self, train_ratio: float = 0.7,
    ) -> tuple[pd.DataFrame, pd.DataFrame, pd.Series, pd.Series] | None:
        """Split labeled data temporally (by time_step) to avoid leakage.

        Parameters
        ----------
        train_ratio : float
            Fraction of time steps for training (default 0.7 = first 70%).

        Returns
        -------
        tuple | None
            (X_train, X_test, y_train, y_test) or None.
        """
        if not self._loaded:
            return None

        merged = self.features_df.merge(self.classes_df, on="txid")
        labeled = merged[merged["class"].isin(["1", "2", 1, 2])].copy()
        labeled["label"] = labeled["class"].astype(int).map({1: 1, 2: 0})

        time_steps = sorted(labeled["time_step"].unique())
        split_idx = int(len(time_steps) * train_ratio)
        train_steps = set(time_steps[:split_idx])

        train_mask = labeled["time_step"].isin(train_steps)

        feature_cols = [
            c for c in labeled.columns
            if c not in ("txid", "time_step", "class", "label")
        ]

        X_train = labeled.loc[train_mask, feature_cols].astype(np.float64)
        X_test = labeled.loc[~train_mask, feature_cols].astype(np.float64)
        y_train = labeled.loc[train_mask, "label"]
        y_test = labeled.loc[~train_mask, "label"]

        logger.info(
            "Temporal split: train=%d (steps %d-%d), test=%d (steps %d-%d)",
            len(X_train), min(train_steps), max(train_steps),
            len(X_test), min(time_steps[split_idx:]), max(time_steps[split_idx:]),
        )

        return X_train, X_test, y_train, y_test

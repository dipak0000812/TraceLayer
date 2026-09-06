"""Tests for SHAP and LIME explainability wrappers."""

import numpy as np
import pytest

from intelligence.config import FEATURE_NAMES
from intelligence.models.isolation_forest import IsolationForestModel


class TestShapExplainer:
    def setup_method(self):
        """Train a small model for testing."""
        self.model = IsolationForestModel()
        # Train on random data
        rng = np.random.RandomState(42)
        X_train = rng.randn(100, len(FEATURE_NAMES))
        self.model.train(X_train)

    def test_shap_returns_attributions(self):
        from intelligence.explainability.shap_explainer import ShapExplainerWrapper

        explainer = ShapExplainerWrapper(self.model)
        feature_vector = np.random.RandomState(42).randn(len(FEATURE_NAMES))
        attributions = explainer.explain(feature_vector)

        assert isinstance(attributions, list)
        assert len(attributions) > 0
        assert len(attributions) <= 5  # MAX_TOP_FEATURES

        for attr in attributions:
            assert "feature" in attr
            assert "value" in attr
            assert "direction" in attr
            assert attr["direction"] in ("risk_increasing", "risk_decreasing")

    def test_shap_feature_names_valid(self):
        from intelligence.explainability.shap_explainer import ShapExplainerWrapper

        explainer = ShapExplainerWrapper(self.model)
        feature_vector = np.random.RandomState(42).randn(len(FEATURE_NAMES))
        attributions = explainer.explain(feature_vector)

        for attr in attributions:
            assert attr["feature"] in FEATURE_NAMES


class TestLimeExplainer:
    def setup_method(self):
        self.model = IsolationForestModel()
        rng = np.random.RandomState(42)
        X_train = rng.randn(100, len(FEATURE_NAMES))
        self.model.train(X_train)

    def test_lime_returns_attributions(self):
        from intelligence.explainability.lime_explainer import LimeExplainerWrapper

        explainer = LimeExplainerWrapper(self.model)
        feature_vector = np.random.RandomState(42).randn(len(FEATURE_NAMES))
        attributions = explainer.explain(feature_vector)

        assert isinstance(attributions, list)
        assert len(attributions) > 0

        for attr in attributions:
            assert "feature" in attr
            assert "value" in attr
            assert "direction" in attr

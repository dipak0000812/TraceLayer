// internal/domain/lead_test.go
package domain

import (
	"errors"
	"testing"
)

func validLead() ForensicLead {
	return ForensicLead{
		LeadID:                       "lead_1a2b3c4d",
		EntityID:                     "ent_9f8e7d6c",
		PrimaryTXID:                  "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
		ChainOnlyScore:               0.42,
		NetworkScore:                 0.88,
		FusedScore:                   0.79,
		ChainOnlyRank:                8,
		FusedRank:                    2,
		RankShift:                    6, // 8 - 2
		NetworkEvidenceQuality:       0.94,
		HeuristicAssociationStrength: 0.81,
		AnomalyFlags:                 []string{"RAPID_DISPERSION", "HIGH_FAN_OUT"},
		Explanation:                  "example",
	}
}

func TestForensicLead_Validate_Valid(t *testing.T) {
	if err := validLead().Validate(); err != nil {
		t.Fatalf("expected valid lead, got err: %v", err)
	}
}

func TestForensicLead_Validate_RankShiftMismatch(t *testing.T) {
	l := validLead()
	l.RankShift = 999
	if err := l.Validate(); !errors.Is(err, ErrRankShiftMismatch) {
		t.Fatalf("expected ErrRankShiftMismatch, got: %v", err)
	}
}

func TestForensicLead_Validate_ScoreOutOfRange(t *testing.T) {
	l := validLead()
	l.FusedScore = 1.5
	if err := l.Validate(); !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got: %v", err)
	}
}
// internal/domain/lead.go
package domain

import (
	"errors"
	"fmt"
)

// ForensicLead is DATA_CONTRACT.md section 5.3. Every score is a heuristic
// priority signal for human analyst review — see API_CONTRACT.md section 1.4
// for exact semantics. None of them are calibrated probabilities.
type ForensicLead struct {
	LeadID                       string
	EntityID                     string
	PrimaryTXID                  string
	ChainOnlyScore               float64 // [0,1]; Isolation Forest output, NOT a probability
	NetworkScore                 float64 // [0,1]; standalone network anomaly signal
	FusedScore                   float64 // [0,1]; sigmoid fusion output, NOT a probability of guilt
	ChainOnlyRank                int     // 1 = most anomalous by ChainOnlyScore
	FusedRank                    int     // 1 = highest priority by FusedScore
	RankShift                    int     // ChainOnlyRank - FusedRank; positive = elevated by network evidence
	NetworkEvidenceQuality       float64 // Q(tx) in [0,1], DATA_CONTRACT.md section 5.1
	HeuristicAssociationStrength float64 // [0,1]; NOT an identity probability
	AnomalyFlags                 []string
	Explanation                  string
}

var (
	ErrScoreOutOfRange   = errors.New("domain: score must be in [0,1]")
	ErrInvalidRank       = errors.New("domain: rank must be >= 1")
	ErrRankShiftMismatch = errors.New("domain: rank_shift does not equal chain_only_rank - fused_rank")
)

// Validate does NOT check PrimaryTXID's business existence — only its
// format — and does not recompute scores. It only enforces the arithmetic
// and range invariants the contract states as facts about a valid lead.
func (l ForensicLead) Validate() error {
	if !ValidTXID(l.PrimaryTXID) {
		return fmt.Errorf("%w: %q", ErrInvalidTXID, l.PrimaryTXID)
	}
	scores := map[string]float64{
		"chain_only_score":               l.ChainOnlyScore,
		"network_score":                  l.NetworkScore,
		"fused_score":                    l.FusedScore,
		"network_evidence_quality":       l.NetworkEvidenceQuality,
		"heuristic_association_strength": l.HeuristicAssociationStrength,
	}
	for name, v := range scores {
		if v < 0 || v > 1 {
			return fmt.Errorf("%w: %s = %f", ErrScoreOutOfRange, name, v)
		}
	}
	if l.ChainOnlyRank < 1 || l.FusedRank < 1 {
		return ErrInvalidRank
	}
	if l.RankShift != l.ChainOnlyRank-l.FusedRank {
		return ErrRankShiftMismatch
	}
	return nil
}
package ranking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dipak0000812/TraceLayer/internal/evidence"
	"github.com/dipak0000812/TraceLayer/internal/intelligence"
)

type candidate struct {
	EntityID, TXID                       string
	ChainScore, NetworkScore, FusedScore float64
	AssociationStrength                  float64
	NetworkEvidenceQuality               float64
	OutputCount                          int
	Flags                                []string
	Explanation                          string
}

// Run assembles forensic leads using the demo chain fan-out heuristic and
// correlated network evidence quality until the trained models are integrated.
func Run(ctx context.Context, pool *pgxpool.Pool, intelClient *intelligence.Client) (int, error) {
	entities, err := loadEntities(ctx, pool)
	if err != nil {
		return 0, fmt.Errorf("ranking: loading entities: %w", err)
	}
	txByAddress, err := loadTxByInputAddress(ctx, pool)
	if err != nil {
		return 0, fmt.Errorf("ranking: loading transactions: %w", err)
	}

	var candidates []candidate

	for _, e := range entities {
		txid := ""
		for _, addr := range e.MemberAddresses {
			if t, ok := txByAddress[addr]; ok {
				txid = t
				break
			}
		}
		if txid == "" {
			continue
		}
		ev, err := evidence.GetEvidence(ctx, pool, txid)
		if err != nil {
			continue
		}

		outputCount := len(ev.Transaction.OutputAddresses)
		qtx := ev.NetworkQuality
		chainScore, networkScore, fused := 0.0, qtx, 0.0
		associationStrength := qtx
		flags := []string{}
		explanation := ""

		if intelClient != nil {
			resp, err := intelClient.Score(ctx, intelligence.ScoreRequest{
				Transactions: []intelligence.TransactionFeatures{intelligence.BuildFeatures(ev)},
			})
			if err == nil && len(resp.Scores) == 1 {
				s := resp.Scores[0]
				chainScore, networkScore, fused = s.ChainScore, s.NetworkScore, s.FusedScore
				associationStrength = s.HeuristicAssociationStrength
				flags = append(flags, "INTELLIGENCE_WORKER")
				flags = append(flags, s.Flags...)
				explanation = s.Explanation
			} else if err != nil {
				flags = append(flags, workerFailureFlag(err))
			}
		} else {
			flags = append(flags, "INTELLIGENCE_WORKER_NOT_CONFIGURED")
		}
		if explanation == "" {
			chainScore = math.Min(1.0, float64(outputCount-1)*0.2)
			fused = sigmoid(2.0*chainScore + 1.5*networkScore - 1.0)
			flags = append(flags, "HEURISTIC_FALLBACK")
			explanation = fmt.Sprintf("Entity %s: %d-output transaction, network evidence quality %.2f (ML worker unavailable — heuristic fallback used).", e.EntityID, outputCount, qtx)
		}

		candidates = append(candidates, candidate{
			EntityID: e.EntityID, TXID: txid, OutputCount: outputCount,
			ChainScore: chainScore, NetworkScore: networkScore, FusedScore: fused,
			AssociationStrength: associationStrength, NetworkEvidenceQuality: qtx,
			Flags: flags, Explanation: explanation,
		})
	}

	chainRank := rankBy(candidates, func(c candidate) float64 { return c.ChainScore })
	fusedRank := rankBy(candidates, func(c candidate) float64 { return c.FusedScore })

	inserted := 0
	for _, c := range candidates {
		cr, fr := chainRank[c.EntityID], fusedRank[c.EntityID]
		flagsJSON, _ := json.Marshal(c.Flags)

		_, err := pool.Exec(ctx, `
			INSERT INTO forensic_leads (
				lead_id, entity_id, primary_txid, chain_only_score, network_score, fused_score,
				chain_only_rank, fused_rank, rank_shift, network_evidence_quality,
				heuristic_association_strength, anomaly_flags, explanation, dataset_id
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14)
			ON CONFLICT (lead_id) DO UPDATE SET
				chain_only_score=$4, network_score=$5, fused_score=$6,
				chain_only_rank=$7, fused_rank=$8, rank_shift=$9,
				network_evidence_quality=$10, heuristic_association_strength=$11,
				anomaly_flags=$12::jsonb, explanation=$13, computed_at=NOW()`,
			c.EntityID, c.EntityID, c.TXID, c.ChainScore, c.NetworkScore, c.FusedScore,
			cr, fr, cr-fr, c.NetworkEvidenceQuality, c.AssociationStrength, string(flagsJSON), c.Explanation, "6bc084b677a63411",
		)
		if err != nil {
			return inserted, fmt.Errorf("ranking: inserting lead for entity %s: %w", c.EntityID, err)
		}
		inserted++
	}
	return inserted, nil
}

func workerFailureFlag(err error) string {
	var workerErr *intelligence.WorkerError
	if errors.As(err, &workerErr) {
		switch workerErr.Kind {
		case intelligence.WorkerTimeout:
			return "INTELLIGENCE_WORKER_TIMEOUT"
		case intelligence.WorkerHTTPError:
			return "INTELLIGENCE_WORKER_HTTP_ERROR"
		case intelligence.WorkerMalformed:
			return "INTELLIGENCE_WORKER_MALFORMED_RESPONSE"
		}
	}
	return "INTELLIGENCE_WORKER_UNAVAILABLE"
}

func rankBy(candidates []candidate, score func(candidate) float64) map[string]int {
	ranked := append([]candidate(nil), candidates...)
	sort.SliceStable(ranked, func(i, j int) bool { return score(ranked[i]) > score(ranked[j]) })
	ranks := make(map[string]int, len(ranked))
	for i, c := range ranked {
		ranks[c.EntityID] = i + 1
	}
	return ranks
}

func sigmoid(x float64) float64 { return 1 / (1 + math.Exp(-x)) }

type entityRow struct {
	EntityID        string
	MemberAddresses []string
}

func loadEntities(ctx context.Context, pool *pgxpool.Pool) ([]entityRow, error) {
	rows, err := pool.Query(ctx, `SELECT entity_id, member_addresses::text FROM entities`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entityRow
	for rows.Next() {
		var id, addrJSON string
		if err := rows.Scan(&id, &addrJSON); err != nil {
			return nil, err
		}
		var addrs []string
		if err := json.Unmarshal([]byte(addrJSON), &addrs); err != nil {
			return nil, err
		}
		out = append(out, entityRow{EntityID: id, MemberAddresses: addrs})
	}
	return out, rows.Err()
}

func loadTxByInputAddress(ctx context.Context, pool *pgxpool.Pool) (map[string]string, error) {
	rows, err := pool.Query(ctx, `SELECT txid, input_addresses::text FROM transactions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var txid, addrJSON string
		if err := rows.Scan(&txid, &addrJSON); err != nil {
			return nil, err
		}
		var addrs []string
		if err := json.Unmarshal([]byte(addrJSON), &addrs); err != nil {
			return nil, err
		}
		for _, a := range addrs {
			if _, exists := out[a]; !exists {
				out[a] = txid
			}
		}
	}
	return out, rows.Err()
}

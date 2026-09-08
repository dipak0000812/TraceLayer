package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type leadDTO struct {
	LeadID                       string   `json:"lead_id"`
	EntityID                     string   `json:"entity_id"`
	PrimaryTXID                  string   `json:"primary_txid"`
	ChainOnlyScore               float64  `json:"chain_only_score"`
	NetworkScore                 float64  `json:"network_score"`
	FusedScore                   float64  `json:"fused_score"`
	ChainOnlyRank                int      `json:"chain_only_rank"`
	FusedRank                    int      `json:"fused_rank"`
	RankShift                    int      `json:"rank_shift"`
	NetworkEvidenceQuality       float64  `json:"network_evidence_quality"`
	HeuristicAssociationStrength float64  `json:"heuristic_association_strength"`
	AnomalyFlags                 []string `json:"anomaly_flags"`
	Explanation                  string   `json:"explanation"`
}

func leadsListHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 {
			limit = 20
		}
		orderCol := "fused_rank"
		if r.URL.Query().Get("sort_by") == "rank_shift" {
			orderCol = "rank_shift"
		}

		var total int
		if err := deps.Pool.QueryRow(r.Context(), `SELECT count(*) FROM forensic_leads`).Scan(&total); err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, err.Error(), nil)
			return
		}

		rows, err := deps.Pool.Query(r.Context(), `
			SELECT lead_id, entity_id, primary_txid, chain_only_score, network_score, fused_score,
			       chain_only_rank, fused_rank, rank_shift, network_evidence_quality,
			       heuristic_association_strength, anomaly_flags::text, explanation
			FROM forensic_leads ORDER BY `+orderCol+` ASC LIMIT $1 OFFSET $2`,
			limit, (page-1)*limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, err.Error(), nil)
			return
		}
		defer rows.Close()

		leads := []leadDTO{}
		for rows.Next() {
			var l leadDTO
			var flagsJSON string
			if err := rows.Scan(&l.LeadID, &l.EntityID, &l.PrimaryTXID, &l.ChainOnlyScore, &l.NetworkScore,
				&l.FusedScore, &l.ChainOnlyRank, &l.FusedRank, &l.RankShift, &l.NetworkEvidenceQuality,
				&l.HeuristicAssociationStrength, &flagsJSON, &l.Explanation); err != nil {
				writeError(w, http.StatusInternalServerError, CodeInternalError, err.Error(), nil)
				return
			}
			_ = parseFlags(flagsJSON, &l.AnomalyFlags)
			leads = append(leads, l)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"total_leads": total, "page": page, "limit": limit, "leads": leads,
		})
	}
}

func leadDetailHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var l leadDTO
		var flagsJSON string
		err := deps.Pool.QueryRow(r.Context(), `
			SELECT lead_id, entity_id, primary_txid, chain_only_score, network_score, fused_score,
			       chain_only_rank, fused_rank, rank_shift, network_evidence_quality,
			       heuristic_association_strength, anomaly_flags::text, explanation
			FROM forensic_leads WHERE lead_id = $1`, id).Scan(
			&l.LeadID, &l.EntityID, &l.PrimaryTXID, &l.ChainOnlyScore, &l.NetworkScore, &l.FusedScore,
			&l.ChainOnlyRank, &l.FusedRank, &l.RankShift, &l.NetworkEvidenceQuality,
			&l.HeuristicAssociationStrength, &flagsJSON, &l.Explanation)
		if err != nil {
			writeError(w, http.StatusNotFound, CodeNotFound, "lead not found", nil)
			return
		}
		_ = parseFlags(flagsJSON, &l.AnomalyFlags)
		writeJSON(w, http.StatusOK, l)
	}
}

func parseFlags(raw string, out *[]string) error {
	return json.Unmarshal([]byte(raw), out)
}

package domain

import (
	"time"
)

// Transaction represents a parsed Bitcoin transaction
type Transaction struct {
	TxID            string      `json:"txid"`
	BlockTime       time.Time   `json:"timestamp"`
	InputAddresses  []string    `json:"input_addresses"`
	OutputAddresses []string    `json:"output_addresses"`
	InputAmounts    []float64   `json:"input_amounts"`
	OutputAmounts   []float64   `json:"output_amounts"`
	Fee             float64     `json:"fee"`
	ScriptType      string      `json:"script_type"`
	Provenance      string      `json:"provenance"`
	DatasetID       string      `json:"dataset_id"`
}

// NetworkObservation represents a parsed P2P network observation
type NetworkObservation struct {
	ObservationID string    `json:"observation_id"`
	ObservedTxID  string    `json:"txid"`
	ObservedAt    time.Time `json:"timestamp"`
	SrcIP         string    `json:"src_ip"`
	DstIP         string    `json:"dst_ip"`
	SrcPort       int       `json:"src_port"`
	DstPort       int       `json:"dst_port"`
	GeoCountry    *string   `json:"geo_country,omitempty"`
	ASN           *string   `json:"asn,omitempty"`
	Provenance    string    `json:"provenance"`
	DatasetID     string    `json:"dataset_id"`
}

// ForensicLead represents a final ranked entity cluster
type ForensicLead struct {
	LeadID               string    `json:"lead_id"`
	EntityID             string    `json:"entity_id"`
	PrimaryTxID          string    `json:"primary_txid"`
	ChainOnlyScore       float64   `json:"chain_only_score"`
	NetworkScore         float64   `json:"network_score"`
	FusedScore           float64   `json:"fused_score"`
	ChainOnlyRank        int       `json:"chain_only_rank"`
	FusedRank            int       `json:"fused_rank"`
	RankShift            int       `json:"rank_shift"`
	NetworkEvidenceQ     float64   `json:"network_evidence_quality"`
	HeuristicAssoc       float64   `json:"heuristic_association_strength"`
	AnomalyFlags         []string  `json:"anomaly_flags"`
	Explanation          string    `json:"explanation"`
}

-- PostgreSQL DDL for TraceLayer

CREATE TABLE IF NOT EXISTS transactions (
    txid VARCHAR(64) PRIMARY KEY,
    block_time TIMESTAMPTZ NOT NULL,
    input_addresses JSONB NOT NULL,
    output_addresses JSONB NOT NULL,
    input_amounts JSONB NOT NULL,
    output_amounts JSONB NOT NULL,
    fee NUMERIC(16,8) NOT NULL,
    script_type VARCHAR(16) NOT NULL,
    provenance VARCHAR(32) NOT NULL DEFAULT 'SYNTHETIC',
    dataset_id VARCHAR(32) NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS network_observations (
    observation_id VARCHAR(64) PRIMARY KEY,
    observed_txid VARCHAR(64) NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    src_ip VARCHAR(45) NOT NULL,
    dst_ip VARCHAR(45) NOT NULL,
    src_port INT NOT NULL,
    dst_port INT NOT NULL,
    geo_country VARCHAR(2),             -- Enriched offline
    asn VARCHAR(16),                    -- Enriched offline
    correlation_status VARCHAR(16) NOT NULL DEFAULT 'PENDING', -- PENDING, CORRELATED, ORPHAN
    provenance VARCHAR(32) NOT NULL DEFAULT 'SYNTHETIC',
    dataset_id VARCHAR(32) NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_net_obs_txid ON network_observations(observed_txid);
CREATE INDEX IF NOT EXISTS idx_net_obs_status ON network_observations(correlation_status);

CREATE TABLE IF NOT EXISTS forensic_leads (
    lead_id VARCHAR(64) PRIMARY KEY,
    entity_id VARCHAR(64) NOT NULL,
    primary_txid VARCHAR(64) NOT NULL,
    chain_only_score FLOAT NOT NULL,
    network_score FLOAT NOT NULL,
    fused_score FLOAT NOT NULL,
    chain_only_rank INTEGER NOT NULL,
    fused_rank INTEGER NOT NULL,
    rank_shift INTEGER NOT NULL,
    network_evidence_quality FLOAT NOT NULL,
    heuristic_association_strength FLOAT NOT NULL,
    anomaly_flags TEXT[] NOT NULL,
    explanation TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_leads_fused_rank ON forensic_leads(fused_rank);

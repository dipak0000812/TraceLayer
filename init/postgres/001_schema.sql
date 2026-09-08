-- init/postgres/001_schema.sql
--
-- TraceLayer Round-2 schema. Mounted into Postgres's docker-entrypoint-initdb.d
-- so it runs automatically on first container start (Phase 14). Every
-- statement is idempotent (IF NOT EXISTS) so manual application against an
-- existing database is also safe.
--
-- Authority: docs/DATA_CONTRACT.md sections 3 and 5.3, docs/API_CONTRACT.md
-- sections 1.5 and 2 (batch/rejection persistence). transactions and
-- network_observations match DATA_CONTRACT.md's DDL snippet verbatim.
-- entities, forensic_leads, ingestion_batches, rejection_logs are not in
-- that snippet but are required by endpoints both docs define elsewhere.

-- ============================================================
-- transactions
-- ============================================================
-- fee is NUMERIC(16,8) BTC — exact arbitrary-precision decimal, NOT
-- floating point. The Go layer converts to/from domain.Satoshis via
-- domain.ParseBTCString / Satoshis.String() only — never through float64.
CREATE TABLE IF NOT EXISTS transactions (
    txid              VARCHAR(64) PRIMARY KEY,
    block_time        TIMESTAMPTZ NOT NULL,
    input_addresses   JSONB NOT NULL,
    output_addresses  JSONB NOT NULL,
    input_amounts     JSONB NOT NULL,
    output_amounts    JSONB NOT NULL,
    fee               NUMERIC(16,8) NOT NULL,
    script_type       VARCHAR(16) NOT NULL,
    provenance        VARCHAR(32) NOT NULL DEFAULT 'SYNTHETIC',
    dataset_id        VARCHAR(32) NOT NULL,
    generator_version VARCHAR(16) NOT NULL,
    ingested_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_transactions_script_type
        CHECK (script_type IN ('P2PKH', 'P2WPKH', 'P2SH', 'P2TR')),
    CONSTRAINT chk_transactions_provenance
        CHECK (provenance IN ('SYNTHETIC', 'DERIVED', 'REAL_NODE_CAPTURE', 'REAL_PUBLIC')),
    CONSTRAINT chk_transactions_fee_nonnegative
        CHECK (fee >= 0)
);
-- txid PRIMARY KEY is the idempotency/conflict boundary Phase 3 relies on.
-- No ON CONFLICT DO UPDATE exists anywhere in this schema, intentionally:
-- silent overwrite is structurally impossible here, not just discouraged.

-- ============================================================
-- network_observations
-- ============================================================
-- observed_txid intentionally has NO foreign key to transactions.txid.
-- Network ingestion is decoupled from blockchain ingestion; do not add one
-- later without an explicit contract-change decision.
CREATE TABLE IF NOT EXISTS network_observations (
    observation_id       VARCHAR(64) PRIMARY KEY,
    observed_txid         VARCHAR(64) NOT NULL,
    observed_at           TIMESTAMPTZ NOT NULL,
    src_ip                VARCHAR(45) NOT NULL,
    dst_ip                VARCHAR(45) NOT NULL,
    src_port              INT NOT NULL,
    dst_port              INT NOT NULL,
    geo_country           VARCHAR(2),
    asn                   VARCHAR(16),
    propagation_delay_ms  BIGINT,
    correlation_status    VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    provenance            VARCHAR(32) NOT NULL DEFAULT 'SYNTHETIC',
    dataset_id            VARCHAR(32) NOT NULL,
    generator_version     VARCHAR(16) NOT NULL,
    ingested_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_net_obs_correlation_status
        CHECK (correlation_status IN ('PENDING', 'CORRELATED', 'ORPHAN')),
    CONSTRAINT chk_net_obs_provenance
        CHECK (provenance IN ('SYNTHETIC', 'DERIVED', 'REAL_NODE_CAPTURE', 'REAL_PUBLIC')),
    CONSTRAINT chk_net_obs_ports
        CHECK (src_port BETWEEN 1024 AND 65535 AND dst_port BETWEEN 1024 AND 65535)
);
-- Dedup key is observation_id (PK). Unlike transactions, the contract
-- defines no "identical payload is a harmless no-op" case for observations —
-- Phase 4 should treat any observation_id collision as a straight rejection
-- unless you decide otherwise.

CREATE INDEX IF NOT EXISTS idx_net_obs_txid   ON network_observations(observed_txid);
CREATE INDEX IF NOT EXISTS idx_net_obs_status ON network_observations(correlation_status);

-- ============================================================
-- entities  (DSU output — DATA_CONTRACT.md section 4.2)
-- ============================================================
CREATE TABLE IF NOT EXISTS entities (
    entity_id         VARCHAR(64) PRIMARY KEY,
    member_addresses  JSONB NOT NULL,
    cluster_size      INT NOT NULL,
    provenance        VARCHAR(32) NOT NULL DEFAULT 'SYNTHETIC',
    dataset_id        VARCHAR(32) NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_entities_cluster_size_positive CHECK (cluster_size > 0)
);

-- ============================================================
-- forensic_leads  (DATA_CONTRACT.md section 5.3)
-- ============================================================
-- entity_id / primary_txid ARE foreign-keyed here — a lead only exists after
-- correlation+clustering have already run against already-ingested data, so
-- there's no decoupling requirement to preserve, unlike network_observations.
CREATE TABLE IF NOT EXISTS forensic_leads (
    lead_id                          VARCHAR(64) PRIMARY KEY,
    entity_id                        VARCHAR(64) NOT NULL REFERENCES entities(entity_id),
    primary_txid                     VARCHAR(64) NOT NULL REFERENCES transactions(txid),
    chain_only_score                 DOUBLE PRECISION NOT NULL,
    network_score                    DOUBLE PRECISION NOT NULL,
    fused_score                      DOUBLE PRECISION NOT NULL,
    chain_only_rank                  INT NOT NULL,
    fused_rank                       INT NOT NULL,
    rank_shift                       INT NOT NULL,
    network_evidence_quality         DOUBLE PRECISION NOT NULL,
    heuristic_association_strength   DOUBLE PRECISION NOT NULL,
    anomaly_flags                    JSONB NOT NULL,
    explanation                      TEXT NOT NULL,
    provenance                       VARCHAR(32) NOT NULL DEFAULT 'SYNTHETIC',
    dataset_id                       VARCHAR(32) NOT NULL,
    computed_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_leads_scores_0_1 CHECK (
        chain_only_score BETWEEN 0 AND 1 AND
        network_score BETWEEN 0 AND 1 AND
        fused_score BETWEEN 0 AND 1 AND
        network_evidence_quality BETWEEN 0 AND 1 AND
        heuristic_association_strength BETWEEN 0 AND 1
    ),
    CONSTRAINT chk_leads_ranks_positive CHECK (chain_only_rank >= 1 AND fused_rank >= 1),
    CONSTRAINT chk_leads_rank_shift CHECK (rank_shift = chain_only_rank - fused_rank)
);

CREATE INDEX IF NOT EXISTS idx_leads_entity ON forensic_leads(entity_id);

-- ============================================================
-- ingestion_batches / rejection_logs
-- (backs API_CONTRACT.md GET /batches/{id} and .../rejections)
-- ============================================================
CREATE TABLE IF NOT EXISTS ingestion_batches (
    batch_id        UUID PRIMARY KEY,
    file_type       VARCHAR(16) NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'ACCEPTED',
    accepted_count  INT NOT NULL DEFAULT 0,
    rejected_count  INT NOT NULL DEFAULT 0,
    dataset_id      VARCHAR(32) NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,

    CONSTRAINT chk_batches_file_type CHECK (file_type IN ('bulk', 'transactions', 'network'))
);

CREATE TABLE IF NOT EXISTS rejection_logs (
    id                BIGSERIAL PRIMARY KEY,
    batch_id          UUID NOT NULL REFERENCES ingestion_batches(batch_id),
    line_number       INT NOT NULL,
    rejection_reason  VARCHAR(64) NOT NULL,
    field_name        VARCHAR(64),
    raw_record        TEXT NOT NULL,
    rejected_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rejection_logs_batch ON rejection_logs(batch_id);
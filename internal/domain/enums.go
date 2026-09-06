// internal/domain/enums.go
package domain

// ScriptType enumerates output script types, per DATA_CONTRACT.md section 2.1.
type ScriptType string

const (
	ScriptTypeP2PKH  ScriptType = "P2PKH"
	ScriptTypeP2WPKH ScriptType = "P2WPKH"
	ScriptTypeP2SH   ScriptType = "P2SH"
	ScriptTypeP2TR   ScriptType = "P2TR"
)

func (t ScriptType) Valid() bool {
	switch t {
	case ScriptTypeP2PKH, ScriptTypeP2WPKH, ScriptTypeP2SH, ScriptTypeP2TR:
		return true
	}
	return false
}

// Provenance tags evidentiary origin. Round-2 data is always SYNTHETIC; the
// other values exist so the schema doesn't change if a later round ingests
// real captures.
type Provenance string

const (
	ProvenanceSynthetic       Provenance = "SYNTHETIC"
	ProvenanceDerived         Provenance = "DERIVED"
	ProvenanceRealNodeCapture Provenance = "REAL_NODE_CAPTURE"
	ProvenanceRealPublic      Provenance = "REAL_PUBLIC"
)

func (p Provenance) Valid() bool {
	switch p {
	case ProvenanceSynthetic, ProvenanceDerived, ProvenanceRealNodeCapture, ProvenanceRealPublic:
		return true
	}
	return false
}

// CorrelationStatus tracks whether a network observation has matched a known
// transaction. PENDING and ORPHAN are expected steady-state values, not
// errors — network ingestion is decoupled from blockchain ingestion.
type CorrelationStatus string

const (
	CorrelationPending    CorrelationStatus = "PENDING"
	CorrelationCorrelated CorrelationStatus = "CORRELATED"
	CorrelationOrphan     CorrelationStatus = "ORPHAN"
)

func (c CorrelationStatus) Valid() bool {
	switch c {
	case CorrelationPending, CorrelationCorrelated, CorrelationOrphan:
		return true
	}
	return false
}
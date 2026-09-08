// internal/domain/doc.go
// Package domain contains TraceLayer's canonical in-process types.
//
// This package has zero dependencies on database, HTTP, or any other
// TraceLayer package. It is the one place downstream layers (ingestion,
// correlation, fusion, API) agree on what a Transaction, NetworkObservation,
// Entity, and ForensicLead actually are. Field-level contracts are per
// docs/DATA_CONTRACT.md; where this package disagrees with that document,
// the document wins and this package has a bug.
package domain
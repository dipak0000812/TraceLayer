// internal/domain/entity.go
package domain

import "errors"

// Entity is the output of common-input-ownership clustering (DSU). Dipak
// owns this implementation directly in Round 2 — there is no separate
// entity-resolution owner.
//
// Address != person. Entity != guaranteed real-world identity. This is a
// heuristic clustering of addresses observed to co-spend, nothing more.
type Entity struct {
	EntityID        string
	MemberAddresses []string
	ClusterSize     int
}

var (
	ErrEmptyEntityID       = errors.New("domain: entity_id must not be empty")
	ErrEmptyMembers        = errors.New("domain: entity must have at least one member address")
	ErrClusterSizeMismatch = errors.New("domain: cluster_size does not match len(member_addresses)")
)

func (e Entity) Validate() error {
	if e.EntityID == "" {
		return ErrEmptyEntityID
	}
	if len(e.MemberAddresses) == 0 {
		return ErrEmptyMembers
	}
	if e.ClusterSize != len(e.MemberAddresses) {
		return ErrClusterSizeMismatch
	}
	return nil
}
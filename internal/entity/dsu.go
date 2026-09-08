// internal/entity/dsu.go
package entity

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/dipak0000812/TraceLayer/internal/domain"
)

// Cluster implements common-input-ownership entity resolution: within each
// transaction, every input address is assumed controlled by the same
// real-world entity (the standard co-spend heuristic). Change-address
// reasoning is deliberately NOT applied — it's a weaker heuristic this
// project's derived-UTXO representation can't reliably support, and adding
// it would be exactly the kind of unsupported sophistication the project
// brief warns against.
//
// inputGroups is one []string of input addresses per transaction; order and
// which transaction they came from don't matter to clustering correctness,
// only which addresses co-occurred.
//
// Address != person; Entity != guaranteed real-world identity — see
// domain.Entity's own doc comment.
func Cluster(inputGroups [][]string) []domain.Entity {
	parent := make(map[string]string)

	var find func(string) string
	find = func(a string) string {
		if parent[a] != a {
			parent[a] = find(parent[a])
		}
		return parent[a]
	}
	union := func(a, b string) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}

	for _, group := range inputGroups {
		for _, addr := range group {
			if _, ok := parent[addr]; !ok {
				parent[addr] = addr
			}
		}
		for i := 1; i < len(group); i++ {
			union(group[0], group[i])
		}
	}

	clusters := make(map[string][]string)
	for addr := range parent {
		root := find(addr)
		clusters[root] = append(clusters[root], addr)
	}

	entities := make([]domain.Entity, 0, len(clusters))
	for _, members := range clusters {
		sort.Strings(members)
		entities = append(entities, domain.Entity{
			EntityID:        deterministicEntityID(members),
			MemberAddresses: members,
			ClusterSize:     len(members),
		})
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].EntityID < entities[j].EntityID })
	return entities
}

// deterministicEntityID hashes sorted member addresses so that identical
// cluster membership always yields the same entity_id, satisfying
// DATA_CONTRACT.md §4.2's "deterministic" requirement independent of
// insertion order or a random UUID generator.
func deterministicEntityID(sortedMembers []string) string {
	h := sha256.New()
	for i, addr := range sortedMembers {
		if i > 0 {
			h.Write([]byte("\n"))
		}
		h.Write([]byte(addr))
	}
	return hex.EncodeToString(h.Sum(nil))
}
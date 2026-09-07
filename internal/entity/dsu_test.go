// internal/entity/dsu_test.go
package entity

import "testing"

func TestCluster_CommonInputsMergeIntoOneEntity(t *testing.T) {
	groups := [][]string{
		{"addrA", "addrB"},
		{"addrB", "addrC"},
	}
	entities := Cluster(groups)
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d: %+v", len(entities), entities)
	}
	if entities[0].ClusterSize != 3 {
		t.Fatalf("ClusterSize = %d, want 3", entities[0].ClusterSize)
	}
	if err := entities[0].Validate(); err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestCluster_DisjointGroupsStaySeparate(t *testing.T) {
	groups := [][]string{
		{"addrA", "addrB"},
		{"addrX", "addrY"},
	}
	entities := Cluster(groups)
	if len(entities) != 2 {
		t.Fatalf("expected 2 separate entities, got %d: %+v", len(entities), entities)
	}
}

func TestCluster_SingleInputTransactionFormsSingletonEntity(t *testing.T) {
	entities := Cluster([][]string{{"addrLonely"}})
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	if entities[0].ClusterSize != 1 {
		t.Fatalf("ClusterSize = %d, want 1", entities[0].ClusterSize)
	}
}

func TestCluster_TransitiveChainMergesAllThree(t *testing.T) {
	// addrA-addrB in tx1, addrB-addrC in tx2 — transitively one entity even
	// though addrA and addrC never co-occur in the same transaction directly.
	groups := [][]string{
		{"addrA", "addrB"},
		{"addrB", "addrC"},
		{"addrD"}, // unrelated singleton
	}
	entities := Cluster(groups)
	if len(entities) != 2 {
		t.Fatalf("expected 2 entities (one merged, one singleton), got %d: %+v", len(entities), entities)
	}
}

func TestCluster_DeterministicAcrossRuns(t *testing.T) {
	groups := [][]string{
		{"addrA", "addrB"},
		{"addrC", "addrD"},
	}
	first := Cluster(groups)
	second := Cluster(groups)
	if len(first) != len(second) {
		t.Fatalf("entity count differs across runs: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].EntityID != second[i].EntityID {
			t.Fatalf("EntityID differs across identical runs: %s vs %s", first[i].EntityID, second[i].EntityID)
		}
	}
}
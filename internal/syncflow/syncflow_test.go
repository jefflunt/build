package syncflow

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalize_SemanticInvariance(t *testing.T) {
	// Two logically identical JSONs with different key ordering and whitespace
	json1 := []byte(`{"type":"tab","id":"build-orchestrator-tab","disabled":false}`)
	json2 := []byte(`{
    "disabled": false,
    "id": "build-orchestrator-tab",
    "type": "tab"
}`)

	norm1, err := Normalize(json1)
	if err != nil {
		t.Fatalf("unexpected error normalizing json1: %v", err)
	}

	norm2, err := Normalize(json2)
	if err != nil {
		t.Fatalf("unexpected error normalizing json2: %v", err)
	}

	if !bytes.Equal(norm1, norm2) {
		t.Errorf("expected normalized JSONs to be identical, but got:\nNORM1:\n%s\n\nNORM2:\n%s", string(norm1), string(norm2))
	}
}

func TestHash_Invariance(t *testing.T) {
	json1 := []byte(`{"id":"1","name":"A"}`)
	json2 := []byte(`{ "name": "A", "id": "1" }`)

	hash1, err := Hash(json1)
	if err != nil {
		t.Fatalf("unexpected error hashing json1: %v", err)
	}

	hash2, err := Hash(json2)
	if err != nil {
		t.Fatalf("unexpected error hashing json2: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("expected hashes to be identical, got %s and %s", hash1, hash2)
	}
}

func TestNormalize_Indent(t *testing.T) {
	jsonInput := []byte(`{"id":"1"}`)
	norm, err := Normalize(jsonInput)
	if err != nil {
		t.Fatalf("unexpected error normalizing: %v", err)
	}

	// Verify pretty-printed 4-space indent is used
	expected := "{\n    \"id\": \"1\"\n}"
	if string(norm) != expected {
		t.Errorf("expected normalized output to be %q, got %q", expected, string(norm))
	}
}

func TestPartitionNodes(t *testing.T) {
	flatJSON := []byte(`[
		{"id": "tab1", "type": "tab", "label": "My Flow"},
		{"id": "subflow1", "type": "subflow", "name": "My Subflow"},
		{"id": "node1", "type": "inject", "z": "tab1"},
		{"id": "node2", "type": "debug", "z": "subflow1"},
		{"id": "config1", "type": "mqtt-broker"}
	]`)

	var nodes []interface{}
	if err := json.Unmarshal(flatJSON, &nodes); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	partitions, err := PartitionNodes(nodes)
	if err != nil {
		t.Fatalf("failed to partition: %v", err)
	}

	if len(partitions) != 3 {
		t.Fatalf("expected 3 partitions, got %d", len(partitions))
	}

	var foundTab, foundSubflow, foundGlobal bool
	for _, p := range partitions {
		switch p.Type {
		case "tab":
			if p.ID != "tab1" || p.Name != "My Flow" || len(p.Nodes) != 2 {
				t.Errorf("invalid tab partition: %+v", p)
			}
			foundTab = true
		case "subflow":
			if p.ID != "subflow1" || p.Name != "My Subflow" || len(p.Nodes) != 2 {
				t.Errorf("invalid subflow partition: %+v", p)
			}
			foundSubflow = true
		case "global":
			if p.ID != "global" || len(p.Nodes) != 1 {
				t.Errorf("invalid global partition: %+v", p)
			}
			foundGlobal = true
		}
	}

	if !foundTab || !foundSubflow || !foundGlobal {
		t.Errorf("missing partitions: tab=%t, subflow=%t, global=%t", foundTab, foundSubflow, foundGlobal)
	}
}

func TestSyncState_LoadAndSave(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sync-state-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	statePath := filepath.Join(tempDir, "sync_state.json")

	// 1. Loading non-existent file should return an empty initialized state
	state, err := LoadSyncState(statePath)
	if err != nil {
		t.Fatalf("expected no error loading non-existent state, got: %v", err)
	}
	if state.Files == nil {
		t.Fatal("expected initialized Files map")
	}

	// 2. Modify and save
	state.LastSyncTime = "2026-06-13T12:00:00Z"
	state.Files["flows/sdlc-orchestrator-v1.json"] = FileSyncState{
		NodeRedID:      "tab1",
		Type:           "tab",
		LastKnownHash:  "abc123hash",
		LastKnownMtime: "some-time",
	}

	err = SaveSyncState(statePath, state)
	if err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// 3. Reload and verify
	reloaded, err := LoadSyncState(statePath)
	if err != nil {
		t.Fatalf("failed to reload state: %v", err)
	}

	if reloaded.LastSyncTime != "2026-06-13T12:00:00Z" {
		t.Errorf("expected sync time %s, got %s", "2026-06-13T12:00:00Z", reloaded.LastSyncTime)
	}

	fState, ok := reloaded.Files["flows/sdlc-orchestrator-v1.json"]
	if !ok {
		t.Fatal("expected file state to exist")
	}
	if fState.NodeRedID != "tab1" || fState.LastKnownHash != "abc123hash" {
		t.Errorf("incorrect file state values: %+v", fState)
	}
}

package syncflow

import (
	"bytes"
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

package syncflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Normalize parses arbitrary JSON and re-serializes it deterministically
// with alphabetically sorted map keys and 4-space indentation.
func Normalize(jsonBytes []byte) ([]byte, error) {
	var obj interface{}
	if err := json.Unmarshal(jsonBytes, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	normalized, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal normalized JSON: %w", err)
	}

	return normalized, nil
}

// Hash normalizes the JSON content first and then computes its SHA256 checksum.
// This ensures that two logically identical JSON payloads with different key orders
// or whitespace configurations return the exact same checksum.
func Hash(jsonBytes []byte) (string, error) {
	normalized, err := Normalize(jsonBytes)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(normalized)
	return hex.EncodeToString(hash[:]), nil
}

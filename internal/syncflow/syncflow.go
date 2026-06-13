package syncflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FileSyncState tracks synchronization metadata for a local file.
type FileSyncState struct {
	NodeRedID      string `json:"node_red_id"`
	Type           string `json:"type"` // "tab", "subflow", or "global"
	LastKnownHash  string `json:"last_known_hash"`
	LastKnownMtime string `json:"last_known_mtime"`
}

// SyncState represents the full synchronization tracker.
type SyncState struct {
	LastSyncTime string                    `json:"last_sync_time"`
	Files        map[string]FileSyncState `json:"files"`
}

// LoadSyncState reads and parses the sync state file. If the file does not exist,
// it returns an empty but initialized SyncState struct.
func LoadSyncState(path string) (*SyncState, error) {
	state := &SyncState{
		Files: make(map[string]FileSyncState),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, fmt.Errorf("failed to read sync state file: %w", err)
	}

	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sync state: %w", err)
	}

	if state.Files == nil {
		state.Files = make(map[string]FileSyncState)
	}

	return state, nil
}

// SaveSyncState serializes and saves the sync state back to disk.
func SaveSyncState(path string, state *SyncState) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for sync state: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal sync state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write sync state: %w", err)
	}

	return nil
}

// Partition represents a self-contained grouping of Node-RED nodes.
type Partition struct {
	ID    string        `json:"id"`
	Type  string        `json:"type"` // "tab", "subflow", or "global"
	Name  string        `json:"name"` // tab label, subflow name, or "global"
	Nodes []interface{} `json:"nodes"`
}

// PartitionNodes splits a flat array of Node-RED nodes into distinct partitions.
func PartitionNodes(nodes []interface{}) ([]Partition, error) {
	type rootInfo struct {
		id   string
		typ  string
		name string
	}

	roots := make(map[string]rootInfo)

	// First pass: identify tabs and subflow definitions
	for _, n := range nodes {
		m, ok := n.(map[string]interface{})
		if !ok {
			continue
		}
		idVal, _ := m["id"].(string)
		typVal, _ := m["type"].(string)

		if typVal == "tab" && idVal != "" {
			labelVal, _ := m["label"].(string)
			roots[idVal] = rootInfo{id: idVal, typ: "tab", name: labelVal}
		} else if typVal == "subflow" && idVal != "" {
			nameVal, _ := m["name"].(string)
			roots[idVal] = rootInfo{id: idVal, typ: "subflow", name: nameVal}
		}
	}

	// Initialize partitions
	partMap := make(map[string]*Partition)
	for id, r := range roots {
		partMap[id] = &Partition{
			ID:    id,
			Type:  r.typ,
			Name:  r.name,
			Nodes: []interface{}{},
		}
	}

	globalPart := &Partition{
		ID:    "global",
		Type:  "global",
		Name:  "global",
		Nodes: []interface{}{},
	}

	// Second pass: group nodes into partitions
	for _, n := range nodes {
		m, ok := n.(map[string]interface{})
		if !ok {
			continue
		}
		idVal, _ := m["id"].(string)
		typVal, _ := m["type"].(string)
		zVal, _ := m["z"].(string)

		if typVal == "tab" || typVal == "subflow" {
			// Tab/subflow definitions themselves belong in their own partition
			if part, exists := partMap[idVal]; exists {
				part.Nodes = append(part.Nodes, n)
			} else {
				// Fallback (shouldn't happen)
				globalPart.Nodes = append(globalPart.Nodes, n)
			}
		} else if zVal != "" {
			if part, exists := partMap[zVal]; exists {
				part.Nodes = append(part.Nodes, n)
			} else {
				// Belong to a non-existent tab/subflow definition, group in global
				globalPart.Nodes = append(globalPart.Nodes, n)
			}
		} else {
			// Config / global node
			globalPart.Nodes = append(globalPart.Nodes, n)
		}
	}

	// Gather result
	var result []Partition
	for _, part := range partMap {
		if len(part.Nodes) > 0 {
			result = append(result, *part)
		}
	}
	if len(globalPart.Nodes) > 0 {
		result = append(result, *globalPart)
	}

	return result, nil
}

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

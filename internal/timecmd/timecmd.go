package timecmd

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// TaskTimeNode represents a task in the hierarchy with its duration information.
type TaskTimeNode struct {
	ID         string
	ParentID   string
	Title      string
	DirectTime int
	TotalTime  int
	Children   []*TaskTimeNode
}

// FormatDuration converts an integer number of seconds into a padded HhMMmSSs format.
// E.g., 3665 -> 1h01m05s
func FormatDuration(seconds int) string {
	if seconds < 0 {
		return "0h00m00s" // fallback for negative
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
}

// BuildTimeTree fetches tasks and durations, builds the tree, and calculates rollups.
func BuildTimeTree(db *sql.DB) ([]*TaskTimeNode, error) {
	query := `
		SELECT t.id, COALESCE(t.parent_id, ''), t.title, COALESCE(SUM(a.duration_seconds), 0) as direct_time
		FROM tasks t
		LEFT JOIN audit_logs a ON t.id = a.task_id
		WHERE t.status = 'done'
		GROUP BY t.id
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks and durations: %w", err)
	}
	defer rows.Close()

	nodesMap := make(map[string]*TaskTimeNode)
	var roots []*TaskTimeNode

	// 1. Load all nodes into a map
	for rows.Next() {
		node := &TaskTimeNode{}
		err := rows.Scan(&node.ID, &node.ParentID, &node.Title, &node.DirectTime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		node.TotalTime = node.DirectTime // Base case
		nodesMap[node.ID] = node
	}

	// 2. Build the tree structure
	for _, node := range nodesMap {
		if node.ParentID != "" {
			if parent, exists := nodesMap[node.ParentID]; exists {
				parent.Children = append(parent.Children, node)
			} else {
				// Parent doesn't exist or isn't 'done', treat this node as a root for our display
				roots = append(roots, node)
			}
		} else {
			roots = append(roots, node)
		}
	}

	// 3. Sort children and roots to ensure deterministic output
	var sortNodes func(nodes []*TaskTimeNode)
	sortNodes = func(nodes []*TaskTimeNode) {
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Title < nodes[j].Title // Sort alphabetically by title
		})
		for _, n := range nodes {
			sortNodes(n.Children)
		}
	}
	sortNodes(roots)

	// 4. Calculate total times (bottom-up)
	var calculateRollup func(node *TaskTimeNode) int
	calculateRollup = func(node *TaskTimeNode) int {
		total := node.DirectTime
		for _, child := range node.Children {
			total += calculateRollup(child)
		}
		node.TotalTime = total
		return total
	}

	for _, root := range roots {
		calculateRollup(root)
	}

	return roots, nil
}

// PrintTimeTree recursively prints the tree to stdout.
func PrintTimeTree(nodes []*TaskTimeNode, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, node := range nodes {
		fmt.Printf("%s[%s] / [%s] %s\n", 
			indent, 
			FormatDuration(node.TotalTime), 
			FormatDuration(node.DirectTime), 
			node.Title)
		PrintTimeTree(node.Children, depth+1)
	}
}

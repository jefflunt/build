package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jefflunt/build/internal/db"
)


func TestWalkBreakdownDir(t *testing.T) {
	// Setup mock directory
	tempDir, err := os.MkdirTemp("", "breakdown-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create root README.md
	os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("# Root Title\nRoot Description"), 0644)

	// Create child directory
	childDir := filepath.Join(tempDir, "child1")
	os.Mkdir(childDir, 0755)
	os.WriteFile(filepath.Join(childDir, "README.md"), []byte("# Child Title\nChild Description"), 0644)
	os.WriteFile(filepath.Join(childDir, "task1.md"), []byte("# Task1 Title\nTask1 Description"), 0644)

	// Run walkBreakdownDir
	node, err := walkBreakdownDir(tempDir, "")
	if err != nil {
		t.Fatal(err)
	}

	if node.Title != "Root Title" {
		t.Errorf("expected Root Title, got %s", node.Title)
	}
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(node.Children))
	}

	child := node.Children[0]
	if child.Title != "Child Title" {
		t.Errorf("expected Child Title, got %s", child.Title)
	}
	if len(child.Children) != 1 {
		t.Fatalf("expected 1 task in child, got %d", len(child.Children))
	}
	if child.Children[0].Title != "Task1 Title" {
		t.Errorf("expected Task1 Title, got %s", child.Children[0].Title)
	}
}

func TestRunCLI_RM_InvalidSyntax(t *testing.T) {
	if os.Getenv("BE_CRASHER_RM_INVALID") == "1" {
		runCLI([]string{"build", "rm"})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestRunCLI_RM_InvalidSyntax")
	cmd.Env = append(os.Environ(), "BE_CRASHER_RM_INVALID=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit status 1 for invalid rm syntax, got 0")
	}
	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "Usage: build rm <id:<task-id>|status:<status>>") {
		t.Errorf("expected usage message, got: %q", out)
	}
}

func TestRunCLI_RM_NoMatches(t *testing.T) {
	// Setup a temporary directory to run the test in so we have a clean/isolated environment
	tempDir, err := os.MkdirTemp("", "main-rm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create .build directory and initialize an empty database
	buildDir := filepath.Join(tempDir, ".build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(buildDir, "build.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	database.Close()

	if os.Getenv("BE_CRASHER_RM_VALID") == "1" {
		runCLI([]string{"build", "rm", "status:nonexistent"})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunCLI_RM_NoMatches")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "BE_CRASHER_RM_VALID=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("expected successful execution, got error: %v, stderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "No tasks found matching the criteria.") {
		t.Errorf("expected 'No tasks found' message, got: %q", out)
	}
}

func TestRunCLI_RM_Success(t *testing.T) {
	// Setup a temporary directory to run the test in so we have a clean/isolated environment
	tempDir, err := os.MkdirTemp("", "main-rm-success-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create .build directory and initialize database
	buildDir := filepath.Join(tempDir, ".build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(buildDir, "build.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	// Insert test data
	_, err = database.Exec(`
		INSERT INTO tasks (id, status) VALUES ('task1', 'todo');
		INSERT INTO comments (task_id, content) VALUES ('task1', 'some comment');
		INSERT INTO audit_logs (task_id, action) VALUES ('task1', 'some action');
	`)
	if err != nil {
		database.Close()
		t.Fatalf("Failed to insert mock data: %v", err)
	}
	database.Close()

	if os.Getenv("BE_CRASHER_RM_SUCCESS") == "1" {
		runCLI([]string{"build", "rm", "id:task1"})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunCLI_RM_Success")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "BE_CRASHER_RM_SUCCESS=1")
	cmd.Stdin = strings.NewReader("y\n") // Simulate confirming deletion

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("expected successful execution, got error: %v, stderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Successfully deleted 1 tasks.") {
		t.Errorf("expected success message, got: %q", out)
	}

	// Verify tasks are deleted from tables
	dbVerify, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB for verification: %v", err)
	}
	defer dbVerify.Close()

	var count int
	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'task1'").Scan(&count)
	if count != 0 {
		t.Error("Expected task1 to be deleted from tasks table")
	}

	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM comments WHERE task_id = 'task1'").Scan(&count)
	if count != 0 {
		t.Error("Expected comments to be deleted from comments table")
	}

	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE task_id = 'task1'").Scan(&count)
	if count != 0 {
		t.Error("Expected audit logs to be deleted from audit_logs table")
	}
}

func TestRunCLI_RM_Abort(t *testing.T) {
	// Setup a temporary directory to run the test in so we have a clean/isolated environment
	tempDir, err := os.MkdirTemp("", "main-rm-abort-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create .build directory and initialize database
	buildDir := filepath.Join(tempDir, ".build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(buildDir, "build.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	// Insert test data
	_, err = database.Exec(`
		INSERT INTO tasks (id, status) VALUES ('task1', 'todo');
		INSERT INTO comments (task_id, content) VALUES ('task1', 'some comment');
		INSERT INTO audit_logs (task_id, action) VALUES ('task1', 'some action');
	`)
	if err != nil {
		database.Close()
		t.Fatalf("Failed to insert mock data: %v", err)
	}
	database.Close()

	if os.Getenv("BE_CRASHER_RM_ABORT") == "1" {
		runCLI([]string{"build", "rm", "id:task1"})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunCLI_RM_Abort")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "BE_CRASHER_RM_ABORT=1")
	cmd.Stdin = strings.NewReader("n\n") // Simulate aborting deletion

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("expected successful execution, got error: %v, stderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Deletion aborted.") {
		t.Errorf("expected abort message, got: %q", out)
	}

	// Verify tasks are NOT deleted from tables
	dbVerify, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB for verification: %v", err)
	}
	defer dbVerify.Close()

	var count int
	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'task1'").Scan(&count)
	if count != 1 {
		t.Error("Expected task1 NOT to be deleted from tasks table")
	}

	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM comments WHERE task_id = 'task1'").Scan(&count)
	if count != 1 {
		t.Error("Expected comments NOT to be deleted from comments table")
	}

	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE task_id = 'task1'").Scan(&count)
	if count != 1 {
		t.Error("Expected audit logs NOT to be deleted from audit_logs table")
	}
}

func TestRunCLI_DeployFlow_Success(t *testing.T) {
	// 1. Start a mock Node-RED server
	var receivedBody string
	var receivedMethod string
	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		receivedBody = buf.String()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"revision": "xyz"}`))
	}))
	defer ts.Close()

	// 2. Setup mock home directory and configuration
	tempHome, err := os.MkdirTemp("", "deploy-flow-test-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", origHome)

	buildDir := filepath.Join(tempHome, ".build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(buildDir, "config.yml")
	configData := []byte("agent_adapter: agent:live_opencode\nnode_red_url: " + ts.URL)
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Create a mock flow file in the workspace
	tempWorkspace, err := os.MkdirTemp("", "deploy-flow-test-workspace")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempWorkspace)

	origWd, _ := os.Getwd()
	os.Chdir(tempWorkspace)
	defer os.Chdir(origWd)

	// Create workflows directory
	if err := os.MkdirAll("workflows", 0755); err != nil {
		t.Fatal(err)
	}
	flowFile := filepath.Join("workflows", "sdlc-orchestrator.json")
	flowData := []byte(`[{"id":"node1","type":"tab"}]`)
	if err := os.WriteFile(flowFile, flowData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create .build in workspace too because init/runCLI checks it
	if err := os.MkdirAll(".build", 0755); err != nil {
		t.Fatal(err)
	}

	// 4. Run CLI command
	deployFlow(flowFile)

	// 5. Verify the HTTP request to mock Node-RED
	if receivedMethod != "POST" {
		t.Errorf("expected POST method, got %s", receivedMethod)
	}
	if receivedPath != "/flows" {
		t.Errorf("expected path /flows, got %s", receivedPath)
	}
	if !strings.Contains(receivedBody, `node1`) {
		t.Errorf("expected body to contain node1, got %s", receivedBody)
	}
}

func TestRunCLI_SyncFlows_AlreadyInSync(t *testing.T) {
	// 1. Mock Node-RED server
	var getCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/flows" {
			getCalled = true
			w.WriteHeader(http.StatusOK)
			// Return identical content, just different spacing
			w.Write([]byte(`[{"type":"tab","id":"node1"}]`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer ts.Close()

	// 2. Setup config
	tempHome, err := os.MkdirTemp("", "sync-test-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", origHome)

	buildDir := filepath.Join(tempHome, ".build")
	os.MkdirAll(buildDir, 0755)

	// We'll create a temp flows.json to simulate remote flow file on disk
	remoteFlowsFile := filepath.Join(tempHome, "flows.json")
	os.WriteFile(remoteFlowsFile, []byte(`[{"id":"node1","type":"tab"}]`), 0644)

	configFile := filepath.Join(buildDir, "config.yml")
	configData := []byte("agent_adapter: agent:live_opencode\nnode_red_url: " + ts.URL + "\nnode_red_flows_path: " + remoteFlowsFile)
	os.WriteFile(configFile, configData, 0644)

	// 3. Setup workspace
	tempWorkspace, err := os.MkdirTemp("", "sync-test-workspace")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempWorkspace)

	origWd, _ := os.Getwd()
	os.Chdir(tempWorkspace)
	defer os.Chdir(origWd)

	os.MkdirAll("workflows", 0755)
	localFlowsFile := filepath.Join("workflows", "sdlc-orchestrator.json")
	os.WriteFile(localFlowsFile, []byte(`[{"id":"node1","type":"tab"}]`), 0644)
	os.MkdirAll(".build", 0755)

	// 4. Run sync
	syncFlows()

	// 5. Verify GET was called, but no files were overwritten because they were already in sync
	if !getCalled {
		t.Error("expected GET /flows to be called to fetch remote flows")
	}
}

func TestRunCLI_SyncFlows_LocalNewer(t *testing.T) {
	// 1. Mock Node-RED server (expects POST)
	var postCalled bool
	var receivedBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/flows" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"type":"tab","id":"old-remote-node"}]`))
			return
		}
		if r.Method == "POST" && r.URL.Path == "/flows" {
			postCalled = true
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)
			receivedBody = buf.String()
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer ts.Close()

	// 2. Setup config & mock remote flow file
	tempHome, err := os.MkdirTemp("", "sync-local-newer-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", origHome)

	buildDir := filepath.Join(tempHome, ".build")
	os.MkdirAll(buildDir, 0755)

	remoteFlowsFile := filepath.Join(tempHome, "flows.json")
	os.WriteFile(remoteFlowsFile, []byte(`[{"id":"old-remote-node","type":"tab"}]`), 0644)

	configFile := filepath.Join(buildDir, "config.yml")
	configData := []byte("agent_adapter: agent:live_opencode\nnode_red_url: " + ts.URL + "\nnode_red_flows_path: " + remoteFlowsFile)
	os.WriteFile(configFile, configData, 0644)

	// 3. Setup workspace
	tempWorkspace, err := os.MkdirTemp("", "sync-local-newer-workspace")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempWorkspace)

	origWd, _ := os.Getwd()
	os.Chdir(tempWorkspace)
	defer os.Chdir(origWd)

	os.MkdirAll("workflows", 0755)
	localFlowsFile := filepath.Join("workflows", "sdlc-orchestrator.json")
	os.WriteFile(localFlowsFile, []byte(`[{"id":"new-local-node","type":"tab"}]`), 0644)
	os.MkdirAll(".build", 0755)

	// Set local file to be modified in the future (newer than remote)
	future := time.Now().Add(1 * time.Hour)
	err = os.Chtimes(localFlowsFile, future, future)
	if err != nil {
		t.Fatal(err)
	}

	// Set remote file to be modified in the past
	past := time.Now().Add(-1 * time.Hour)
	err = os.Chtimes(remoteFlowsFile, past, past)
	if err != nil {
		t.Fatal(err)
	}

	// 4. Run sync
	syncFlows()

	// 5. Verify POST was called with local content
	if !postCalled {
		t.Error("expected POST /flows to be called because local was newer")
	}
	if !strings.Contains(receivedBody, "new-local-node") {
		t.Errorf("expected POST body to contain new-local-node, got %s", receivedBody)
	}
}

func TestRunCLI_SyncFlows_RemoteNewer(t *testing.T) {
	// 1. Mock Node-RED server (expects GET)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/flows" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"type":"tab","id":"newer-remote-node"}]`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer ts.Close()

	// 2. Setup config & mock remote flow file
	tempHome, err := os.MkdirTemp("", "sync-remote-newer-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", origHome)

	buildDir := filepath.Join(tempHome, ".build")
	os.MkdirAll(buildDir, 0755)

	remoteFlowsFile := filepath.Join(tempHome, "flows.json")
	os.WriteFile(remoteFlowsFile, []byte(`[{"id":"newer-remote-node","type":"tab"}]`), 0644)

	configFile := filepath.Join(buildDir, "config.yml")
	configData := []byte("agent_adapter: agent:live_opencode\nnode_red_url: " + ts.URL + "\nnode_red_flows_path: " + remoteFlowsFile)
	os.WriteFile(configFile, configData, 0644)

	// 3. Setup workspace
	tempWorkspace, err := os.MkdirTemp("", "sync-remote-newer-workspace")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempWorkspace)

	origWd, _ := os.Getwd()
	os.Chdir(tempWorkspace)
	defer os.Chdir(origWd)

	os.MkdirAll("workflows", 0755)
	localFlowsFile := filepath.Join("workflows", "sdlc-orchestrator.json")
	os.WriteFile(localFlowsFile, []byte(`[{"id":"old-local-node","type":"tab"}]`), 0644)
	os.MkdirAll(".build", 0755)

	// Set remote file to be modified in the future (newer than local)
	future := time.Now().Add(1 * time.Hour)
	err = os.Chtimes(remoteFlowsFile, future, future)
	if err != nil {
		t.Fatal(err)
	}

	// Set local file to be modified in the past
	past := time.Now().Add(-1 * time.Hour)
	err = os.Chtimes(localFlowsFile, past, past)
	if err != nil {
		t.Fatal(err)
	}

	// 4. Run sync
	syncFlows()

	// 5. Verify local file was overwritten with remote content
	localData, err := os.ReadFile(localFlowsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(localData), "newer-remote-node") {
		t.Errorf("expected local file to be updated with newer-remote-node, got %s", string(localData))
	}
}

func TestRunCLI_SyncFlows_SemanticInvariance(t *testing.T) {
	// 1. Mock Node-RED server (expects GET)
	var postCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/flows" {
			w.WriteHeader(http.StatusOK)
			// Return identical content with different formatting and key order
			w.Write([]byte(`[{"type":"tab","id":"node1","disabled":false}]`))
			return
		}
		if r.Method == "POST" && r.URL.Path == "/flows" {
			postCalled = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer ts.Close()

	// 2. Setup config & mock remote flow file
	tempHome, err := os.MkdirTemp("", "sync-invariant-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", origHome)

	buildDir := filepath.Join(tempHome, ".build")
	os.MkdirAll(buildDir, 0755)

	remoteFlowsFile := filepath.Join(tempHome, "flows.json")
	os.WriteFile(remoteFlowsFile, []byte(`[{"id":"node1","type":"tab","disabled":false}]`), 0644)

	configFile := filepath.Join(buildDir, "config.yml")
	configData := []byte("agent_adapter: agent:live_opencode\nnode_red_url: " + ts.URL + "\nnode_red_flows_path: " + remoteFlowsFile)
	os.WriteFile(configFile, configData, 0644)

	// 3. Setup workspace
	tempWorkspace, err := os.MkdirTemp("", "sync-invariant-workspace")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempWorkspace)

	origWd, _ := os.Getwd()
	os.Chdir(tempWorkspace)
	defer os.Chdir(origWd)

	os.MkdirAll("workflows", 0755)
	localFlowsFile := filepath.Join("workflows", "sdlc-orchestrator.json")
	// Write with different key order, spacing, and minification
	os.WriteFile(localFlowsFile, []byte(`[ { "disabled": false, "id": "node1", "type": "tab" } ]`), 0644)
	os.MkdirAll(".build", 0755)

	// Make local mod time newer than remote, which would normally trigger sync
	future := time.Now().Add(1 * time.Hour)
	err = os.Chtimes(localFlowsFile, future, future)
	if err != nil {
		t.Fatal(err)
	}

	// 4. Run sync
	syncFlows()

	// 5. Verify no POST was called because the contents are semantically identical!
	if postCalled {
		t.Error("expected NO POST /flows to be called because files are semantically identical")
	}
}


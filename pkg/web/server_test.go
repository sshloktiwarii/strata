package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sshloktiwarii/lsm-tree-db/pkg/lsm"
)

func TestWebWorkbenchEndpoints(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "web_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := lsm.Open(lsm.DefaultOptions(tmpDir))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	srv := NewServer(db, 5001)

	// Test GET /
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.handleIndex(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for index, got %d", w.Code)
	}

	// Test POST /api/kv (PUT)
	body, _ := json.Marshal(map[string]string{
		"action": "put",
		"key":    "api-test-key",
		"value":  "api-test-val",
	})
	req = httptest.NewRequest("POST", "/api/kv", bytes.NewReader(body))
	w = httptest.NewRecorder()
	srv.handleKV(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for put, got %d", w.Code)
	}

	// Test POST /api/kv (GET)
	body, _ = json.Marshal(map[string]string{
		"action": "get",
		"key":    "api-test-key",
	})
	req = httptest.NewRequest("POST", "/api/kv", bytes.NewReader(body))
	w = httptest.NewRecorder()
	srv.handleKV(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for get, got %d", w.Code)
	}

	// Test GET /api/stats
	req = httptest.NewRequest("GET", "/api/stats", nil)
	w = httptest.NewRecorder()
	srv.handleStats(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for stats, got %d", w.Code)
	}

	// Test POST /api/benchmark
	benchBody, _ := json.Marshal(map[string]int{"count": 500})
	req = httptest.NewRequest("POST", "/api/benchmark", bytes.NewReader(benchBody))
	w = httptest.NewRecorder()
	srv.handleBenchmark(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for benchmark, got %d", w.Code)
	}
}

package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/sshloktiwarii/lsm-tree-db/pkg/lsm"
)

//go:embed static/*
var staticFS embed.FS

// Server hosts the embedded benchmark & inspection workbench.
type Server struct {
	db     *lsm.DB
	port   int
	server *http.Server
}

// NewServer creates a new web workbench server for the LSM database.
func NewServer(db *lsm.DB, port int) *Server {
	if port <= 0 {
		port = 5001
	}
	return &Server{db: db, port: port}
}

// Start boots the HTTP workbench on the designated port.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/levels", s.handleLevels)
	mux.HandleFunc("/api/kv", s.handleKV)
	mux.HandleFunc("/api/benchmark", s.handleBenchmark)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	return s.server.ListenAndServe()
}

// Close gracefully terminates the HTTP server.
func (s *Server) Close() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.db.Stats()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleLevels(w http.ResponseWriter, r *http.Request) {
	tables := s.db.InspectLevels()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tables)
}

type kvReq struct {
	Action string `json:"action"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

func (s *Server) handleKV(w http.ResponseWriter, r *http.Request) {
	var req kvReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	switch req.Action {
	case "put":
		if err := s.db.Put([]byte(req.Key), []byte(req.Value)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	case "get":
		val, err := s.db.Get([]byte(req.Key))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "key not found"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"key": req.Key, "value": string(val)})
	case "delete":
		if err := s.db.Delete([]byte(req.Key)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	case "flush":
		if err := s.db.Flush(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "flushed"})
	case "compact":
		if err := s.db.Compact(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "compacted"})
	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
	}
}

type benchReq struct {
	Count int `json:"count"`
}

func (s *Server) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	var req benchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Count <= 0 {
		req.Count = 10000
	}
	if req.Count > 100000 {
		req.Count = 100000
	}

	val := []byte("bench-pressure-data-payload-128b-block-pad-00000000000000000000000000000000000000000000000000000000000000000000000000000000")

	// 1. Write phase
	writeStart := time.Now()
	for i := 0; i < req.Count; i++ {
		k := []byte(fmt.Sprintf("bench-web-%08d", i))
		_ = s.db.Put(k, val)
	}
	writeDur := time.Since(writeStart)
	writeOps := float64(req.Count) / writeDur.Seconds()

	// 2. Read phase
	readStart := time.Now()
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < req.Count; i++ {
		target := rnd.Intn(req.Count)
		k := []byte(fmt.Sprintf("bench-web-%08d", target))
		_, _ = s.db.Get(k)
	}
	readDur := time.Since(readStart)
	readOps := float64(req.Count) / readDur.Seconds()

	stats := s.db.Stats()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"count":         req.Count,
		"write_ops_sec": int(writeOps),
		"read_ops_sec":  int(readOps),
		"read_amp":      stats.ReadAmplification,
		"bloom_saves":   stats.BloomFilterSaves,
		"total_gets":    stats.TotalGets,
	})
}

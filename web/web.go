package web

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Sagor0078/distribKV/config"
	"github.com/Sagor0078/distribKV/db"
	"github.com/Sagor0078/distribKV/replication"
)

// Server contains HTTP handlers to interact with the key-value store.
type Server struct {
	db     *db.Database
	shards *config.Shards
}

// NewServer creates a new HTTP server instance with database and shard metadata.
func NewServer(db *db.Database, shards *config.Shards) *Server {
	return &Server{
		db:     db,
		shards: shards,
	}
}

// redirect forwards a request to the correct shard based on key hash.
func (s *Server) redirect(shard int, w http.ResponseWriter, r *http.Request) {
	target := "http://" + s.shards.Addrs[shard] + r.RequestURI
	log.Printf("Redirecting request to shard %d → %d (%s)", s.shards.CurIdx, shard, target)

	resp, err := http.Get(target)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error redirecting request: %v", err)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// GetHandler handles GET requests for a key.
func (s *Server) GetHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	if key == "" {
		http.Error(w, "Missing key", http.StatusBadRequest)
		return
	}

	shard := s.shards.Index(key)
	if shard != s.shards.CurIdx {
		s.redirect(shard, w, r)
		return
	}

	val, err := s.db.GetKey(key)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, "Value: %s", val)
}

// SetHandler handles write requests for a key-value pair.
func (s *Server) SetHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	value := r.Form.Get("value")
	if key == "" || value == "" {
		http.Error(w, "Missing key or value", http.StatusBadRequest)
		return
	}

	shard := s.shards.Index(key)
	if shard != s.shards.CurIdx {
		s.redirect(shard, w, r)
		return
	}

	err := s.db.SetKey(key, []byte(value))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to set key: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Key set successfully on shard %d", shard)
}

// DeleteExtraKeysHandler deletes keys that don't belong to the current shard.
func (s *Server) DeleteExtraKeysHandler(w http.ResponseWriter, r *http.Request) {
	err := s.db.DeleteExtraKeys(func(key string) bool {
		return s.shards.Index(key) != s.shards.CurIdx
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete extra keys: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Extra keys deleted successfully")
}

// GetNextKeyForReplication serves the next key to be replicated to followers.
func (s *Server) GetNextKeyForReplication(w http.ResponseWriter, r *http.Request) {
	key, value, err := s.db.GetNextKeyForReplication()
	res := &replication.NextKeyValue{
		Key:   string(key),
		Value: string(value),
		Err:   err,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// DeleteReplicationKey deletes a key from the replication queue.
func (s *Server) DeleteReplicationKey(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	value := r.Form.Get("value")

	if key == "" || value == "" {
		http.Error(w, "Missing key or value", http.StatusBadRequest)
		return
	}

	err := s.db.DeleteReplicationKey([]byte(key), []byte(value))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting replication key: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "ok")
}

package web_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Sagor0078/distribKV/config"
	"github.com/Sagor0078/distribKV/db"
	"github.com/Sagor0078/distribKV/replication"
	"github.com/Sagor0078/distribKV/web"
)

func createTempDB(t *testing.T, idx int) *db.Database {
	t.Helper()

	dir, err := os.MkdirTemp("", fmt.Sprintf("shard%d", idx))
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	database, closer, err := db.NewDatabase(dir, false)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	
	t.Cleanup(func() {
	    if err := closer(); err != nil {
	        t.Errorf("Cleanup error: %v", err)
	    }
	})
	return database
}

func createTestServer(t *testing.T, idx int, addrs map[int]string) (*db.Database, *web.Server) {
	t.Helper()
	database := createTempDB(t, idx)
	cfg := &config.Shards{
		Addrs:  addrs,
		Count:  len(addrs),
		CurIdx: idx,
	}
	server := web.NewServer(database, cfg)
	return database, server
}

func TestSetAndGetHandler(t *testing.T) {
	var ts1Handler, ts2Handler http.HandlerFunc

	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts1Handler(w, r)
	}))
	defer ts1.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts2Handler(w, r)
	}))
	defer ts2.Close()

	addrs := map[int]string{
		0: strings.TrimPrefix(ts1.URL, "http://"),
		1: strings.TrimPrefix(ts2.URL, "http://"),
	}

	_, server1 := createTestServer(t, 0, addrs)
	_, server2 := createTestServer(t, 1, addrs)

	ts1Handler = func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/set") {
			server1.SetHandler(w, r)
		} else {
			server1.GetHandler(w, r)
		}
	}

	ts2Handler = func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/set") {
			server2.SetHandler(w, r)
		} else {
			server2.GetHandler(w, r)
		}
	}

	// Pick keys that map to different shards
	key1 := "key-one"
	key2 := "key-two"
	value1 := "val-one"
	value2 := "val-two"

	// SET requests
	for _, kv := range [][2]string{{key1, value1}, {key2, value2}} {
		url := fmt.Sprintf("%s/set?key=%s&value=%s", ts1.URL, kv[0], kv[1])
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("Failed to set key %q: %v", kv[0], err)
		}
		resp.Body.Close()
	}

	// GET requests
	for _, kv := range [][2]string{{key1, value1}, {key2, value2}} {
		url := fmt.Sprintf("%s/get?key=%s", ts1.URL, kv[0])
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("Failed to get key %q: %v", kv[0], err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if !bytes.Contains(body, []byte(kv[1])) {
			t.Errorf("Expected value %q for key %q, got: %q", kv[1], kv[0], body)
		}
	}
}

func TestDeleteExtraKeysHandler(t *testing.T) {
	addrs := map[int]string{
		0: "localhost:1111",
		1: "localhost:1112",
	}

	db := createTempDB(t, 0)
	server := web.NewServer(db, &config.Shards{
		Addrs:  addrs,
		Count:  2,
		CurIdx: 0,
	})

	// Add keys that belong to shard 1 (so they're "extra")
	extraKey := "some-key-for-shard1"
	_ = db.SetKey(extraKey, []byte("should-be-deleted"))

	req := httptest.NewRequest("POST", "/delete-extra", nil)
	w := httptest.NewRecorder()
	server.DeleteExtraKeysHandler(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if !bytes.Contains(body, []byte("Extra keys deleted successfully")) {
		t.Errorf("Unexpected response: %q", body)
	}
}

func TestGetNextKeyForReplication(t *testing.T) {
	db := createTempDB(t, 0)
	server := web.NewServer(db, &config.Shards{Addrs: map[int]string{}, Count: 1, CurIdx: 0})

	// Add to replication queue
	key := "foo"
	value := []byte("bar")
	_ = db.SetKey(key, value)

	req := httptest.NewRequest("GET", "/next", nil)
	w := httptest.NewRecorder()
	server.GetNextKeyForReplication(w, req)

	resp := w.Result()
	var result replication.NextKeyValue
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Key != key || result.Value != string(value) || result.Err != nil {
		t.Errorf("Unexpected replication result: %+v", result)
	}
}

func TestDeleteReplicationKey(t *testing.T) {
	db := createTempDB(t, 0)
	server := web.NewServer(db, &config.Shards{Addrs: map[int]string{}, Count: 1, CurIdx: 0})

	key := "temp-key"
	value := "temp-val"

	_ = db.SetKey(key, []byte(value)) // Enqueue for replication

	req := httptest.NewRequest("POST", fmt.Sprintf("/delete-replication?key=%s&value=%s", key, value), nil)
	w := httptest.NewRecorder()
	server.DeleteReplicationKey(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("Expected 'ok', got: %q", body)
	}
}

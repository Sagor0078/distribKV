package main

import (
	"fmt"
	"log"
	"net/http"
	"distribKV/store"
)

func setHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")
	if key == "" || value == "" {
		http.Error(w, "key and value required", http.StatusBadRequest)
		return
	}
	store.Set(key, value)
	w.Write([]byte("OK"))
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	if val, ok := store.Get(key); ok {
		w.Write([]byte(val))
	} else {
		http.NotFound(w, r)
	}
}

func main() {
	http.HandleFunc("/set", setHandler)
	http.HandleFunc("/get", getHandler)
	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
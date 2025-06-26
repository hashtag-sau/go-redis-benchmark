package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	cache = make(map[string]string)
	mu    sync.RWMutex
)

func main() {
	http.HandleFunc("/data", dataHandler)

	log.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("id")
	if key == "" {
		http.Error(w, "Missing id param", http.StatusBadRequest)
		return
	}

	
	mu.RLock()
	val, found := cache[key]
	mu.RUnlock()

	if found {
		fmt.Fprintf(w, "From Cache: %s\n", val)
		return
	}

	
	time.Sleep(100 * time.Millisecond)
	val = fmt.Sprintf("Value-for-%s", key)

	
	mu.Lock()
	cache[key] = val
	mu.Unlock()

	fmt.Fprintf(w, "From DB: %s\n", val)
}

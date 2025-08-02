package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"user-service/cache"
	"user-service/db"
	"user-service/handlers"
)

func main() {

	fmt.Println("USE_INMEMORY =", os.Getenv("USE_INMEMORY"))

	db.Init()
	cache.Init()

	http.HandleFunc("/user", handlers.UserHandler)

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Cache Hits: %d\n", atomic.LoadInt64(&cache.Hits))
		fmt.Fprintf(w, "Cache Misses: %d\n", atomic.LoadInt64(&cache.Misses))
	})

	log.Println("Service running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
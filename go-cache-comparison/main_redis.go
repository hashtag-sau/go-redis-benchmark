package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9" //using redis
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

func main() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", 
		Password: "",               
		DB:       0,                
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	http.HandleFunc("/data", dataHandler)

	log.Println("Server running on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("id")
	if key == "" {
		http.Error(w, "Missing id param", http.StatusBadRequest)
		return
	}

	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		
		time.Sleep(100 * time.Millisecond)
		val = fmt.Sprintf("Value-for-%s", key)

		err := rdb.Set(ctx, key, val, 10*time.Minute).Err()
		if err != nil {
			http.Error(w, "Failed to cache value", 500)
			return
		}

		fmt.Fprintf(w, "From DB: %s\n", val)
	} else if err != nil {
		http.Error(w, "Redis error", 500)
	} else {
		fmt.Fprintf(w, "From Redis: %s\n", val)
	}
}

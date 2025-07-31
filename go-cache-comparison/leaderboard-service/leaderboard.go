package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

var ctx = context.Background()

// Configurable cache type: "redis" or "inmemory"
var cacheType = os.Getenv("CACHE_TYPE")

// Redis client
var rdb *redis.Client

// In-memory leaderboard
var memStore = struct {
	sync.RWMutex
	scores map[string]int
}{scores: make(map[string]int)}

// Internal metrics
var metrics = struct {
	sync.Mutex
	RequestCount     int
	CacheHits        int
	CacheMisses      int
	RedisOps         int
	Latencies        []float64
	StartTime        time.Time
}{
	Latencies: make([]float64, 0, 100000),
	StartTime: time.Now(),
}

func main() {
	if cacheType == "redis" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: "",
			DB:       0,
		})
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Fatalf("Redis connection failed: %v", err)
		}
		log.Println("Using Redis cache")
	} else {
		log.Println("Using In-Memory cache")
	}

	r := mux.NewRouter()
	r.HandleFunc("/score/{user_id}", metricsMiddleware(postScoreHandler)).Methods("POST")
	r.HandleFunc("/leaderboard/top", metricsMiddleware(getLeaderboardHandler)).Methods("GET")
	r.HandleFunc("/metrics/summary", metricsSummaryHandler).Methods("GET")

	addr := getEnv("SERVICE_ADDR", ":8080")
	log.Printf("Server running at %s", addr)
	http.ListenAndServe(addr, r)
}

func postScoreHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["user_id"]
	scoreStr := r.URL.Query().Get("score")
	score, err := strconv.Atoi(scoreStr)
	if err != nil {
		http.Error(w, "Invalid score", http.StatusBadRequest)
		return
	}

	if cacheType == "redis" {
		rdb.ZAdd(ctx, "leaderboard", &redis.Z{Score: float64(score), Member: userID})
		metrics.Lock()
		metrics.RedisOps++
		metrics.Unlock()
	} else {
		memStore.Lock()
		prev, ok := memStore.scores[userID]
		if ok && prev == score {
			metrics.Lock()
			metrics.CacheHits++
			metrics.Unlock()
		} else {
			metrics.Lock()
			metrics.CacheMisses++
			metrics.Unlock()
		}
		memStore.scores[userID] = score
		memStore.Unlock()
	}

	w.WriteHeader(http.StatusOK)
}

func getLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	topNStr := r.URL.Query().Get("n")
	if topNStr == "" {
		topNStr = "10"
	}
	topN, err := strconv.Atoi(topNStr)
	if err != nil || topN <= 0 {
		http.Error(w, "Invalid number of top users", http.StatusBadRequest)
		return
	}

	if cacheType == "redis" {
		results, err := rdb.ZRevRangeWithScores(ctx, "leaderboard", 0, int64(topN-1)).Result()
		if err != nil {
			http.Error(w, "Redis error", http.StatusInternalServerError)
			return
		}
		metrics.Lock()
		metrics.RedisOps++
		metrics.Unlock()
		json.NewEncoder(w).Encode(results)
	} else {
		memStore.RLock()
		snap := make([]struct {
			UserID string  `json:"user_id"`
			Score  float64 `json:"score"`
		}, 0, len(memStore.scores))
		for k, v := range memStore.scores {
			snap = append(snap, struct {
				UserID string  `json:"user_id"`
				Score  float64 `json:"score"`
			}{UserID: k, Score: float64(v)})
		}
		memStore.RUnlock()

		sort.Slice(snap, func(i, j int) bool {
			return snap[i].Score > snap[j].Score
		})

		if topN < len(snap) {
			snap = snap[:topN]
		}

		json.NewEncoder(w).Encode(snap)
	}
}

func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		latency := time.Since(start).Seconds()
		metrics.Lock()
		metrics.RequestCount++
		metrics.Latencies = append(metrics.Latencies, latency)
		metrics.Unlock()
	}
}

func metricsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	metrics.Lock()
	defer metrics.Unlock()

	latencies := append([]float64(nil), metrics.Latencies...)
	sort.Float64s(latencies)

	p50 := percentile(latencies, 50)
	p95 := percentile(latencies, 95)
	p99 := percentile(latencies, 99)

	summary := map[string]interface{}{
		"total_requests": metrics.RequestCount,
		"cache_hits":     metrics.CacheHits,
		"cache_misses":   metrics.CacheMisses,
		"redis_ops":      metrics.RedisOps,
		"avg_latency":    average(latencies),
		"p50_latency":    p50,
		"p95_latency":    p95,
		"p99_latency":    p99,
		"uptime_seconds": time.Since(metrics.StartTime).Seconds(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func percentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return 0
	}
	k := float64(len(data)-1) * p / 100.0
	f := math.Floor(k)
	c := math.Ceil(k)
	if f == c {
		return data[int(k)]
	}
	d0 := data[int(f)] * (c - k)
	d1 := data[int(c)] * (k - f)
	return d0 + d1
}

func average(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

var ctx = context.Background()

// --- CONFIG KNOBS --- //
var (
	USE_REDIS            = getEnvBool("USE_REDIS", false)
	SESSION_TTL          = getEnvDuration("SESSION_TTL", 10*time.Second)
	CLEANUP_INTERVAL     = getEnvDuration("CLEANUP_INTERVAL", 5*time.Second)
	PAYLOAD_SIZE_BYTES   = getEnvInt("PAYLOAD_SIZE", 512) // payload size in bytes
	REDIS_ADDR           = getEnv("REDIS_ADDR", "localhost:6379")
)

// --- IN-MEMORY CACHE --- //
type Session struct {
	Data      string
	ExpiresAt time.Time
}

type InMemoryCache struct {
	mu     sync.RWMutex
	store  map[string]Session
}

func NewInMemoryCache() *InMemoryCache {
	c := &InMemoryCache{store: make(map[string]Session)}
	go c.cleanup()
	return c
}

func (c *InMemoryCache) Set(key string, data string, ttl time.Duration) {
	exp := time.Now().Add(ttl)
	c.mu.Lock()
	c.store[key] = Session{Data: data, ExpiresAt: exp}
	c.mu.Unlock()
}

func (c *InMemoryCache) Get(key string) (string, bool) {
	c.mu.RLock()
	sess, ok := c.store[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(sess.ExpiresAt) {
		return "", false
	}
	return sess.Data, true
}

func (c *InMemoryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.store, key)
	c.mu.Unlock()
}

func (c *InMemoryCache) cleanup() {
	for {
		time.Sleep(CLEANUP_INTERVAL)
		now := time.Now()
		c.mu.Lock()
		for k, v := range c.store {
			if now.After(v.ExpiresAt) {
				delete(c.store, k)
			}
		}
		c.mu.Unlock()
	}
}

// --- REDIS CACHE --- //
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache() *RedisCache {
	return &RedisCache{
		client: redis.NewClient(&redis.Options{
			Addr: REDIS_ADDR,
		}),
	}
}

func (c *RedisCache) Set(key string, data string, ttl time.Duration) {
	c.client.Set(ctx, key, data, ttl)
}

func (c *RedisCache) Get(key string) (string, bool) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil || err != nil {
		return "", false
	}
	return val, true
}

func (c *RedisCache) Delete(key string) {
	c.client.Del(ctx, key)
}

// --- UTILS --- //
func getEnv(key string, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func getEnvInt(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return i
}

func getEnvBool(key string, def bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return def
	}
	return b
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return def
	}
	return d
}

// --- MAIN SERVER --- //
func main() {
	var cache interface {
		Set(string, string, time.Duration)
		Get(string) (string, bool)
		Delete(string)
	}

	if USE_REDIS {
		cache = NewRedisCache()
		log.Println("Using Redis cache")
	} else {
		cache = NewInMemoryCache()
		log.Println("Using In-Memory cache")
	}

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		id := fmt.Sprintf("sess_%d", rand.Intn(1e9))
		payload := make([]byte, PAYLOAD_SIZE_BYTES)
		rand.Read(payload)
		encoded := base64.StdEncoding.EncodeToString(payload)
		cache.Set(id, encoded, SESSION_TTL)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(id))
	})

	http.HandleFunc("/session/", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path[len("/session/"):]
		val, ok := cache.Get(key)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(val))
	})

	http.HandleFunc("/logout/", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path[len("/logout/"):]
		cache.Delete(key)
		w.WriteHeader(http.StatusOK)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

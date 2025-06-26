package cache

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	Rdb   *redis.Client
	Ctx   = context.Background()
	store = make(map[string]string)
	mu    sync.RWMutex
	useInMemory = true
)

func Init() {
	if useInMemory { // skip redis 
		return 
	}

	// else use redis
	Rdb = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_HOST") + ":6379",
	})
	if _, err := Rdb.Ping(Ctx).Result(); err != nil {
		panic(err)
	}
}

var Hits, Misses int64

func Get(key string) (string, error) {
	if useInMemory {
		mu.RLock()
		defer mu.RUnlock()
		val, ok := store[key]
		if !ok {
			atomic.AddInt64(&Misses, 1)
			return "", redis.Nil
		}
		atomic.AddInt64(&Hits, 1)
		return val, nil
	}
	val, err := Rdb.Get(Ctx, key).Result()
	if err == nil {
		atomic.AddInt64(&Hits, 1)
	} else {
		atomic.AddInt64(&Misses, 1)
	}
	return val, err
}


func Set(key string, value string, ttl time.Duration) error {
	if useInMemory {
		mu.Lock()
		defer mu.Unlock()
		store[key] = value
		return nil
	}
	return Rdb.Set(Ctx, key, value, ttl).Err()
}

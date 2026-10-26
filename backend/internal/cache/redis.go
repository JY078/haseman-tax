package cache

import (
	"context"
	"log"
	"time"

	"advertising-image/backend/internal/config"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func Connect() {
	c := redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.Redis.Addr,
		Password: config.AppConfig.Redis.Password,
		DB:       config.AppConfig.Redis.DB,

		// Connection pool tuned for a 2-core 2 GB server
		PoolSize:        20,            // max active connections
		MinIdleConns:    5,             // keep warm connections ready
		PoolTimeout:     3 * time.Second,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 30 * time.Minute,

		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	_, err := c.Ping(context.Background()).Result()
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v. Caching will be disabled.", err)
		// Leave RDB nil so callers can guard with `if cache.RDB == nil`
		return
	}

	RDB = c
	log.Println("Connected to Redis successfully")
}

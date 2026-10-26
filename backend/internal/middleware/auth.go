package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/cache"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/utils"

	"github.com/gin-gonic/gin"
)


func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		tokenHash := utils.HashToken(tokenString)

		// Fast path: Redis cache hit skips the DB round-trip entirely.
		if cache.RDB != nil {
			val, redisErr := cache.RDB.Get(context.Background(), cache.SessionKey(tokenHash)).Result()
			if redisErr == nil && val == "1" {
				cache.RDB.Expire(context.Background(), cache.SessionKey(tokenHash), cache.SessionCacheTTL)
				c.Set("userID", claims.UserID)
				c.Set("username", claims.Username)
				c.Next()
				return
			}
		}

		// Slow path: verify session in DB
		var session model.UserSession
		if err := database.DB.Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).
			First(&session).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or logged out from another device"})
			c.Abort()
			return
		}

		// Warm the cache so next request takes the fast path
		cache.CacheSession(tokenHash)

		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

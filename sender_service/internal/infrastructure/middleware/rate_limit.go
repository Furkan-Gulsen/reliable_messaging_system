package middleware

import (
	"net/http"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ratelimit"

	"github.com/gin-gonic/gin"
)

func RateLimit(rps float64, burst int) gin.HandlerFunc {
	limiter := ratelimit.NewRateLimiter(rps, burst)
	
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}
		c.Next()
	}
} 
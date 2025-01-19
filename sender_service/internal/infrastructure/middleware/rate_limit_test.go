package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		rps           float64
		burst         int
		requests      int
		expectedCodes []int
	}{
		{
			name:      "allow requests within limit",
			rps:       10,
			burst:     2,
			requests:  2,
			expectedCodes: []int{
				http.StatusOK,
				http.StatusOK,
			},
		},
		{
			name:      "rate limit exceeded",
			rps:       1,
			burst:     1,
			requests:  3,
			expectedCodes: []int{
				http.StatusOK,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RateLimit(tt.rps, tt.burst))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			for i := 0; i < tt.requests; i++ {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, tt.expectedCodes[i], w.Code)
			}
		})
	}
}

func TestRateLimit_Recovery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimit(1, 1)) 
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	time.Sleep(1100 * time.Millisecond)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
} 
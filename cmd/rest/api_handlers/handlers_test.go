package apihandlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/avatar31/halmidi/test/unit/helpers"
)

func TestHealthHandler(t *testing.T) {
	router, apiGroup := getApiGroupRouter()
	apiGroup.GET("/health", HealthHandler)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected status code 200")
}

func TestConcurrentRequests(t *testing.T) {
	router, apiGroup := getApiGroupRouter()
	apiGroup.GET("/health", HealthHandler)

	concurrentRequests := 10
	results := make(chan int, concurrentRequests)

	// Launch concurrent requests
	for i := range concurrentRequests {
		go func(id int) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
			req.Header.Set("x-test-id", string(rune(id)))
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			results <- w.Code
		}(i)
	}

	// Collect and verify results
	for range concurrentRequests {
		statusCode := <-results
		assert.Equal(t, http.StatusOK, statusCode, "Expected all concurrent requests to succeed")
	}
}

// BenchmarkHealthEndpoint benchmarks the health endpoint
func BenchmarkHealthEndpoint(b *testing.B) {
	router, apiGroup := getApiGroupRouter()
	apiGroup.GET("/health", HealthHandler)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func getApiGroupRouter() (*gin.Engine, *gin.RouterGroup) {
	_, router := testhelpers.NewGinTestContext()
	return router, router.Group("/api/v1")
}

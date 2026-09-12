package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	s3handlers "github.com/avatar31/halmidi/cmd/halmidi/rest/s3_handlers"
	testhelpers "github.com/avatar31/halmidi/test/unit/helpers"
)

func TestWithLoggingAndPanicRecovery(t *testing.T) {
	_, router := testhelpers.NewGinTestContext()
	router.Use(WithLoggingAndPanicRecovery())
	router.GET("/panic", func(c *gin.Context) { panic("intentional panic for testing") })
	router.GET("/api/v1/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	t.Run("Should recover from panic and return 500 status", func(t *testing.T) {
		// Perform test request
		req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code, "Expected 500 status after panic")

		var response map[string]string
		testhelpers.DecodeJSONResponseBody(t, w, &response)
		assert.Equal(t, "Internal Server Error", response["error"])
	})

	t.Run("Should send x-amz-request-id in response headers if not exist in request headers", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Expected 200 status")
		assert.NotEmpty(t, w.Header().Get("x-amz-request-id"), "Expected x-amz-request-id in response")
	})

	t.Run("Should send x-amz-request-id in response headers same as request headers", func(t *testing.T) {
		traceId := "test-trace-id-12345"
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.Header.Set(s3handlers.AMZ_REQUEST_ID, traceId)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Expected 200 status")
		assert.Equal(t, traceId, w.Header().Get(s3handlers.AMZ_REQUEST_ID), "Expected x-amz-request-id in response")
	})
}

// func TestWithSigV4Authentication(t *testing.T) {
// 	_, router := testhelpers.NewGinTestContext()
// 	router.Use(
// 		WithLoggingAndPanicRecovery(),
// 		WithSigV4Authentication(),
// 	)
// 	router.GET("/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"data": "secure information"}) })

// 	req, _ := http.NewRequest(http.MethodGet, "/", nil)
// 	req.Header.Set("Authorization", "Bearer valid-token")
// 	w := httptest.NewRecorder()
// 	router.ServeHTTP(w, req)

// 	assert.Equal(t, http.StatusForbidden, w.Code, "Expected 403 status for valid authentication")

// 	var errResp s3nativeapihandlers.IAMErrorResponse
// 	testhelpers.DecodeXMLResponse(t, w, &errResp)
// 	assert.Equal(t, s3common.AccessDenied, errResp.Error.Code, "Expected AccessDenied error code")
// }

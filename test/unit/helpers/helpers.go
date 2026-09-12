package testhelpers

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestContext creates a test context with logger
func TestContext() context.Context {
	return context.Background()
}

// NewGinTestContext creates a Gin context for testing
func NewGinTestContext() (*gin.Context, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, e := gin.CreateTestContext(w)
	return c, e
}

func NewMockRequest(method, url string, headers map[string]string, body io.Reader, params,
	queryParams map[string]string,
) *http.Request {
	c, _ := NewGinTestContext()
	req, _ := http.NewRequestWithContext(TestContext(), method, url, body)
	c.Request = req

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	for key, value := range params {
		c.Params = append(c.Params, gin.Param{Key: key, Value: value})
	}

	q := req.URL.Query()
	for key, value := range queryParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	return req
}

func DecodeXMLResponse(t *testing.T, w *httptest.ResponseRecorder, result any) {
	decoder := xml.NewDecoder(w.Body)
	err := decoder.Decode(result)
	require.NoError(t, err, "Failed to parse error response")
}

func EncodeJSONRequestBody(body any) io.Reader {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	_ = encoder.Encode(body)
	return &buf
}

func DecodeJSONResponseBody(t *testing.T, w *httptest.ResponseRecorder, result any) {
	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(result)
	require.NoError(t, err, "Failed to parse error response")
}

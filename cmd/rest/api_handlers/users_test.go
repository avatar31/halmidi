package apihandlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/avatar31/halmidi/internal/core/iam"
	"github.com/avatar31/halmidi/test/unit/helpers"
)

func TestCreateUserHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test as database not initialized")
	}

	router, apiGroup := getApiGroupRouter()
	apiGroup.POST("/users", CreateUserHandler)

	tests := []struct {
		name           string
		requestBody    any
		expectedStatus int
	}{
		{
			name: "Valid_user_creation",
			requestBody: iam.CreateUserRequest{
				UserName: "testuser",
				Path:     "/",
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := testhelpers.EncodeJSONRequestBody(tc.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/users", body)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code, fmt.Sprintf("Expected status code %d", tc.expectedStatus))
		})
	}
}

func TestCreateAccessKeyHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test as database not initialized")
	}

	router, apiGroup := getApiGroupRouter()
	apiGroup.POST("/users/:userName/accesskey", CreateAccessKeyHandler)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "Valid_user_accesskey_creation",
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/testuser/accesskey", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code, fmt.Sprintf("Expected status code %d", tc.expectedStatus))
		})
	}
}

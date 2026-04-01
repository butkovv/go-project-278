package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetPing(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "get ping success",
			url:            "/ping",
			expectedStatus: http.StatusOK,
			expectedBody:   "pong!\n",
		},
		{
			name:           "get ping not found",
			url:            "/pingg",
			expectedStatus: http.StatusNotFound,
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := SetupRouter()

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			if res.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, res.Code)
			}

			if tt.expectedBody != "" && res.Body.String() != tt.expectedBody {
				t.Fatalf("expected body %q, got %q", tt.expectedBody, res.Body.String())
			}
		})
	}
}

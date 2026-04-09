package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	db "url-shortener/db/generated"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func setupTestRouterWithRedirectMock(t *testing.T) (*gin.Engine, sqlmock.Sqlmock) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}

	h := &RedirectHandler{Queries: db.New(dbConn)}
	r := gin.New()
	redirect := r.Group("/r")
	h.Register(redirect)

	t.Cleanup(func() {
		_ = dbConn.Close()
	})

	return r, mock
}

func TestRedirect(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedTarget string
		setup          func(mock sqlmock.Sqlmock)
		requestSetup   func(req *http.Request)
	}{
		{
			name:           "redirect success",
			url:            "/r/hexlet",
			expectedStatus: http.StatusFound,
			expectedTarget: "https://example.com",
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url", "created_at"}).
					AddRow(int64(1), "https://example.com", "hexlet", "https://short.local/hexlet", time.Now())
				mock.ExpectQuery("SELECT(.*)FROM links").WithArgs("hexlet").WillReturnRows(rows)
				mock.ExpectExec("INSERT INTO link_visits").
					WithArgs(int64(1), "203.0.113.10", "curl/8.0", int32(http.StatusFound)).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			requestSetup: func(req *http.Request) {
				req.RemoteAddr = "203.0.113.10:4567"
				req.Header.Set("User-Agent", "curl/8.0")
			},
		},
		{
			name:           "redirect link not found",
			url:            "/r/missing",
			expectedStatus: http.StatusNotFound,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT(.*)FROM links").WithArgs("missing").WillReturnError(sql.ErrNoRows)
			},
		},
		{
			name:           "redirect visit insert failure",
			url:            "/r/hexlet",
			expectedStatus: http.StatusInternalServerError,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url", "created_at"}).
					AddRow(int64(1), "https://example.com", "hexlet", "https://short.local/hexlet", time.Now())
				mock.ExpectQuery("SELECT(.*)FROM links").WithArgs("hexlet").WillReturnRows(rows)
				mock.ExpectExec("INSERT INTO link_visits").
					WithArgs(int64(1), "203.0.113.11", "Mozilla/5.0", int32(http.StatusFound)).
					WillReturnError(sql.ErrConnDone)
			},
			requestSetup: func(req *http.Request) {
				req.RemoteAddr = "203.0.113.11:4567"
				req.Header.Set("User-Agent", "Mozilla/5.0")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mock := setupTestRouterWithRedirectMock(t)
			if tt.setup != nil {
				tt.setup(mock)
			}

			req, _ := http.NewRequest(http.MethodGet, tt.url, nil)
			if tt.requestSetup != nil {
				tt.requestSetup(req)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedTarget != "" && w.Header().Get("Location") != tt.expectedTarget {
				t.Fatalf("expected Location %q, got %q", tt.expectedTarget, w.Header().Get("Location"))
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet db expectations: %v", err)
			}
		})
	}
}

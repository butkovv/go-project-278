package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	db "url-shortener/db/generated"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func assertJSONFields(t *testing.T, body string, expected map[string]any) {
	t.Helper()

	payload := map[string]any{}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	for key, value := range expected {
		if payload[key] != value {
			t.Fatalf("expected %s %v, got %v", key, value, payload[key])
		}
	}
}

func setupTestRouterWithMock(t *testing.T) (*gin.Engine, sqlmock.Sqlmock) {
	t.Helper()

	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	t.Setenv("APP_HOST", "https://short.local")

	gin.SetMode(gin.TestMode)

	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}

	h := &LinkHandler{Queries: db.New(dbConn)}
	r := gin.New()
	api := r.Group("/api")
	links := api.Group("/links")
	h.Register(links)

	t.Cleanup(func() {
		_ = dbConn.Close()
	})

	return r, mock
}

func TestCreateLink(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedStatus int
		setup          func(mock sqlmock.Sqlmock)
		assertBody     func(t *testing.T, body string)
	}{
		{
			name:           "create link success",
			body:           `{"original_url":"https://example.com","short_name":"hexlet"}`,
			expectedStatus: http.StatusCreated,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url", "created_at"}).
					AddRow(int64(1), "https://example.com", "hexlet", "https://short.local/hexlet", time.Now())
				mock.ExpectQuery("INSERT INTO links").
					WithArgs("https://example.com", "hexlet", "https://short.local/hexlet").
					WillReturnRows(rows)
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				payload := map[string]any{}
				if err := json.Unmarshal([]byte(body), &payload); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if payload["id"] != float64(1) {
					t.Fatalf("expected id 1, got %v", payload["id"])
				}
				if payload["original_url"] != "https://example.com" {
					t.Fatalf("expected original_url https://example.com, got %v", payload["original_url"])
				}
				if payload["short_name"] != "hexlet" {
					t.Fatalf("expected short_name hexlet, got %v", payload["short_name"])
				}
				if payload["short_url"] != "https://short.local/hexlet" {
					t.Fatalf("expected short_url https://short.local/hexlet, got %v", payload["short_url"])
				}
			},
		},
		{
			name:           "create link invalid json",
			body:           `{"original_url":}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "create link missing original_url",
			body:           `{"short_name":"hexlet"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "create link invalid url",
			body:           `{"original_url":"bad-url","short_name":"hexlet"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mock := setupTestRouterWithMock(t)

			if tt.setup != nil {
				tt.setup(mock)
			}

			req, _ := http.NewRequest(http.MethodPost, "/api/links", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.assertBody != nil {
				tt.assertBody(t, w.Body.String())
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet db expectations: %v", err)
			}
		})
	}
}

func TestGetLink(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedStatus int
		setup          func(mock sqlmock.Sqlmock)
		assertBody     func(t *testing.T, body string)
	}{
		{
			name:           "get link success",
			url:            "/api/links/1",
			expectedStatus: http.StatusOK,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url"}).
					AddRow(int64(1), "https://example.com", "hexlet", "https://short.local/hexlet")
				mock.ExpectQuery("SELECT(.*)FROM links").WithArgs(int64(1)).WillReturnRows(rows)
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertJSONFields(t, body, map[string]any{
					"id":           float64(1),
					"original_url": "https://example.com",
					"short_name":   "hexlet",
					"short_url":    "https://short.local/hexlet",
				})
			},
		},
		{
			name:           "get link not found",
			url:            "/api/links/999",
			expectedStatus: http.StatusNotFound,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT(.*)FROM links").WithArgs(int64(999)).WillReturnError(sql.ErrNoRows)
			},
		},
		{
			name:           "get link invalid id",
			url:            "/api/links/abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "get link negative id",
			url:            "/api/links/-1",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mock := setupTestRouterWithMock(t)

			if tt.setup != nil {
				tt.setup(mock)
			}

			req, _ := http.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.assertBody != nil {
				tt.assertBody(t, w.Body.String())
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet db expectations: %v", err)
			}
		})
	}
}

func TestListLinks(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
		setup          func(mock sqlmock.Sqlmock)
		assertBody     func(t *testing.T, body string)
	}{
		{
			name:           "list links empty",
			expectedStatus: http.StatusOK,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url", "created_at"})
				mock.ExpectQuery("SELECT(.*)FROM links").WillReturnRows(rows)
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				if body != "null" {
					t.Fatalf("expected body %q, got %q", "null", body)
				}
			},
		},
		{
			name:           "list links with records",
			expectedStatus: http.StatusOK,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url", "created_at"}).
					AddRow(int64(2), "https://b.example", "b", "https://short.local/b", time.Now()).
					AddRow(int64(1), "https://a.example", "a", "https://short.local/a", time.Now())
				mock.ExpectQuery("SELECT(.*)FROM links").WillReturnRows(rows)
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				var payload []map[string]any
				if err := json.Unmarshal([]byte(body), &payload); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if len(payload) != 2 {
					t.Fatalf("expected 2 links, got %d", len(payload))
				}
				if payload[0]["id"] != float64(2) || payload[0]["short_name"] != "b" {
					t.Fatalf("unexpected first link payload: %v", payload[0])
				}
				if payload[1]["id"] != float64(1) || payload[1]["short_name"] != "a" {
					t.Fatalf("unexpected second link payload: %v", payload[1])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mock := setupTestRouterWithMock(t)

			tt.setup(mock)

			req, _ := http.NewRequest(http.MethodGet, "/api/links", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.assertBody != nil {
				tt.assertBody(t, w.Body.String())
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet db expectations: %v", err)
			}
		})
	}
}

func TestUpdateLink(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		body           string
		expectedStatus int
		setup          func(mock sqlmock.Sqlmock)
		assertBody     func(t *testing.T, body string)
	}{
		{
			name:           "update link success",
			url:            "/api/links/1",
			body:           `{"original_url":"https://example.com/new","short_name":"new"}`,
			expectedStatus: http.StatusOK,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url", "created_at"}).
					AddRow(int64(1), "https://example.com/new", "new", "https://short.local/new", time.Now())
				mock.ExpectQuery("UPDATE links").
					WithArgs(int64(1), "https://example.com/new", "new", "https://short.local/new").
					WillReturnRows(rows)
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertJSONFields(t, body, map[string]any{
					"id":           float64(1),
					"original_url": "https://example.com/new",
					"short_name":   "new",
					"short_url":    "https://short.local/new",
				})
			},
		},
		{
			name:           "update link invalid id",
			url:            "/api/links/abc",
			body:           `{"original_url":"https://example.com/new","short_name":"new"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "update link invalid body",
			url:            "/api/links/1",
			body:           `{"original_url":}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "update link not found",
			url:            "/api/links/999",
			body:           `{"original_url":"https://example.com/new","short_name":"new"}`,
			expectedStatus: http.StatusNotFound,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("UPDATE links").
					WithArgs(int64(999), "https://example.com/new", "new", "https://short.local/new").
					WillReturnError(sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mock := setupTestRouterWithMock(t)

			if tt.setup != nil {
				tt.setup(mock)
			}

			req, _ := http.NewRequest(http.MethodPut, tt.url, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.assertBody != nil {
				tt.assertBody(t, w.Body.String())
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet db expectations: %v", err)
			}
		})
	}
}

func TestDeleteLink(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedStatus int
		setup          func(mock sqlmock.Sqlmock)
	}{
		{
			name:           "delete link success",
			url:            "/api/links/1",
			expectedStatus: http.StatusNoContent,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM links").WithArgs(int64(1)).WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:           "delete link invalid id",
			url:            "/api/links/abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "delete link negative id",
			url:            "/api/links/-1",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mock := setupTestRouterWithMock(t)

			if tt.setup != nil {
				tt.setup(mock)
			}

			req, _ := http.NewRequest(http.MethodDelete, tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet db expectations: %v", err)
			}
		})
	}
}

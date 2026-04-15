package handlers

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	db "url-shortener/internal/db/generated"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

type nonEmptyStringArg struct{}

func (a nonEmptyStringArg) Match(v driver.Value) bool {
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) != ""
}

type stringPrefixArg struct {
	prefix string
}

func (a stringPrefixArg) Match(v driver.Value) bool {
	s, ok := v.(string)
	return ok && strings.HasPrefix(s, a.prefix)
}

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

func assertValidationFieldError(t *testing.T, body, field, message string) {
	t.Helper()

	payload := map[string]any{}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	errorsMap, ok := payload["errors"].(map[string]any)
	if !ok {
		t.Fatalf("expected errors object, got %T", payload["errors"])
	}

	if errorsMap[field] != message {
		t.Fatalf("expected errors[%s] = %q, got %v", field, message, errorsMap[field])
	}
}

func assertCreatedAtPresent(t *testing.T, payload map[string]any) {
	t.Helper()

	createdAt, ok := payload["created_at"]
	if !ok {
		t.Fatal("expected created_at field in response")
	}

	switch v := createdAt.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			t.Fatalf("expected non-empty created_at string, got %q", v)
		}
	case map[string]any:
		if v["Valid"] != true {
			t.Fatalf("expected created_at.Valid true, got %v", v["Valid"])
		}
	default:
		t.Fatalf("expected created_at string or object, got %T", createdAt)
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
					AddRow(int64(1), "https://example.com", "hexlet", "https://short.local/r/hexlet", time.Now())
				mock.ExpectQuery("INSERT INTO links").
					WithArgs("https://example.com", "hexlet", "https://short.local/r/hexlet").
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
				if payload["short_url"] != "https://short.local/r/hexlet" {
					t.Fatalf("expected short_url https://short.local/r/hexlet, got %v", payload["short_url"])
				}
				assertCreatedAtPresent(t, payload)
			},
		},
		{
			name:           "create link generates short_name when omitted",
			body:           `{"original_url":"https://example.com"}`,
			expectedStatus: http.StatusCreated,
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url", "created_at"}).
					AddRow(int64(2), "https://example.com", "generated-name", "https://short.local/r/generated-name", time.Now())
				mock.ExpectQuery("INSERT INTO links").
					WithArgs("https://example.com", nonEmptyStringArg{}, stringPrefixArg{prefix: "https://short.local/r/"}).
					WillReturnRows(rows)
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				payload := map[string]any{}
				if err := json.Unmarshal([]byte(body), &payload); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				shortName, ok := payload["short_name"].(string)
				if !ok || strings.TrimSpace(shortName) == "" {
					t.Fatalf("expected generated non-empty short_name, got %v", payload["short_name"])
				}
				shortURL, ok := payload["short_url"].(string)
				if !ok || !strings.HasSuffix(shortURL, "/r/"+shortName) {
					t.Fatalf("expected short_url to end with /r/%s, got %v", shortName, payload["short_url"])
				}
				assertCreatedAtPresent(t, payload)
			},
		},
		{
			name:           "create link invalid json",
			body:           `{"original_url":}`,
			expectedStatus: http.StatusBadRequest,
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertJSONFields(t, body, map[string]any{"error": "invalid request"})
			},
		},
		{
			name:           "create link missing original_url",
			body:           `{"short_name":"hexlet"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertValidationFieldError(t, body, "original_url", "is required")
			},
		},
		{
			name:           "create link invalid url",
			body:           `{"original_url":"bad-url","short_name":"hexlet"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertValidationFieldError(t, body, "original_url", "must be a valid URL")
			},
		},
		{
			name:           "create link short_name too short",
			body:           `{"original_url":"https://example.com","short_name":"ab"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertValidationFieldError(t, body, "short_name", "must be at least 3 characters")
			},
		},
		{
			name:           "create link short_name too long",
			body:           `{"original_url":"https://example.com","short_name":"abcdefghijklmnopqrstuvwxyz1234567"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertValidationFieldError(t, body, "short_name", "must be at most 32 characters")
			},
		},
		{
			name:           "create link duplicate short_name",
			body:           `{"original_url":"https://example.com","short_name":"hexlet"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO links").
					WithArgs("https://example.com", "hexlet", "https://short.local/r/hexlet").
					WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "links_short_name_key"})
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertValidationFieldError(t, body, "short_name", "has already been taken")
			},
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
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url", "created_at"}).
					AddRow(int64(1), "https://example.com", "hexlet", "https://short.local/r/hexlet", time.Now())
				mock.ExpectQuery("SELECT(.*)FROM links").WithArgs(int64(1)).WillReturnRows(rows)
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertJSONFields(t, body, map[string]any{
					"id":           float64(1),
					"original_url": "https://example.com",
					"short_name":   "hexlet",
					"short_url":    "https://short.local/r/hexlet",
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
		url            string
		expectedStatus int
		expectedRange  string
		setup          func(mock sqlmock.Sqlmock)
		assertBody     func(t *testing.T, body string)
	}{
		{
			name:           "list links empty",
			url:            "/api/links",
			expectedStatus: http.StatusOK,
			expectedRange:  "links 0-0/0",
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url", "created_at"})
				mock.ExpectQuery("SELECT(.*)FROM links").WithArgs(int32(10), int32(0)).WillReturnRows(rows)
				mock.ExpectQuery(`SELECT count\(\*\) AS total_count FROM links`).WillReturnRows(sqlmock.NewRows([]string{"total_count"}).AddRow(int64(0)))
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
			url:            "/api/links",
			expectedStatus: http.StatusOK,
			expectedRange:  "links 0-2/2",
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url", "created_at"}).
					AddRow(int64(2), "https://b.example", "b", "https://short.local/r/b", time.Now()).
					AddRow(int64(1), "https://a.example", "a", "https://short.local/r/a", time.Now())
				mock.ExpectQuery("SELECT(.*)FROM links").WithArgs(int32(10), int32(0)).WillReturnRows(rows)
				mock.ExpectQuery(`SELECT count\(\*\) AS total_count FROM links`).WillReturnRows(sqlmock.NewRows([]string{"total_count"}).AddRow(int64(2)))
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
		{
			name:           "list links with custom pagination range",
			url:            "/api/links?range=[1,3]",
			expectedStatus: http.StatusOK,
			expectedRange:  "links 1-3/5",
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "original_url", "short_name", "short_url", "created_at"}).
					AddRow(int64(4), "https://d.example", "d", "https://short.local/r/d", time.Now()).
					AddRow(int64(3), "https://c.example", "c", "https://short.local/r/c", time.Now())
				mock.ExpectQuery("SELECT(.*)FROM links").WithArgs(int32(2), int32(1)).WillReturnRows(rows)
				mock.ExpectQuery(`SELECT count\(\*\) AS total_count FROM links`).WillReturnRows(sqlmock.NewRows([]string{"total_count"}).AddRow(int64(5)))
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
				if payload[0]["id"] != float64(4) || payload[1]["id"] != float64(3) {
					t.Fatalf("unexpected pagination payload: %v", payload)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mock := setupTestRouterWithMock(t)

			tt.setup(mock)

			req, _ := http.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.assertBody != nil {
				tt.assertBody(t, w.Body.String())
			}

			if w.Header().Get("Content-Range") != tt.expectedRange {
				t.Fatalf("expected Content-Range %q, got %q", tt.expectedRange, w.Header().Get("Content-Range"))
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
					AddRow(int64(1), "https://example.com/new", "new", "https://short.local/r/new", time.Now())
				mock.ExpectQuery("UPDATE links").
					WithArgs(int64(1), "https://example.com/new", "new", "https://short.local/r/new").
					WillReturnRows(rows)
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertJSONFields(t, body, map[string]any{
					"id":           float64(1),
					"original_url": "https://example.com/new",
					"short_name":   "new",
					"short_url":    "https://short.local/r/new",
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
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertJSONFields(t, body, map[string]any{"error": "invalid request"})
			},
		},
		{
			name:           "update link missing original_url",
			url:            "/api/links/1",
			body:           `{"short_name":"new"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertValidationFieldError(t, body, "original_url", "is required")
			},
		},
		{
			name:           "update link invalid url",
			url:            "/api/links/1",
			body:           `{"original_url":"bad-url","short_name":"new"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertValidationFieldError(t, body, "original_url", "must be a valid URL")
			},
		},
		{
			name:           "update link short_name too short",
			url:            "/api/links/1",
			body:           `{"original_url":"https://example.com/new","short_name":"ab"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertValidationFieldError(t, body, "short_name", "must be at least 3 characters")
			},
		},
		{
			name:           "update link short_name too long",
			url:            "/api/links/1",
			body:           `{"original_url":"https://example.com/new","short_name":"abcdefghijklmnopqrstuvwxyz1234567"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertValidationFieldError(t, body, "short_name", "must be at most 32 characters")
			},
		},
		{
			name:           "update link not found",
			url:            "/api/links/999",
			body:           `{"original_url":"https://example.com/new","short_name":"new"}`,
			expectedStatus: http.StatusNotFound,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("UPDATE links").
					WithArgs(int64(999), "https://example.com/new", "new", "https://short.local/r/new").
					WillReturnError(sql.ErrNoRows)
			},
		},
		{
			name:           "update link duplicate short_name",
			url:            "/api/links/1",
			body:           `{"original_url":"https://example.com/new","short_name":"new"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("UPDATE links").
					WithArgs(int64(1), "https://example.com/new", "new", "https://short.local/r/new").
					WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "links_short_name_key"})
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				assertValidationFieldError(t, body, "short_name", "has already been taken")
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

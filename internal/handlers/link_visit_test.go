package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	db "url-shortener/internal/db/generated"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func setupTestRouterWithLinkVisitsMock(t *testing.T) (*gin.Engine, sqlmock.Sqlmock) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}

	h := &LinkVisitHandler{Queries: db.New(dbConn)}
	r := gin.New()
	api := r.Group("/api")
	linkVisits := api.Group("/link_visits")
	h.Register(linkVisits)

	t.Cleanup(func() {
		_ = dbConn.Close()
	})

	return r, mock
}

func TestListLinkVisits(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedRange  string
		setup          func(mock sqlmock.Sqlmock)
		assertBody     func(t *testing.T, body string)
	}{
		{
			name:           "list link visits empty",
			url:            "/api/link_visits",
			expectedStatus: http.StatusOK,
			expectedRange:  "link_visits 0-0/0",
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "link_id", "created_at", "ip", "user_agent", "status"})
				mock.ExpectQuery("SELECT(.*)FROM link_visits").WithArgs(int32(10), int32(0)).WillReturnRows(rows)
				mock.ExpectQuery(`SELECT count\(\*\) AS total_count FROM link_visits`).WillReturnRows(sqlmock.NewRows([]string{"total_count"}).AddRow(int64(0)))
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				if body != "null" {
					t.Fatalf("expected body %q, got %q", "null", body)
				}
			},
		},
		{
			name:           "list link visits with records",
			url:            "/api/link_visits",
			expectedStatus: http.StatusOK,
			expectedRange:  "link_visits 0-2/2",
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "link_id", "created_at", "ip", "user_agent", "status"}).
					AddRow(int64(2), int64(1), time.Now(), "203.0.113.2", "curl/8.0", int32(302)).
					AddRow(int64(1), int64(1), time.Now(), "203.0.113.1", "Mozilla/5.0", int32(302))
				mock.ExpectQuery("SELECT(.*)FROM link_visits").WithArgs(int32(10), int32(0)).WillReturnRows(rows)
				mock.ExpectQuery(`SELECT count\(\*\) AS total_count FROM link_visits`).WillReturnRows(sqlmock.NewRows([]string{"total_count"}).AddRow(int64(2)))
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				var payload []map[string]any
				if err := json.Unmarshal([]byte(body), &payload); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if len(payload) != 2 {
					t.Fatalf("expected 2 link visits, got %d", len(payload))
				}
				if payload[0]["id"] != float64(2) || payload[0]["link_id"] != float64(1) {
					t.Fatalf("unexpected first link visit payload: %v", payload[0])
				}
			},
		},
		{
			name:           "list link visits with custom pagination range",
			url:            "/api/link_visits?range=[1,3]",
			expectedStatus: http.StatusOK,
			expectedRange:  "link_visits 1-3/5",
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "link_id", "created_at", "ip", "user_agent", "status"}).
					AddRow(int64(4), int64(2), time.Now(), "198.51.100.4", "curl/8.0", int32(302)).
					AddRow(int64(3), int64(2), time.Now(), "198.51.100.3", "Mozilla/5.0", int32(302))
				mock.ExpectQuery("SELECT(.*)FROM link_visits").WithArgs(int32(2), int32(1)).WillReturnRows(rows)
				mock.ExpectQuery(`SELECT count\(\*\) AS total_count FROM link_visits`).WillReturnRows(sqlmock.NewRows([]string{"total_count"}).AddRow(int64(5)))
			},
			assertBody: func(t *testing.T, body string) {
				t.Helper()
				var payload []map[string]any
				if err := json.Unmarshal([]byte(body), &payload); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if len(payload) != 2 {
					t.Fatalf("expected 2 link visits, got %d", len(payload))
				}
			},
		},
		{
			name:           "list link visits db failure",
			url:            "/api/link_visits",
			expectedStatus: http.StatusInternalServerError,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT(.*)FROM link_visits").WithArgs(int32(10), int32(0)).WillReturnError(sql.ErrConnDone)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mock := setupTestRouterWithLinkVisitsMock(t)

			if tt.setup != nil {
				tt.setup(mock)
			}

			req, _ := http.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedRange != "" && w.Header().Get("Content-Range") != tt.expectedRange {
				t.Fatalf("expected Content-Range %q, got %q", tt.expectedRange, w.Header().Get("Content-Range"))
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

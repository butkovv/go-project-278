package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"
	db "url-shortener/db/generated"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func SetupRouter(database *sql.DB) *gin.Engine {
	queries := db.New(database)
	h := NewLinksHandler(queries)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	r := gin.New()

	r.Use(func(c *gin.Context) {
		start := time.Now()

		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
		}
		c.Header("X-Reques-ID", rid)

		c.Set("request_id", rid)

		c.Next()
		slog.Info("request handled",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", time.Since(start),
			"request_id", rid,
		)
	})

	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		rid := c.GetString("request_id")

		slog.Error("panic recovered",
			"error", recovered,
			"request_id", rid,
			"path", c.Request.URL.Path,
		)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
			"id":    rid,
		})
	}))

	api := r.Group("/api")
	//h.Register(api)
	links := api.Group("links")
	h.Register(links)

	return r
}

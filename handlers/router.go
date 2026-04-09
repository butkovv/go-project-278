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

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	r := gin.New()

	r.Use(func(c *gin.Context) {
		start := time.Now()

		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
		}
		c.Header("X-Request-ID", rid)

		c.Set("request_id", rid)

		slog.Info("proxy headers",
			"cf_connecting_ip", c.GetHeader("CF-Connecting-IP"),
			"x_forwarded_for", c.GetHeader("X-Forwarded-For"),
			"x_real_ip", c.GetHeader("X-Real-IP"),
			"remote_addr", c.Request.RemoteAddr,
			"client_ip", c.ClientIP(),
		)
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
	linksHandler := NewLinksHandler(queries)
	linkVisitsHandler := NewLinkVisitsHandler(queries)
	redirectHandler := NewRedirectHandler(queries)

	api := r.Group("/api")
	links := api.Group("/links")
	linkVisits := api.Group("/link_visits")

	linksHandler.Register(links)
	linkVisitsHandler.Register(linkVisits)

	redirect := r.Group("/r")
	redirectHandler.Register(redirect)

	r.TrustedPlatform = gin.PlatformCloudflare

	return r
}

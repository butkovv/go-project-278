package handlers

import (
	"log/slog"
	"net/http"
	db "url-shortener/internal/db/generated"

	"github.com/gin-gonic/gin"
)

type RedirectHandler struct {
	Queries *db.Queries
}

func NewRedirectHandler(queries *db.Queries) *RedirectHandler {
	return &RedirectHandler{Queries: queries}
}

func (h *RedirectHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/:code", h.Redirect)
}

func (h *RedirectHandler) Redirect(c *gin.Context) {
	code := c.Param("code")
	link, err := h.Queries.GetLinkByCode(c, code)
	slog.Info("requested", "code", code)
	if err != nil {
		handleDBError(c, err)
		return
	}

	linkVisit := db.CreateLinkVisitParams{
		LinkID:    link.ID,
		Ip:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Status:    http.StatusFound,
	}
	err = h.Queries.CreateLinkVisit(c, linkVisit)
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.Redirect(http.StatusFound, link.OriginalUrl)
}

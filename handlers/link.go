package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"url-shortener/config"
	db "url-shortener/db/generated"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/gin-gonic/gin"
)

type LinkParams struct {
	OriginalUrl string `json:"original_url" binding:"required,url,max=2048"`
	ShortName   string `json:"short_name" binding:"max=255"`
}

type LinkHandler struct {
	Queries *db.Queries
}

func NewLinksHandler(queries *db.Queries) *LinkHandler {
	return &LinkHandler{Queries: queries}
}

func (h *LinkHandler) Register(rg *gin.RouterGroup) {
	rg.POST("", h.Create)
	rg.GET("/:id", h.Get)
	rg.GET("", h.List)
	rg.PUT("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
}

func (h *LinkHandler) Create(c *gin.Context) {
	params := db.CreateLinkParams{}
	input, ok := h.parseAndValidateParams(c)
	if !ok {
		return
	}
	cfg, err := config.Load()
	if err != nil {
		slog.Error("error parsing config", "error", err)
		return
	}
	params.OriginalUrl = input.OriginalUrl
	params.ShortName = input.ShortName
	params.ShortUrl = fmt.Sprintf("%s/%s", cfg.AppHost, params.ShortName)
	fmt.Print(params)
	link, err := h.Queries.CreateLink(c, params)
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusCreated, link)
}

func (h *LinkHandler) Get(c *gin.Context) {
	id, err := h.parseID(c)
	if err != nil {
		badRequest(c, err)
		return
	}

	link, err := h.Queries.GetLinkById(c, id)
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusOK, link)
}

func (h *LinkHandler) List(c *gin.Context) {
	links, err := h.Queries.ListLinks(c)
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusOK, links)
}

func (h *LinkHandler) Update(c *gin.Context) {
	params := db.UpdateLinkParams{}

	id, err := h.parseID(c)
	if err != nil {
		badRequest(c, err)
		return
	}

	input, ok := h.parseAndValidateParams(c)
	if !ok {
		return
	}
	cfg, err := config.Load()
	if err != nil {
		slog.Error("error parsing config", "error", err)
		return
	}
	params.ID = id
	params.OriginalUrl = input.OriginalUrl
	params.ShortName = input.ShortName
	params.ShortUrl = fmt.Sprintf("%s/%s", cfg.AppHost, params.ShortName)
	link, err := h.Queries.UpdateLink(c, params)
	if err != nil {
		handleDBError(c, err)
	}
	c.JSON(http.StatusOK, link)
}

func (h *LinkHandler) Delete(c *gin.Context) {
	id, err := h.parseID(c)
	if err != nil {
		badRequest(c, err)
		return
	}
	err = h.Queries.DeleteLink(c, id)
	if err != nil {
		handleDBError(c, err)
	}
	c.Status(http.StatusNoContent)
}

func (h *LinkHandler) parseAndValidateParams(c *gin.Context) (LinkParams, bool) {
	params := LinkParams{}

	err := c.ShouldBindJSON(&params)
	if err != nil {
		badRequest(c, err)
		return params, false
	}

	params.OriginalUrl = strings.TrimSpace(params.OriginalUrl)
	if len(params.OriginalUrl) == 0 {
		badRequest(c, ErrorURLEmpty)
		return params, false
	}
	params.ShortName = strings.TrimSpace(params.ShortName)
	if len(params.ShortName) == 0 {
		params.ShortName = namesgenerator.GetRandomName(0)
	}
	return params, true
}

func (h *LinkHandler) parseID(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrorInvalidID
	}
	return id, nil
}

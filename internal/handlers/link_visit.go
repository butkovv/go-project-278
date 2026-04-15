package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	db "url-shortener/internal/db/generated"

	"github.com/gin-gonic/gin"
)

type LinkVisitHandler struct {
	Queries *db.Queries
}

func NewLinkVisitsHandler(queries *db.Queries) *LinkVisitHandler {
	return &LinkVisitHandler{Queries: queries}
}

func (h *LinkVisitHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
}

func (h *LinkVisitHandler) List(c *gin.Context) {
	pagination, err := h.getPaginationParams(c)
	if err != nil {
		badRequest(c, err)
		return
	}
	linkVisits, err := h.Queries.ListLinkVisits(c, pagination)
	if err != nil {
		handleDBError(c, err)
		return
	}
	total, err := h.Queries.GetTotalLinkVisitsCount(c)
	if err != nil {
		handleDBError(c, err)
		return
	}
	endRange := pagination.Limit + pagination.Offset
	if total < int64(endRange) {
		endRange = int32(total)
	}
	contentRange := fmt.Sprintf("link_visits %d-%d/%d", pagination.Offset, endRange, total)
	c.Header("Content-Range", contentRange)
	c.JSON(http.StatusOK, linkVisits)
}

func (h *LinkVisitHandler) getPaginationParams(c *gin.Context) (db.ListLinkVisitsParams, error) {
	pagination := db.ListLinkVisitsParams{Offset: 0, Limit: 10}
	rangeString := c.DefaultQuery("range", "")
	if strings.TrimSpace(rangeString) == "" {
		return pagination, nil
	}

	trimmedRangeString := strings.Trim(rangeString, "[]")
	parts := strings.Split(trimmedRangeString, ",")
	if len(parts) == 2 {
		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return pagination, ErrorInvalidRange
		}
		end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return pagination, ErrorInvalidRange
		}
		if start < 0 || end < start {
			return pagination, ErrorInvalidRange
		}
		pagination.Offset = int32(start)
		pagination.Limit = int32(end - start)
		return pagination, nil
	}

	if trimmedRangeString != "" {
		return pagination, ErrorInvalidRange
	}

	return pagination, nil
}

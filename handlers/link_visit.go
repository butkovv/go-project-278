package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	db "url-shortener/db/generated"

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
	pagination := h.getPaginationParams(c)
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

func (h *LinkVisitHandler) getPaginationParams(c *gin.Context) db.ListLinkVisitsParams {
	pagination := db.ListLinkVisitsParams{Offset: 0, Limit: 10}
	rangeString := c.DefaultQuery("range", "")
	trimmedRangeString := strings.Trim(rangeString, "[]")
	parts := strings.Split(trimmedRangeString, ",")
	if len(parts) == 2 {
		start, err := strconv.Atoi(parts[0])
		if err != nil {
			handleDBError(c, err)
			return pagination
		}
		end, err := strconv.Atoi(parts[1])
		if err != nil {
			handleDBError(c, err)
			return pagination
		}
		pagination.Offset = int32(start)
		pagination.Limit = int32(end - start)
	}
	return pagination
}

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PingHandler struct{}

func NewPingHandler() *PingHandler {
	return &PingHandler{}
}

func (h *PingHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.Respond)
}

func (h *PingHandler) Respond(c *gin.Context) {
	c.String(http.StatusOK, "pong!\n")
}

package handlers

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	handler := NewPingHandler()

	router := gin.Default()
	ping := router.Group("/ping")
	handler.Register(ping)

	return router
}

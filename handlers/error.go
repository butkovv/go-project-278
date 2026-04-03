package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	ErrorInvalidID   = errors.New("invalid id")
	ErrorURLEmpty    = errors.New("url cannot be empty")
	ErrorNameTooLong = errors.New("name is too long")
	ErrorLinkExists  = errors.New("link already exists")
)

func handleDBError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	if errors.Is(err, sql.ErrNoRows) {
		notFound(c)
		return
	}

	internalServerError(c, err)
}

func badRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error":   "Bad Request",
		"message": err.Error(),
	})
}

func notFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"error":   "Not Found",
		"message": "Resource not found",
	})
}

func internalServerError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "Internal Server Error",
		"message": "Something went wrong",
	})
}

func conflict(c *gin.Context, err error) {
	c.JSON(http.StatusConflict, gin.H{
		"error":   "Conflict",
		"message": err.Error(),
	})
}

func unprocessableEntity(c *gin.Context, err error) {
	c.JSON(http.StatusUnprocessableEntity, gin.H{
		"error":   "Unprocessable Entity",
		"message": err.Error(),
	})
}

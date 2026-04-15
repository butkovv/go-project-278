package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrorInvalidID    = errors.New("invalid id")
	ErrorInvalidRange = errors.New("invalid range")
	ErrorURLEmpty     = errors.New("url cannot be empty")
	ErrorNameTooLong  = errors.New("name is too long")
)

func handleDBError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	if errors.Is(err, sql.ErrNoRows) {
		notFound(c)
		return
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		field := "request"
		if pgErr.ConstraintName == "links_short_name_key" {
			field = "short_name"
		}
		validationErrors(c, map[string]string{field: "has already been taken"})
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

func invalidRequest(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": "invalid request",
	})
}

func validationErrors(c *gin.Context, fieldErrors map[string]string) {
	c.JSON(http.StatusUnprocessableEntity, gin.H{
		"errors": fieldErrors,
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
		"message": err.Error(),
	})
}

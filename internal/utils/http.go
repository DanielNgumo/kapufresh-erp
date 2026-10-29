package utils

import "github.com/gin-gonic/gin"

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func RespondData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

func RespondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": ErrorResponse{Code: code, Message: message}})
}

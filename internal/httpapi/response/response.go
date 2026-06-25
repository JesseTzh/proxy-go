package response

import "github.com/gin-gonic/gin"

type Envelope struct {
	OK    bool       `json:"ok"`
	Data  any        `json:"data,omitempty"`
	Error *ErrorBody `json:"error,omitempty"`
}

type ErrorBody struct {
	Message string `json:"message"`
}

func JSON(c *gin.Context, status int, data any) {
	c.JSON(status, Envelope{OK: true, Data: data})
}

func OK(c *gin.Context) {
	c.JSON(200, Envelope{OK: true})
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, Envelope{OK: false, Error: &ErrorBody{Message: message}})
}

func AbortError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, Envelope{OK: false, Error: &ErrorBody{Message: message}})
}

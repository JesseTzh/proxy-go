package api

import (
	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/httpapi"
)

type Deps = httpapi.Deps

func Router(d Deps) *gin.Engine {
	return httpapi.Router(d)
}

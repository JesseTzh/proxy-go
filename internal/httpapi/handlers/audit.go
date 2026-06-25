package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/httpapi/response"
	"github.com/proxy-go/proxy-go/internal/models"
)

func Capabilities(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []models.ProtocolCapability
		d.DB.Find(&items)
		response.JSON(c, 200, items)
	}
}

func AuditLogs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []models.AuditLog
		q := d.DB.Order("id desc").Limit(200)
		if v := c.Query("action"); v != "" {
			q = q.Where("action=?", v)
		}
		if v := c.Query("resourceType"); v != "" {
			q = q.Where("resource_type=?", v)
		}
		q.Find(&items)
		response.JSON(c, 200, items)
	}
}

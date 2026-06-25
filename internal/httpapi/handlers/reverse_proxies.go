package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/security"
	rpsvc "github.com/proxy-go/proxy-go/internal/services/reverseproxy"
)

func ListReverseProxies(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := rpsvc.New(d.DB).List()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, items)
	}
}

func CreateReverseProxy(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var item models.ReverseProxyRule
		if c.BindJSON(&item) != nil {
			c.JSON(400, gin.H{"error": "invalid json"})
			return
		}
		item, err := rpsvc.New(d.DB).Create(item)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if err := applyNginxAfterReverseProxyChange(c, d); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		d.Audit.Record("create_reverse_proxy", "reverse_proxy", fmt.Sprint(item.ID), item, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		c.JSON(200, item)
	}
}

func UpdateReverseProxy(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		var item models.ReverseProxyRule
		if c.BindJSON(&item) != nil {
			c.JSON(400, gin.H{"error": "invalid json"})
			return
		}
		item, err = rpsvc.New(d.DB).Update(id, item)
		if err != nil {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		if err := applyNginxAfterReverseProxyChange(c, d); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, item)
	}
}

func DeleteReverseProxy(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		if err := rpsvc.New(d.DB).Delete(id); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if err := applyNginxAfterReverseProxyChange(c, d); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		d.Audit.Record("delete_reverse_proxy", "reverse_proxy", fmt.Sprint(id), nil, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		c.JSON(200, gin.H{"ok": true})
	}
}

func SetReverseProxyEnabled(d Deps, enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		if err := rpsvc.New(d.DB).SetEnabled(id, enabled); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if err := applyNginxAfterReverseProxyChange(c, d); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	}
}

func applyNginxAfterReverseProxyChange(c *gin.Context, d Deps) error {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	if d.Nginx == nil {
		return fmt.Errorf("nginx service is not configured")
	}
	if err := d.Nginx.Apply(ctx); err != nil {
		return err
	}
	now := time.Now()
	return d.DB.Model(&models.SystemSetting{}).Where("id=1").Update("last_nginx_reload_at", &now).Error
}

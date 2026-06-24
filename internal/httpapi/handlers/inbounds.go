package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/security"
	inboundsvc "github.com/proxy-go/proxy-go/internal/services/inbounds"
	"github.com/proxy-go/proxy-go/internal/xray"
	"gorm.io/gorm"
)

func ListInbounds(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := inboundService(d).List()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, items)
	}
}

func CreateInbound(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req inboundsvc.CreateRequest
		if c.BindJSON(&req) != nil {
			c.JSON(400, gin.H{"error": "invalid json"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		item, err := inboundService(d).Create(ctx, req)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		d.Audit.Record("create_inbound", "inbound", fmt.Sprint(item.ID), item, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		c.JSON(200, item)
	}
}

func GetInbound(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		item, err := inboundService(d).Get(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, item)
	}
}

func UpdateInbound(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		var req inboundsvc.CreateRequest
		if c.BindJSON(&req) != nil {
			c.JSON(400, gin.H{"error": "invalid json"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		item, err := inboundService(d).Update(ctx, id, req)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, item)
	}
}

func DeleteInbound(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		if err := inboundService(d).Delete(id); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	}
}

func SetInboundEnabled(d Deps, enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		if err := inboundService(d).SetEnabled(id, enabled); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	}
}

func InboundConfig(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		cfg, err := inboundService(d).ConfigDetails(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, cfg)
	}
}

func InboundShare(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		share, err := inboundService(d).ShareDetails(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, share)
	}
}

func inboundService(d Deps) *inboundsvc.Service {
	return inboundsvc.New(d.DB, d.Cfg, xray.CLICredentialGenerator{Binary: d.Cfg.Runtime.XrayBinary})
}

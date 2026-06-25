package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/httpapi/response"
	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/security"
	inboundsvc "github.com/proxy-go/proxy-go/internal/services/inbounds"
	"github.com/proxy-go/proxy-go/internal/singbox"
	"gorm.io/gorm"
)

type inboundResponse struct {
	ID                     uint          `json:"id"`
	Name                   string        `json:"name"`
	Template               string        `json:"template"`
	Protocol               string        `json:"protocol"`
	DomainID               uint          `json:"domainId"`
	Domain                 models.Domain `json:"domain"`
	Network                string        `json:"network"`
	Security               string        `json:"security"`
	Flow                   string        `json:"flow"`
	RouteSNI               string        `json:"routeSni"`
	ListenAddr             string        `json:"listenAddr"`
	ListenPort             int           `json:"listenPort"`
	RealityHandshakeServer string        `json:"realityHandshakeServer"`
	Enabled                bool          `json:"enabled"`
}

func ListInbounds(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := inboundService(d).List()
		if err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.JSON(c, 200, inboundResponses(items))
	}
}

func CreateInbound(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req inboundsvc.CreateRequest
		if c.BindJSON(&req) != nil {
			response.Error(c, 400, "invalid json")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		item, err := inboundService(d).Create(ctx, req)
		if err != nil {
			response.Error(c, 400, err.Error())
			return
		}
		d.Audit.Record("create_inbound", "inbound", fmt.Sprint(item.ID), item, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.JSON(c, 200, toInboundResponse(item))
	}
}

func GetInbound(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		item, err := inboundService(d).Get(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "not found")
			return
		}
		if err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.JSON(c, 200, toInboundResponse(item))
	}
}

func UpdateInbound(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		var req inboundsvc.CreateRequest
		if c.BindJSON(&req) != nil {
			response.Error(c, 400, "invalid json")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		item, err := inboundService(d).Update(ctx, id, req)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "not found")
			return
		}
		if err != nil {
			response.Error(c, 400, err.Error())
			return
		}
		response.JSON(c, 200, toInboundResponse(item))
	}
}

func DeleteInbound(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		if err := inboundService(d).Delete(id); err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.OK(c)
	}
}

func SetInboundEnabled(d Deps, enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		if err := inboundService(d).SetEnabled(id, enabled); err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.OK(c)
	}
}

func InboundConfig(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		cfg, err := inboundService(d).ConfigDetails(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "not found")
			return
		}
		if err != nil {
			response.Error(c, 400, err.Error())
			return
		}
		response.JSON(c, 200, cfg)
	}
}

func InboundShare(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		share, err := inboundService(d).ShareDetails(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "not found")
			return
		}
		if err != nil {
			response.Error(c, 400, err.Error())
			return
		}
		response.JSON(c, 200, share)
	}
}

func inboundService(d Deps) *inboundsvc.Service {
	return inboundsvc.New(d.DB, d.Cfg, singbox.CLICredentialGenerator{Binary: d.Cfg.Runtime.SingBoxBinary})
}

func inboundResponses(items []models.ProxyInbound) []inboundResponse {
	out := make([]inboundResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toInboundResponse(item))
	}
	return out
}

func toInboundResponse(item models.ProxyInbound) inboundResponse {
	return inboundResponse{
		ID:                     item.ID,
		Name:                   item.Name,
		Template:               item.Template,
		Protocol:               item.Protocol,
		DomainID:               item.DomainID,
		Domain:                 item.Domain,
		Network:                item.Network,
		Security:               item.Security,
		Flow:                   item.Flow,
		RouteSNI:               item.RouteSNI,
		ListenAddr:             item.ListenAddr,
		ListenPort:             item.ListenPort,
		RealityHandshakeServer: item.RealityHandshakeServer,
		Enabled:                item.Enabled,
	}
}

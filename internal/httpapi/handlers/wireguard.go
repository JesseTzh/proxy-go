package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/httpapi/response"
	"github.com/proxy-go/proxy-go/internal/security"
	"github.com/proxy-go/proxy-go/internal/wireguard"
	"gorm.io/gorm"
)

func WireGuardState(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		state, err := d.WireGuard.State(c.Request.Context())
		if err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.JSON(c, 200, state)
	}
}

func UpdateWireGuard(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req wireguard.UpdateRequest
		if c.BindJSON(&req) != nil {
			response.Error(c, 400, "invalid json")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		server, err := d.WireGuard.Update(ctx, req)
		if err != nil {
			response.Error(c, 400, err.Error())
			return
		}
		d.Audit.Record("update_wireguard", "wireguard", "1", server, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.JSON(c, 200, server)
	}
}

func SetDomainAsWireGuardEndpoint(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		server, err := d.WireGuard.SetDomain(ctx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "not found")
			return
		}
		if err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		d.Audit.Record("set_wireguard_domain", "domain", fmt.Sprint(id), gin.H{"domain": server.Domain}, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.JSON(c, 200, server)
	}
}

func CreateWireGuardClient(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name string `json:"name"`
		}
		if c.BindJSON(&req) != nil {
			response.Error(c, 400, "invalid json")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		client, err := d.WireGuard.CreateClient(ctx, req.Name)
		if err != nil {
			response.Error(c, 400, err.Error())
			return
		}
		d.Audit.Record("create_wireguard_client", "wireguard_client", fmt.Sprint(client.ID), client, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.JSON(c, 200, client)
	}
}

func SetWireGuardClientEnabled(d Deps, enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		if err := d.WireGuard.SetClientEnabled(ctx, id, enabled); errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "not found")
			return
		} else if err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		d.Audit.Record("set_wireguard_client_enabled", "wireguard_client", fmt.Sprint(id), gin.H{"enabled": enabled}, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.OK(c)
	}
}

func DeleteWireGuardClient(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		if err := d.WireGuard.DeleteClient(ctx, id); errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "not found")
			return
		} else if err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		d.Audit.Record("delete_wireguard_client", "wireguard_client", fmt.Sprint(id), nil, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.OK(c)
	}
}

func WireGuardClientConfig(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		client, content, err := d.WireGuard.ClientConfig(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "not found")
			return
		}
		if err != nil {
			response.Error(c, 400, err.Error())
			return
		}
		filename := wireguard.ConfigFilename(client.Name)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(content))
	}
}

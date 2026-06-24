package handlers

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/nginx"
	"github.com/proxy-go/proxy-go/internal/security"
	runtimesvc "github.com/proxy-go/proxy-go/internal/services/runtime"
	"github.com/proxy-go/proxy-go/internal/xray"
)

func RuntimeStatus(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := runtimeService(d).Status()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, status)
	}
}

func ApplyRuntime(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		if err := runtimeService(d).Apply(ctx); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		d.Audit.Record("apply_runtime", "runtime", "", nil, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		c.JSON(200, gin.H{"ok": true})
	}
}

func ReloadNginx(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).ReloadNginx(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	}
}

func StartNginx(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).StartNginx(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	}
}

func StopNginx(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).StopNginx(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	}
}

func RestartNginx(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).RestartNginx(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	}
}

func StartXray(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).StartXray(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	}
}

func StopXray(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).StopXray(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	}
}

func RestartXray(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).RestartXray(c.Request.Context()); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": true})
	}
}

func RuntimeLogs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, runtimeService(d).Logs())
	}
}

func XrayLogs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, runtimeService(d).XrayLogs())
	}
}

func runtimeService(d Deps) *runtimesvc.Service {
	return &runtimesvc.Service{
		DB:    d.DB,
		Cfg:   d.Cfg,
		Nginx: nginxRuntimeAdapter{service: d.Nginx},
		Xray:  xrayRuntimeAdapter{service: d.Xray},
	}
}

type nginxRuntimeAdapter struct {
	service *nginx.Service
}

func (a nginxRuntimeAdapter) Apply(ctx context.Context) error   { return a.service.Apply(ctx) }
func (a nginxRuntimeAdapter) Reload(ctx context.Context) error  { return a.service.Reload(ctx) }
func (a nginxRuntimeAdapter) Start(ctx context.Context) error   { return a.service.Start(ctx) }
func (a nginxRuntimeAdapter) Stop(ctx context.Context) error    { return a.service.Stop(ctx) }
func (a nginxRuntimeAdapter) Restart(ctx context.Context) error { return a.service.Restart(ctx) }
func (a nginxRuntimeAdapter) Status() any                       { return a.service.Status() }

type xrayRuntimeAdapter struct {
	service *xray.Service
}

func (a xrayRuntimeAdapter) Apply(ctx context.Context) error { return a.service.Apply(ctx) }
func (a xrayRuntimeAdapter) Start(ctx context.Context) error { return a.service.Start(ctx) }
func (a xrayRuntimeAdapter) Stop(ctx context.Context) error  { return a.service.Stop(ctx) }
func (a xrayRuntimeAdapter) Restart(ctx context.Context) error {
	return a.service.Restart(ctx)
}
func (a xrayRuntimeAdapter) Status() any { return a.service.Status() }
func (a xrayRuntimeAdapter) Logs() []string {
	return a.service.Logs()
}

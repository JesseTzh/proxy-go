package handlers

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/httpapi/response"
	"github.com/proxy-go/proxy-go/internal/nginx"
	"github.com/proxy-go/proxy-go/internal/security"
	runtimesvc "github.com/proxy-go/proxy-go/internal/services/runtime"
	"github.com/proxy-go/proxy-go/internal/singbox"
)

func RuntimeStatus(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := runtimeService(d).Status()
		if err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.JSON(c, 200, status)
	}
}

func ApplyRuntime(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		if err := runtimeService(d).Apply(ctx); err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		d.Audit.Record("apply_runtime", "runtime", "", nil, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.OK(c)
	}
}

func ReloadNginx(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).ReloadNginx(c.Request.Context()); err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.OK(c)
	}
}

func StartNginx(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).StartNginx(c.Request.Context()); err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.OK(c)
	}
}

func StopNginx(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).StopNginx(c.Request.Context()); err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.OK(c)
	}
}

func RestartNginx(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).RestartNginx(c.Request.Context()); err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.OK(c)
	}
}

func StartSingBox(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).StartSingBox(c.Request.Context()); err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.OK(c)
	}
}

func StopSingBox(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).StopSingBox(c.Request.Context()); err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.OK(c)
	}
}

func RestartSingBox(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := runtimeService(d).RestartSingBox(c.Request.Context()); err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.OK(c)
	}
}

func SetSingBoxDebug(d Deps, enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		if err := runtimeService(d).SetSingBoxDebug(ctx, enabled); err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.OK(c)
	}
}

func RuntimeLogs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.JSON(c, 200, runtimeService(d).Logs())
	}
}

func SingBoxLogs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.JSON(c, 200, runtimeService(d).SingBoxLogs())
	}
}

func NginxConfig(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		config, err := runtimeService(d).NginxConfig()
		if err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.JSON(c, 200, config)
	}
}

func runtimeService(d Deps) *runtimesvc.Service {
	return &runtimesvc.Service{
		DB:      d.DB,
		Cfg:     d.Cfg,
		Nginx:   nginxRuntimeAdapter{service: d.Nginx},
		SingBox: singBoxRuntimeAdapter{service: d.SingBox},
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

type singBoxRuntimeAdapter struct {
	service *singbox.Service
}

func (a singBoxRuntimeAdapter) Apply(ctx context.Context) error { return a.service.Apply(ctx) }
func (a singBoxRuntimeAdapter) Start(ctx context.Context) error { return a.service.Start(ctx) }
func (a singBoxRuntimeAdapter) Stop(ctx context.Context) error  { return a.service.Stop(ctx) }
func (a singBoxRuntimeAdapter) Restart(ctx context.Context) error {
	return a.service.Restart(ctx)
}
func (a singBoxRuntimeAdapter) Status() any { return a.service.Status() }
func (a singBoxRuntimeAdapter) Logs() []string {
	return a.service.Logs()
}

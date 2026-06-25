package httpapi

import (
	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/httpapi/handlers"
	"github.com/proxy-go/proxy-go/internal/httpapi/middleware"
	"github.com/proxy-go/proxy-go/internal/httpapi/response"
)

type Deps = handlers.Deps

func Router(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.NoRoute(func(c *gin.Context) {
		response.Error(c, 404, "not found")
	})
	r.GET("/.well-known/acme-challenge/:token", handlers.ACMEChallenge(d))

	api := r.Group("/api")
	api.POST("/auth/login", handlers.Login(d))
	api.POST("/auth/logout", middleware.Auth(d.DB), handlers.Logout(d))
	api.GET("/auth/me", middleware.Auth(d.DB), handlers.Me())
	api.GET("/init/status", handlers.InitStatus(d))
	api.POST("/init/management-domain", middleware.Auth(d.DB), handlers.SetManagementDomain(d))
	api.POST("/init/disable-initial-port", middleware.Auth(d.DB), handlers.DisableInitialPort(d))

	protected := api.Group("", middleware.Auth(d.DB))
	protected.GET("/domains", handlers.ListDomains(d))
	protected.POST("/domains", handlers.CreateDomain(d))
	protected.GET("/domains/:id", handlers.GetDomain(d))
	protected.PUT("/domains/:id", handlers.UpdateDomain(d))
	protected.DELETE("/domains/:id", handlers.DeleteDomain(d))
	protected.POST("/domains/:id/dns-check", handlers.DNSCheck(d))
	protected.GET("/domains/:id/usage", handlers.DomainUsage(d))
	protected.POST("/domains/:id/certificate/issue", handlers.IssueDomainCertificate(d))
	protected.POST("/domains/:id/certificate/renew", handlers.RenewDomainCertificate(d))
	protected.DELETE("/domains/:id/certificate", handlers.DeleteDomainCertificate(d))
	protected.GET("/reverse-proxies", handlers.ListReverseProxies(d))
	protected.POST("/reverse-proxies", handlers.CreateReverseProxy(d))
	protected.PUT("/reverse-proxies/:id", handlers.UpdateReverseProxy(d))
	protected.DELETE("/reverse-proxies/:id", handlers.DeleteReverseProxy(d))
	protected.POST("/reverse-proxies/:id/enable", handlers.SetReverseProxyEnabled(d, true))
	protected.POST("/reverse-proxies/:id/disable", handlers.SetReverseProxyEnabled(d, false))
	protected.GET("/inbounds", handlers.ListInbounds(d))
	protected.POST("/inbounds", handlers.CreateInbound(d))
	protected.GET("/inbounds/:id", handlers.GetInbound(d))
	protected.PUT("/inbounds/:id", handlers.UpdateInbound(d))
	protected.DELETE("/inbounds/:id", handlers.DeleteInbound(d))
	protected.POST("/inbounds/:id/enable", handlers.SetInboundEnabled(d, true))
	protected.POST("/inbounds/:id/disable", handlers.SetInboundEnabled(d, false))
	protected.GET("/inbounds/:id/config", handlers.InboundConfig(d))
	protected.GET("/inbounds/:id/share", handlers.InboundShare(d))
	protected.GET("/runtime/status", handlers.RuntimeStatus(d))
	protected.POST("/runtime/apply", handlers.ApplyRuntime(d))
	protected.POST("/runtime/nginx/start", handlers.StartNginx(d))
	protected.POST("/runtime/nginx/stop", handlers.StopNginx(d))
	protected.POST("/runtime/nginx/restart", handlers.RestartNginx(d))
	protected.POST("/runtime/nginx/reload", handlers.ReloadNginx(d))
	protected.GET("/runtime/nginx/config", handlers.NginxConfig(d))
	protected.POST("/runtime/xray/start", handlers.StartXray(d))
	protected.POST("/runtime/xray/stop", handlers.StopXray(d))
	protected.POST("/runtime/xray/restart", handlers.RestartXray(d))
	protected.GET("/runtime/xray/logs", handlers.XrayLogs(d))
	protected.GET("/runtime/logs", handlers.RuntimeLogs(d))
	protected.GET("/capabilities", handlers.Capabilities(d))
	protected.GET("/audit-logs", handlers.AuditLogs(d))
	protected.GET("/settings", handlers.Settings(d))
	protected.PUT("/settings", handlers.UpdateSettings(d))
	return r
}

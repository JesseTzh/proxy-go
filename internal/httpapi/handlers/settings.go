package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/httpapi/response"
	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/security"
)

func InitStatus(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) { var s models.SystemSetting; d.DB.First(&s, 1); response.JSON(c, 200, s) }
}

func SetManagementDomain(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Domain string `json:"domain" validate:"required,fqdn"`
		}
		if c.BindJSON(&req) != nil || d.Validator.Struct(req) != nil {
			response.Error(c, 400, "invalid domain")
			return
		}
		d.DB.Model(&models.SystemSetting{}).Where("id=1").Update("management_domain", req.Domain)
		d.Audit.Record("set_management_domain", "system", "1", req, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.OK(c)
	}
}

func DisableInitialPort(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		d.DB.Model(&models.SystemSetting{}).Where("id=1").Update("initial_port_enabled", false)
		d.Audit.Record("disable_initial_port", "system", "1", nil, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.JSON(c, 200, gin.H{"message": "restart proxy-go to stop initial listener"})
	}
}

func Settings(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var s models.SystemSetting
		d.DB.First(&s, 1)
		response.JSON(c, 200, gin.H{"settings": s, "paths": d.Cfg.Paths, "versions": gin.H{"proxyGo": "dev", "nginx": d.Nginx.Binary, "singBox": d.SingBox.Binary}})
	}
}

func UpdateSettings(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ACMEEmail        string `json:"acmeEmail"`
			ManagementDomain string `json:"managementDomain"`
		}
		_ = c.BindJSON(&req)
		d.DB.Model(&models.SystemSetting{}).Where("id=1").Updates(map[string]any{"acme_email": req.ACMEEmail, "management_domain": req.ManagementDomain})
		response.OK(c)
	}
}

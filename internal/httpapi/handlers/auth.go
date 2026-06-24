package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/security"
)

func ACMEChallenge(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if key, ok := d.ACME.GetKeyAuthorization(c.Param("token")); ok {
			c.String(200, key)
			return
		}
		c.String(404, "not found")
	}
}

func Me() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(200, gin.H{"authenticated": true}) }
}

func Login(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Password string `json:"password" validate:"required"`
		}
		if err := c.BindJSON(&req); err != nil || d.Validator.Struct(req) != nil {
			c.JSON(400, gin.H{"error": "password required"})
			return
		}
		ip := security.NormalizeIP(c.Request.RemoteAddr)
		if !d.Limiter.Allow(ip) {
			c.JSON(429, gin.H{"error": "too many login failures"})
			return
		}
		var authCfg models.AuthConfig
		if err := d.DB.First(&authCfg, 1).Error; err != nil {
			c.JSON(500, gin.H{"error": "auth not initialized"})
			return
		}
		if !security.CheckPassword(authCfg.PasswordHash, req.Password) {
			d.Limiter.Fail(ip)
			d.Audit.Record("login_failed", "auth", "", gin.H{"reason": "invalid_password"}, ip, c.Request.UserAgent())
			c.JSON(401, gin.H{"error": "invalid password"})
			return
		}
		token, err := security.NewToken()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ttl := time.Duration(d.Cfg.Security.SessionTTLHours) * time.Hour
		d.DB.Create(&models.Session{TokenHash: security.HashToken(token), IP: ip, UserAgent: c.Request.UserAgent(), ExpiresAt: time.Now().Add(ttl)})
		d.Limiter.Success(ip)
		http.SetCookie(c.Writer, &http.Cookie{Name: "proxy_go_session", Value: token, Path: "/", HttpOnly: true, Secure: d.Cfg.Server.CookieSecure, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(ttl)})
		d.Audit.Record("login_success", "auth", "", nil, ip, c.Request.UserAgent())
		c.JSON(200, gin.H{"ok": true})
	}
}

func Logout(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token, err := c.Cookie("proxy_go_session"); err == nil {
			d.DB.Where("token_hash = ?", security.HashToken(token)).Delete(&models.Session{})
		}
		http.SetCookie(c.Writer, &http.Cookie{Name: "proxy_go_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
		d.Audit.Record("logout", "auth", "", nil, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		c.JSON(200, gin.H{"ok": true})
	}
}

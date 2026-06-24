package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/models"
	"github.com/proxy-go/proxy-go/internal/security"
	"gorm.io/gorm"
)

func Auth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("proxy_go_session")
		if err != nil || token == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}
		var s models.Session
		if err := db.Where("token_hash = ? AND expires_at > ?", security.HashToken(token), time.Now()).First(&s).Error; err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

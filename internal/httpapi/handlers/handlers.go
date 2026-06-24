package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/proxy-go/proxy-go/internal/acme"
	"github.com/proxy-go/proxy-go/internal/audit"
	"github.com/proxy-go/proxy-go/internal/config"
	"github.com/proxy-go/proxy-go/internal/nginx"
	"github.com/proxy-go/proxy-go/internal/security"
	"github.com/proxy-go/proxy-go/internal/xray"
	"gorm.io/gorm"
)

type Deps struct {
	Cfg       *config.Config
	DB        *gorm.DB
	Audit     *audit.Logger
	ACME      *acme.Service
	Nginx     *nginx.Service
	Xray      *xray.Service
	Limiter   *security.LoginLimiter
	Validator *validator.Validate
}

func idParam(c *gin.Context) (uint, error) {
	i, err := strconv.Atoi(c.Param("id"))
	return uint(i), err
}

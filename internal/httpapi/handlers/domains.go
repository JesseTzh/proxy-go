package handlers

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/proxy-go/proxy-go/internal/httpapi/response"
	"github.com/proxy-go/proxy-go/internal/security"
	domainssvc "github.com/proxy-go/proxy-go/internal/services/domains"
	"gorm.io/gorm"
)

func ListDomains(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := domainssvc.New(d.DB).List()
		if err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.JSON(c, 200, items)
	}
}

func CreateDomain(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Domain, Remark string
			Status         string
		}
		if c.BindJSON(&req) != nil || req.Domain == "" {
			response.Error(c, 400, "domain required")
			return
		}
		item, err := domainssvc.New(d.DB).Create(req.Domain, req.Remark, req.Status)
		if err != nil {
			response.Error(c, 400, err.Error())
			return
		}
		d.Audit.Record("create_domain", "domain", fmt.Sprint(item.ID), item, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		item, _ = domainssvc.New(d.DB).Get(item.ID)
		response.JSON(c, 200, item)
	}
}

func GetDomain(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		item, err := domainssvc.New(d.DB).Get(id)
		if err != nil {
			response.Error(c, 404, "not found")
			return
		}
		response.JSON(c, 200, item)
	}
}

func UpdateDomain(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		var req struct {
			Remark, Status string
			CertificateID  *uint
		}
		_ = c.BindJSON(&req)
		item, err := domainssvc.New(d.DB).Update(id, req.Remark, req.Status, req.CertificateID)
		if err != nil {
			response.Error(c, 404, "not found")
			return
		}
		d.Audit.Record("update_domain", "domain", fmt.Sprint(item.ID), item, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		item, _ = domainssvc.New(d.DB).Get(item.ID)
		response.JSON(c, 200, item)
	}
}

func DeleteDomain(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		err = domainssvc.New(d.DB).Delete(id)
		if err != nil {
			if err.Error() == "domain is in use" {
				response.Error(c, 409, err.Error())
				return
			}
			response.Error(c, 400, err.Error())
			return
		}
		d.Audit.Record("delete_domain", "domain", fmt.Sprint(id), nil, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.OK(c)
	}
}

func DNSCheck(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		result, err := domainssvc.New(d.DB).DNSCheck(id)
		if err != nil {
			response.Error(c, 404, "not found")
			return
		}
		response.JSON(c, 200, result)
	}
}

func DomainUsage(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		usage, err := domainssvc.New(d.DB).Usage(id)
		if err != nil {
			response.Error(c, 500, err.Error())
			return
		}
		response.JSON(c, 200, usage)
	}
}

func IssueDomainCertificate(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		svc := domainssvc.NewWithCertificateIssuer(d.DB, d.ACME, d.Cfg)
		if err := svc.IssueCertificate(id); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.Error(c, 404, "not found")
				return
			}
			d.Audit.Record("issue_domain_certificate_failed", "domain", fmt.Sprint(id), gin.H{"error": err.Error()}, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
			response.Error(c, 501, err.Error())
			return
		}
		item, err := domainssvc.New(d.DB).Get(id)
		if err != nil {
			response.Error(c, 404, "not found")
			return
		}
		d.Audit.Record("issue_domain_certificate", "domain", fmt.Sprint(id), item, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.JSON(c, 200, item)
	}
}

func RenewDomainCertificate(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		svc := domainssvc.NewWithCertificateIssuer(d.DB, d.ACME, d.Cfg)
		if err := svc.RenewCertificate(id); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.Error(c, 404, "not found")
				return
			}
			response.Error(c, 500, err.Error())
			return
		}
		item, err := domainssvc.New(d.DB).Get(id)
		if err != nil {
			response.Error(c, 404, "not found")
			return
		}
		d.Audit.Record("renew_domain_certificate", "domain", fmt.Sprint(id), item, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.JSON(c, 200, item)
	}
}

func DeleteDomainCertificate(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := idParam(c)
		if err != nil {
			response.Error(c, 400, "invalid id")
			return
		}
		svc := domainssvc.NewWithCertificateIssuer(d.DB, d.ACME, d.Cfg)
		if err := svc.DeleteCertificate(id); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.Error(c, 404, "not found")
				return
			}
			response.Error(c, 400, err.Error())
			return
		}
		item, err := domainssvc.New(d.DB).Get(id)
		if err != nil {
			response.Error(c, 404, "not found")
			return
		}
		d.Audit.Record("delete_domain_certificate", "domain", fmt.Sprint(id), item, security.NormalizeIP(c.Request.RemoteAddr), c.Request.UserAgent())
		response.JSON(c, 200, item)
	}
}

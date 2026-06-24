package audit

import (
	"encoding/json"

	"github.com/proxy-go/proxy-go/internal/models"
	"gorm.io/gorm"
)

type Logger struct{ db *gorm.DB }

func New(db *gorm.DB) *Logger { return &Logger{db: db} }

func (l *Logger) Record(action, resourceType, resourceID string, detail any, ip, ua string) {
	var d string
	if detail != nil {
		b, _ := json.Marshal(detail)
		d = string(b)
	}
	_ = l.db.Create(&models.AuditLog{Action: action, ResourceType: resourceType, ResourceID: resourceID, Detail: d, IP: ip, UserAgent: ua}).Error
}

package ports

import (
	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

// EmailService es el puerto para el servicio de email
type EmailService interface {
	SendEmail(to []string, subject, body string, isHTML bool) error
	SendMonitoringNotification(to []string, status domain.MonitoringStatus) error
}

// EmailTemplateEngine es el puerto para generación de templates
type EmailTemplateEngine interface {
	GenerateMonitoringTemplate(status domain.MonitoringStatus) (html, plainText string)
	GenerateAlertTemplate(alertType, message string) (html, plainText string)
}

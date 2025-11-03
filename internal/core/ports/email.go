package ports

import (
	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

// IEmailService es el puerto para el servicio de email
type IEmailService interface {
	SendEmail(subject, body string, isHTML bool) error
	SendMonitoringNotification(status domain.MonitoringStatus) error
	//config email sender
	GetSenderConfig() (*domain.EmailConfig, error)
	SaveSenderConfig(config *domain.EmailConfig) error
	GetNotificationEmails() ([]*domain.NotificationEmail, error)
	AddNotificationEmail(email string) error
	RemoveNotificationEmail(email string) error
}

// EmailTemplateEngine es el puerto para generación de templates
type EmailTemplateEngine interface {
	GenerateMonitoringTemplate(status domain.MonitoringStatus) (html, plainText string)
	GenerateAlertTemplate(alertType, message string) (html, plainText string)
}

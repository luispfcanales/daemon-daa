package services

import (
	"fmt"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
	"github.com/luispfcanales/daemon-daa/internal/core/ports"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/adapters/email"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/adapters/email/templates"
)

// EmailService implementa el puerto EmailService
type EmailService struct {
	smtpAdapter *email.SMTPAdapter
	config      *domain.EmailConfig
}

// NewEmailService crea una nueva instancia del servicio de email
func NewEmailService(cfg *domain.EmailConfig) ports.EmailService {
	smtpAdapter := email.NewSMTPAdapter(cfg)
	return &EmailService{
		smtpAdapter: smtpAdapter,
		config:      cfg,
	}
}

// SendEmail envía un correo electrónico
func (s *EmailService) SendEmail(to []string, subject, body string, isHTML bool) error {
	// Enviar email
	if err := s.smtpAdapter.Send(to, subject, body, isHTML); err != nil {
		return fmt.Errorf("error enviando email: %w", err)
	}

	return nil
}

// SendMonitoringNotification envía una notificación de estado de monitoreo
func (s *EmailService) SendMonitoringNotification(to []string, status domain.MonitoringStatus) error {
	// Generar templates
	htmlBody := templates.MonitoringTemplate(status)

	// Determinar el subject basado en el estado
	subject := "🔴 Daemon DAA - Monitoreo Detenido"

	if status.IsRunning {
		subject = "🟢 Daemon DAA - Monitoreo Iniciado"
	}

	// Intentar enviar como HTML, fallback a texto plano
	err := s.SendEmail(to, subject, htmlBody, true)
	if err != nil {
		return fmt.Errorf("error enviando notificación de monitoreo: %w", err)
	}

	return nil
}

package services

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
	"github.com/luispfcanales/daemon-daa/internal/core/ports"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/adapters/email"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/adapters/email/templates"
)

var (
	ErrNotConfigured = errors.New("email service not configured")
)

const (
	senderConfigFile       = "email_sender.csv"
	notificationEmailsFile = "notification_emails.csv"
)

// EmailService implementa el puerto EmailService
type EmailService struct {
	smtpAdapter *email.SMTPAdapter
	config      *domain.EmailConfig
	dataDir     string
}

func (s *EmailService) initializeFiles() error {
	// Inicializar archivo de configuración del remitente si no existe
	senderFilePath := filepath.Join(s.dataDir, senderConfigFile)
	if _, err := os.Stat(senderFilePath); os.IsNotExist(err) {
		file, err := os.Create(senderFilePath)
		if err != nil {
			return fmt.Errorf("no se pudo crear archivo de configuración del remitente: %w", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Escribir encabezados
		headers := []string{"email", "gmail_app_password", "created_at", "updated_at"}
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("error escribiendo headers del remitente: %w", err)
		}
	}

	// Inicializar archivo de correos de notificación si no existe
	emailsFilePath := filepath.Join(s.dataDir, notificationEmailsFile)
	if _, err := os.Stat(emailsFilePath); os.IsNotExist(err) {
		file, err := os.Create(emailsFilePath)
		if err != nil {
			return fmt.Errorf("no se pudo crear archivo de correos de notificación: %w", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Escribir encabezados
		headers := []string{"email", "created_at"}
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("error escribiendo headers de notificaciones: %w", err)
		}
	}

	return nil
}

func (s *EmailService) GetSenderConfig() (*domain.EmailConfig, error) {
	filePath := filepath.Join(s.dataDir, senderConfigFile)

	// Verificar si el archivo existe
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // No hay configuración guardada
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error abriendo archivo de configuración: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error leyendo CSV: %w", err)
	}

	// El archivo debe tener al menos una fila de datos (sin contar headers)
	if len(records) < 2 {
		return nil, nil
	}

	// Saltar headers y tomar la primera fila de datos
	record := records[1]

	if len(record) < 4 {
		return nil, fmt.Errorf("formato de archivo inválido")
	}

	config := &domain.EmailConfig{
		Email:            record[0],
		GmailAppPassword: record[1],
		CreatedAt:        record[2],
		UpdatedAt:        record[3],
	}

	return config, nil
}

func (s *EmailService) SaveSenderConfig(config *domain.EmailConfig) error {
	filePath := filepath.Join(s.dataDir, senderConfigFile)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creando archivo: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Escribir headers
	headers := []string{"email", "gmail_app_password", "created_at", "updated_at"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("error escribiendo headers: %w", err)
	}

	// Usar timestamp actual si no existe
	now := time.Now().Format(time.RFC3339)
	if config.CreatedAt == "" {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	// Escribir datos
	record := []string{
		config.Email,
		config.GmailAppPassword,
		config.CreatedAt,
		config.UpdatedAt,
	}

	if err := writer.Write(record); err != nil {
		return fmt.Errorf("error escribiendo datos: %w", err)
	}

	return nil
}
func (s *EmailService) GetNotificationEmails() ([]*domain.NotificationEmail, error) {
	panic("not implemented") // TODO: Implement
}
func (s *EmailService) AddNotificationEmail(email string) error {
	panic("not implemented") // TODO: Implement
}
func (s *EmailService) RemoveNotificationEmail(email string) error {
	panic("not implemented") // TODO: Implement
}

// NewEmailService crea una nueva instancia del servicio de email
func NewEmailService(cfg *domain.EmailConfig, pathDir string) ports.IEmailService {
	smtpAdapter := email.NewSMTPAdapter(cfg)
	srv := &EmailService{
		smtpAdapter: smtpAdapter,
		config:      cfg,
	}

	srv.dataDir = pathDir
	srv.initializeFiles()
	return srv
}

// SendEmail envía un correo electrónico
func (s *EmailService) SendEmail(subject, body string, isHTML bool) error {
	if !s.isValidToSendEmail() {
		return ErrNotConfigured
	}
	// Enviar email
	if err := s.smtpAdapter.Send([]string{}, subject, body, isHTML); err != nil {
		return fmt.Errorf("error enviando email: %w", err)
	}

	return nil
}

func (s *EmailService) isValidToSendEmail() bool {

	isValid := s.config.Email != "" && s.config.GmailAppPassword != ""

	return isValid
}

// SendMonitoringNotification envía una notificación de estado de monitoreo
func (s *EmailService) SendMonitoringNotification(status domain.MonitoringStatus) error {
	// Generar templates
	htmlBody := templates.MonitoringTemplate(status)

	// Determinar el subject basado en el estado
	subject := "🔴 Daemon DAA - Monitoreo Detenido"

	if status.IsRunning {
		subject = "🟢 Daemon DAA - Monitoreo Iniciado"
	}

	// Intentar enviar como HTML, fallback a texto plano
	err := s.SendEmail(subject, htmlBody, true)
	if err != nil {
		return fmt.Errorf("error enviando notificación de monitoreo: %w", err)
	}

	return nil
}

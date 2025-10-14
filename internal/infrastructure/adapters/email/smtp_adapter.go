package email

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

// SMTPAdapter implementa el adaptador SMTP para envío de emails
type SMTPAdapter struct {
	config *domain.EmailConfig
}

// NewSMTPAdapter crea una nueva instancia del adaptador SMTP
func NewSMTPAdapter(cfg *domain.EmailConfig) *SMTPAdapter {
	return &SMTPAdapter{
		config: cfg,
	}
}

// Send envía un email a través de SMTP
func (a *SMTPAdapter) Send(to []string, subject, body string, isHTML bool) error {
	var contentType string
	if isHTML {
		contentType = "text/html; charset=\"UTF-8\""
	} else {
		contentType = "text/plain; charset=\"UTF-8\""
	}

	// Generar un ID único para cada email
	uniqueID := uuid.New().String()
	now := time.Now()

	// Headers del email
	headers := make(map[string]string)
	headers["From"] = a.config.From
	headers["To"] = strings.Join(to, ", ")
	headers["Subject"] = subject
	headers["Date"] = now.Format(time.RFC1123Z)

	// Message-ID único
	headers["Message-ID"] = fmt.Sprintf("<%s.%d@%s>",
		uniqueID,
		now.UnixNano(),
		a.config.Host)

	headers["MIME-version"] = "1.0"
	headers["Content-Type"] = contentType

	// CLAVE: Headers para forzar NO agrupación
	headers["X-Entity-Ref-ID"] = uniqueID
	headers["X-Message-ID"] = fmt.Sprintf("%d", now.UnixNano())

	// Construir mensaje
	message := ""
	orderedHeaders := []string{
		"From", "To", "Subject", "Date", "Message-ID",
		"MIME-version", "Content-Type", "X-Entity-Ref-ID", "X-Message-ID",
	}

	for _, key := range orderedHeaders {
		if val, exists := headers[key]; exists {
			message += fmt.Sprintf("%s: %s\r\n", key, val)
		}
	}

	message += "\r\n" + body

	// Autenticación
	auth := smtp.PlainAuth("", a.config.Username, a.config.Password, a.config.Host)

	return a.sendMail(to, []byte(message), auth)
}

// sendMail maneja el envío real
func (a *SMTPAdapter) sendMail(to []string, message []byte, auth smtp.Auth) error {
	addr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)

	// Enviar usando STARTTLS (puerto 587)
	return smtp.SendMail(addr, auth, a.config.From, to, message)
}

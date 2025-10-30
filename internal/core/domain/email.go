package domain

import "time"

// Email representa un correo electrónico
type Email struct {
	ID        string
	To        []string
	Subject   string
	Body      string
	IsHTML    bool
	Status    EmailStatus
	CreatedAt time.Time
	SentAt    *time.Time
}

// EmailStatus representa el estado de un email
type EmailStatus string

const (
	EmailStatusPending EmailStatus = "pending"
	EmailStatusSent    EmailStatus = "sent"
	EmailStatusFailed  EmailStatus = "failed"
)

// MonitoringStatus representa el estado del monitoreo para notificaciones
type MonitoringStatus struct {
	IsRunning bool          `json:"is_running"`
	Interval  time.Duration `json:"interval"`
	StartedAt time.Time     `json:"started_at,omitempty"`
	Message   string        `json:"message,omitempty"`
}

// EmailConfig configuración para el servicio de email
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// NotificationEmail representa un correo para notificaciones
type NotificationEmail struct {
	Email     string `json:"email"`
	CreatedAt string `json:"created_at,omitempty"`
}

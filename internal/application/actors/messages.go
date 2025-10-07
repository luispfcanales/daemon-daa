package actors

import (
	"time"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

// Mensajes del sistema
type CheckDomain struct {
	Name string
}

type CheckAllDomains struct{}

type DomainChecked struct {
	Check domain.DomainCheck
}

type Alert struct {
	Message string
	Level   string // "WARNING", "ERROR"
}

type StartMonitoring struct {
	Interval int // segundos
}

type StopMonitoring struct{}

type GetStatus struct{}

// Nuevos mensajes para control del monitoreo
type GetMonitoringStatus struct{}

type MonitoringStatus struct {
	IsRunning bool          `json:"is_running"`
	Interval  time.Duration `json:"interval"`
	StartedAt time.Time     `json:"started_at,omitempty"`
	Message   string        `json:"message,omitempty"`
}

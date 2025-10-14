package actors

import (
	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

// Mensajes del sistema
type CheckDomain struct {
	Name string
}

type CheckAllDomains struct{}

type DomainChecked struct {
	Check domain.DomainCheck `json:"check,omitempty"`
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

type GetStatsDomain struct {
	Stats domain.StatsDomain `json:"stats,omitempty"`
}

// Nuevos mensajes para control del monitoreo
type GetMonitoringStatus struct{}

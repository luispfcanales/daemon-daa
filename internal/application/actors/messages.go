package actors

import (
	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

// Messages actor single checker
type CheckDomain struct{}

type GenerateStatsDomain struct{}

type NotifyStats struct {
	Stats domain.StatsDomain `json:"stats,omitempty"`
}

// messages actor monitoring
type CheckAllDomains struct{}

type DomainChecked struct {
	Check domain.DomainCheck `json:"check,omitempty"`
}

type Alert struct {
	Message string
	Level   string // "WARNING", "ERROR"
}

type StartMonitoring struct {
	Interval int
}

type StopMonitoring struct{}

// Nuevos mensajes para control del monitoreo
type GetMonitoringStatus struct{}

package actors

import "github.com/luispfcanales/daemon-daa/internal/core/domain"

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

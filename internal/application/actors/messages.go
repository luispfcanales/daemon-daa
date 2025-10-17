package actors

import (
	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

// CheckDomain Messages actor single checker
type CheckDomain struct{}

type NotifyStats struct {
	Stats domain.StatsDomain `json:"stats,omitempty"`
}

// CheckAllDomains  messages actor monitoring
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

// GetMonitoringStatus Nuevos mensajes para control del monitoreo
type GetMonitoringStatus struct{}

// GetCachedStats solicita las estadísticas en caché de un dominio
type GetCachedStats struct{}

// CachedStatsResponse respuesta con las estadísticas en caché
type CachedStatsResponse struct {
	Stats *domain.StatsDomain
	Found bool
}

// GetAllCachedStats solicita estadísticas de todos los dominios
type GetAllCachedStats struct{}

// AllCachedStatsResponse respuesta con todas las estadísticas
type AllCachedStatsResponse struct {
	Stats []*domain.StatsDomain `json:"stats"`
}

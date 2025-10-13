package ports

import (
	"time"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

type DomainRepository interface {
	GetDomainConfigs() ([]domain.DomainConfig, error)
	SaveDomainCheck(check domain.DomainCheck) error
	GetChecks() ([]domain.DomainCheck, error)
	GetChecksByDomain(domain string) ([]domain.DomainCheck, error)
	GetRecentChecks(limit int) ([]domain.DomainCheck, error)
	GetChecksByTimeRange(start, end time.Time) ([]domain.DomainCheck, error)
	AddDomainConfig(config domain.DomainConfig) error
	RemoveDomainConfig(domain string) error
	GetDomainStats(domain string) (map[string]any, error)
}

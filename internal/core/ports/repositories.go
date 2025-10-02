package ports

import "github.com/luispfcanales/daemon-daa/internal/core/domain"

type DomainRepository interface {
	GetDomainConfigs() ([]domain.DomainConfig, error)
	SaveDomainCheck(check domain.DomainCheck) error
	GetChecks() []domain.DomainCheck
}

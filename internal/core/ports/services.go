package ports

import "github.com/luispfcanales/daemon-daa/internal/core/domain"

type IPService interface {
	GetStats(domain string) (map[string]any, error)
	ListDomains() ([]domain.DomainConfig, error)
	AddDomain(c domain.DomainConfig) error
}

package repositories

import (
	"sync"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

type InMemoryDomainRepository struct {
	configs []domain.DomainConfig
	checks  []domain.DomainCheck
	mutex   sync.RWMutex
}

func NewInMemoryDomainRepository() *InMemoryDomainRepository {
	return &InMemoryDomainRepository{
		configs: []domain.DomainConfig{
			{Domain: "intranet.unamad.edu.pe", ExpectedIP: "110.238.69.0"},
			{Domain: "aulavirtual.unamad.edu.pe", ExpectedIP: "110.238.69.0"},
			{Domain: "matricula.unamad.edu.pe", ExpectedIP: "110.238.69.0"},
		},
		checks: []domain.DomainCheck{},
	}
}

func (r *InMemoryDomainRepository) GetDomainConfigs() ([]domain.DomainConfig, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.configs, nil
}

func (r *InMemoryDomainRepository) SaveDomainCheck(check domain.DomainCheck) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.checks = append(r.checks, check)
	return nil
}

func (r *InMemoryDomainRepository) GetChecks() []domain.DomainCheck {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.checks
}

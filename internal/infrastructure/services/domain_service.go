package services

import (
	"github.com/luispfcanales/daemon-daa/internal/core/domain"
	"github.com/luispfcanales/daemon-daa/internal/core/ports"
)

type IPDomainService struct {
	repo ports.DomainRepository
}

func NewIPDomainService(repo ports.DomainRepository) ports.IPService {
	return &IPDomainService{
		repo: repo,
	}
}

func (s *IPDomainService) ListDomains() ([]domain.DomainConfig, error) {
	return s.repo.GetDomainConfigs()
}

func (s *IPDomainService) AddDomain(c domain.DomainConfig) error {
	return s.repo.AddDomainConfig(c)
}

func (s *IPDomainService) DeleteDomainIP(domainName string) error {
	return s.repo.RemoveDomainConfig(domainName)
}

func (s *IPDomainService) UpdateDomainIP(up domain.DomainConfig) error {
	return s.repo.UpdateDomainConfig(up.ID, up)
}

func (s *IPDomainService) GetStats(domain string) (map[string]any, error) {
	// Obtener estadísticas del dominio usando el repositorio
	stats, err := s.repo.GetDomainStats(domain)
	if err != nil {
		return nil, err
	}

	// Podemos enriquecer las estadísticas con información adicional si es necesario
	// Por ejemplo, calcular tendencias o agregar metadata

	// Estructura estándar de respuesta
	result := map[string]any{
		"domain": domain,
		"stats":  stats,
		"status": "completed",
	}

	// Determinar estado general basado en el success rate
	if stats.SuccessRate >= 95.0 {
		result["health_status"] = "excellent"
	} else if stats.SuccessRate >= 80.0 {
		result["health_status"] = "good"
	} else if stats.SuccessRate >= 60.0 {
		result["health_status"] = "degraded"
	} else {
		result["health_status"] = "poor"
	}

	return result, nil
}

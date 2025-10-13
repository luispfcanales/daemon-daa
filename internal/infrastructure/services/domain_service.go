package services

import (
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
	if successRate, ok := stats["success_rate"].(float64); ok {
		if successRate >= 95.0 {
			result["health_status"] = "excellent"
		} else if successRate >= 80.0 {
			result["health_status"] = "good"
		} else if successRate >= 60.0 {
			result["health_status"] = "degraded"
		} else {
			result["health_status"] = "poor"
		}
	}

	return result, nil
}

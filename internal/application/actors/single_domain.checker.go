package actors

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/luispfcanales/daemon-daa/internal/application/events"
	"github.com/luispfcanales/daemon-daa/internal/core/domain"
	"github.com/luispfcanales/daemon-daa/internal/core/ports"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/adapters"
)

type SingleDomainChecker struct {
	config      domain.DomainConfig
	dnsResolver *adapters.DNSResolver

	// Caché de estadísticas
	statDomain *domain.StatsDomain
	mu         sync.RWMutex

	repository ports.DomainRepository
	eventBus   *events.EventBus
}

func NewSingleDomainChecker(
	config domain.DomainConfig,
	repository ports.DomainRepository,
	eventBus *events.EventBus,
) actor.Producer {
	return func() actor.Receiver {
		return &SingleDomainChecker{
			config:      config,
			dnsResolver: adapters.NewDNSResolver(),
			repository:  repository,
			eventBus:    eventBus,
		}
	}
}

func (s *SingleDomainChecker) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Started:
		s.generateStast()
		slog.Info("SingleDomainChecker started",
			"domain", s.config.Domain,
			"pid", c.PID())

	case CheckDomain:
		s.handleCheckDomain(c)

	case NotifyStats:
		s.handleNotifyStats(c)

	case GetCachedStats:
		s.handleGetCachedStats(c)

	case actor.Stopped:
		slog.Debug("SingleDomainChecker stopped",
			"domain", s.config.Domain,
			"pid", c.PID())
	}
}

func (s *SingleDomainChecker) handleGetCachedStats(c *actor.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := CachedStatsResponse{
		Stats: nil,
		Found: false,
	}

	if s.statDomain != nil {
		statsCopy := *s.statDomain
		response.Stats = &statsCopy
		response.Found = true

		slog.Debug("Cached stats retrieved",
			"domain", s.config.Domain,
			"total_checks", statsCopy.TotalChecks)
	} else {
		slog.Debug("No cached stats available yet", "domain", s.config.Domain)
	}

	if c.Sender() != nil {
		c.Send(c.Sender(), response)
	}
}

func (s *SingleDomainChecker) handleCheckDomain(c *actor.Context) {
	slog.Info("Checking domain concurrently", "domain", s.config.Domain)
	startTime := time.Now()

	check := domain.DomainCheck{
		Domain:     s.config.Domain,
		ExpectedIP: s.config.ExpectedIP,
		Timestamp:  startTime,
	}

	var err error
	var ips []string
	ips, err = s.dnsResolver.ResolveIP(s.config.Domain)
	if err != nil {
		check.Error = err.Error()
		check.IsValid = false
	} else {
		check.ActualIPs = ips
		check.IsValid = s.validateIPs(ips, s.config.ExpectedIP)
	}

	totalDuration := time.Since(startTime)
	check.DurationMs = float64(totalDuration.Nanoseconds()) / 1e6

	s.repository.SaveDomainCheck(check)
	s.generateStast()

	c.Send(c.Parent(), DomainChecked{Check: check})
	c.Send(c.PID(), NotifyStats{})

	// Alertas
	if !check.IsValid {
		alertMsg := fmt.Sprintf("ALERTA: Dominio %s tiene IPs inesperadas. Esperado: %s, Obtenido: %v (Tiempo: %.2fms)",
			s.config.Domain, s.config.ExpectedIP, ips, check.DurationMs)
		c.Send(c.Parent(), Alert{
			Message: alertMsg,
			Level:   "WARNING",
		})
	}
}

func (s *SingleDomainChecker) handleNotifyStats(_ *actor.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.statDomain == nil {
		slog.Warn("No stats available in cache yet", "domain", s.config.Domain)
		return
	}

	stats := *s.statDomain

	s.eventBus.Broadcast(events.Event{
		Type: "monitoring_domain_stats",
		Data: NotifyStats{
			Stats: stats,
		},
		Timestamp: time.Now(),
	})
}

func (s *SingleDomainChecker) generateStast() {
	data, err := s.repository.GetDomainStats(s.config.Domain)
	if err != nil {
		slog.Error("Error getting domain stats",
			"domain", s.config.Domain,
			"error", err)
		return
	}

	s.mu.Lock()
	s.statDomain = &domain.StatsDomain{
		TotalChecks:      data.TotalChecks,
		SuccessCount:     data.SuccessCount,
		FailureCount:     data.FailureCount,
		SuccessRate:      data.SuccessRate,
		AverageUptime:    data.AverageUptime,
		LastCheck:        data.LastCheck,
		AvgResponseTime:  data.AvgResponseTime,
		MinResponseTime:  data.MinResponseTime,
		MaxResponseTime:  data.MaxResponseTime,
		P95ResponseTime:  data.P95ResponseTime,
		ChecksWithTiming: data.ChecksWithTiming,
		DNS:              s.config.Domain,
	}
	s.mu.Unlock()
}

func (s *SingleDomainChecker) validateIPs(actualIPs []string, expectedIP string) bool {
	return slices.Contains(actualIPs, expectedIP)
}

package actors

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/adapters"

	"github.com/anthdm/hollywood/actor"
)

type SingleDomainChecker struct {
	config      domain.DomainConfig
	dnsResolver *adapters.DNSResolver
}

func NewSingleDomainChecker(config domain.DomainConfig) actor.Producer {
	return func() actor.Receiver {
		return &SingleDomainChecker{
			config:      config,
			dnsResolver: adapters.NewDNSResolver(),
		}
	}
}

func (s *SingleDomainChecker) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Debug("SingleDomainChecker started",
			"domain", s.config.Domain,
			"pid", c.PID())

	case CheckDomain:
		s.handleCheckDomain(c, msg)

	case actor.Stopped:
		slog.Debug("SingleDomainChecker stopped",
			"domain", s.config.Domain,
			"pid", c.PID())
	}
}

func (s *SingleDomainChecker) handleCheckDomain(c *actor.Context, msg CheckDomain) {
	slog.Info("Checking domain concurrently", "domain", msg.Name)

	startTime := time.Now()

	check := domain.DomainCheck{
		Domain:     s.config.Domain,
		ExpectedIP: s.config.ExpectedIP,
		Timestamp:  startTime, // Usar el mismo timestamp de inicio
	}

	var err error
	var ips []string

	dnsStart := time.Now()
	ips, err = s.dnsResolver.ResolveIP(msg.Name)
	dnsDuration := time.Since(dnsStart).Milliseconds()

	if err != nil {
		check.Error = err.Error()
		check.IsValid = false
	} else {
		check.ActualIPs = ips
		check.IsValid = s.validateIPs(ips, s.config.ExpectedIP)
	}

	totalDuration := time.Since(startTime)
	check.DurationMs = float64(totalDuration.Nanoseconds()) / 1e6

	// Enviar resultado al padre
	c.Send(c.Parent(), DomainChecked{Check: check})

	// Log con métricas detalladas
	slog.Info("Domain check completed",
		"domain", msg.Name,
		"total_duration_ms", totalDuration,
		"dns_duration_ms", dnsDuration,
		"success", check.IsValid,
		"ips_count", len(ips))

	// Alertas
	if !check.IsValid {
		alertMsg := fmt.Sprintf("ALERTA: Dominio %s tiene IPs inesperadas. Esperado: %s, Obtenido: %v (Tiempo: %dms)",
			msg.Name, s.config.ExpectedIP, ips, totalDuration)
		c.Send(c.Parent(), Alert{
			Message: alertMsg,
			Level:   "WARNING",
		})
	}

	// Auto-destruirse
	c.Engine().Poison(c.PID())
}

func (s *SingleDomainChecker) validateIPs(actualIPs []string, expectedIP string) bool {
	for _, ip := range actualIPs {
		if ip == expectedIP {
			return true
		}
	}
	return false
}

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

	check := domain.DomainCheck{
		Domain:     s.config.Domain,
		ExpectedIP: s.config.ExpectedIP,
		Timestamp:  time.Now(),
	}

	// Resolver IPs
	ips, err := s.dnsResolver.ResolveIP(msg.Name)
	if err != nil {
		check.Error = err.Error()
		check.IsValid = false
	} else {
		check.ActualIPs = ips
		check.IsValid = s.validateIPs(ips, s.config.ExpectedIP)
	}

	// Enviar resultado al padre (MonitorActor)
	c.Send(c.Parent(), DomainChecked{Check: check})

	// Enviar alerta si es inválido
	if !check.IsValid {
		c.Send(c.Parent(), Alert{
			Message: fmt.Sprintf("ALERTA: Dominio %s tiene IPs inesperadas. Esperado: %s, Obtenido: %v",
				msg.Name, s.config.ExpectedIP, ips),
			Level: "WARNING",
		})
	}

	// Auto-destruirse después de procesar (opcional)
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

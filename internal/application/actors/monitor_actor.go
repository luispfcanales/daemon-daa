package actors

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/repositories"

	"github.com/anthdm/hollywood/actor"
)

type MonitorActor struct {
	repository *repositories.InMemoryDomainRepository
	monitoring bool
	interval   time.Duration
	stopCh     chan struct{}
	// Ya no necesitamos un DomainChecker fijo
}

func NewMonitorActor(repo *repositories.InMemoryDomainRepository) actor.Producer {
	return func() actor.Receiver {
		return &MonitorActor{
			repository: repo,
			monitoring: false,
			stopCh:     make(chan struct{}),
		}
	}
}

func (m *MonitorActor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("MonitorActor started", "pid", c.PID())
		// Ya no creamos un DomainChecker fijo

	case StartMonitoring:
		m.handleStartMonitoring(c, msg)

	case StopMonitoring:
		m.handleStopMonitoring(c)

	case CheckAllDomains:
		m.handleCheckAllDomains(c) // ¡MEJORADO!

	case CheckDomain:
		m.handleCheckSingleDomain(c, msg)

	case DomainChecked:
		m.handleDomainChecked(c, msg)

	case Alert:
		m.handleAlert(c, msg)

	case GetStatus:
		m.handleGetStatus(c)

	case actor.Stopped:
		m.handleStopMonitoring(c)
		slog.Info("MonitorActor stopped", "pid", c.PID())
	}
}

// NUEVO: Manejo concurrente mejorado
func (m *MonitorActor) handleCheckAllDomains(c *actor.Context) {
	configs, err := m.repository.GetDomainConfigs()
	if err != nil {
		slog.Error("Error getting domain configs", "error", err)
		return
	}

	slog.Info("Starting concurrent domain check", "domains", len(configs))

	// Crear un SingleDomainChecker por cada dominio - CONCURRENCIA MÁXIMA
	for _, config := range configs {
		// Crear un actor temporal para este dominio específico
		checkerPID := c.SpawnChild(
			NewSingleDomainChecker(config),
			"checker-"+config.Domain,
			actor.WithInboxSize(1024), // Buffer para alta concurrencia
		)

		// Enviar el mensaje de verificación
		c.Send(checkerPID, CheckDomain{Name: config.Domain})

		slog.Debug("Spawned domain checker",
			"domain", config.Domain,
			"pid", checkerPID)
	}
}

// NUEVO: Manejo de dominio individual
func (m *MonitorActor) handleCheckSingleDomain(c *actor.Context, msg CheckDomain) {
	configs, err := m.repository.GetDomainConfigs()
	if err != nil {
		c.Send(c.Parent(), Alert{
			Message: fmt.Sprintf("Error getting configs: %v", err),
			Level:   "ERROR",
		})
		return
	}

	var config domain.DomainConfig
	for _, cfg := range configs {
		if cfg.Domain == msg.Name {
			config = cfg
			break
		}
	}

	if config.Domain == "" {
		c.Send(c.Parent(), Alert{
			Message: fmt.Sprintf("Domain %s not found in configuration", msg.Name),
			Level:   "ERROR",
		})
		return
	}

	// Crear checker temporal para este dominio único
	checkerPID := c.SpawnChild(NewSingleDomainChecker(config), "checker-single-"+msg.Name)
	c.Send(checkerPID, CheckDomain{Name: msg.Name})
}

func (m *MonitorActor) handleStartMonitoring(c *actor.Context, msg StartMonitoring) {
	if m.monitoring {
		slog.Warn("Monitoring already started")
		return
	}

	m.interval = time.Duration(msg.Interval) * time.Second
	m.monitoring = true

	slog.Info("Starting concurrent domain monitoring", "interval", m.interval)

	go m.monitoringLoop(c)
}

func (m *MonitorActor) handleStopMonitoring(c *actor.Context) {
	if m.monitoring {
		slog.Info("Stopping domain monitoring")
		m.monitoring = false
		close(m.stopCh)
		m.stopCh = make(chan struct{})
	}
}

func (m *MonitorActor) monitoringLoop(c *actor.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if m.monitoring {
				slog.Debug("Monitoring tick - checking all domains")
				c.Send(c.PID(), CheckAllDomains{})
			}
		case <-m.stopCh:
			return
		}
	}
}

func (m *MonitorActor) handleDomainChecked(c *actor.Context, msg DomainChecked) {
	// Guardar en el repositorio
	m.repository.SaveDomainCheck(msg.Check)

	// Log del resultado
	status := "✅ VÁLIDO"
	if !msg.Check.IsValid {
		status = "❌ INVÁLIDO"
	}

	slog.Info("Domain check completed",
		"domain", msg.Check.Domain,
		"status", status,
		"expected", msg.Check.ExpectedIP,
		"actual", msg.Check.ActualIPs,
		"duration", time.Since(msg.Check.Timestamp))
}

func (m *MonitorActor) handleAlert(c *actor.Context, msg Alert) {
	emoji := "⚠️"
	if msg.Level == "ERROR" {
		emoji = "🚨"
	}

	slog.Warn("ALERTA", "level", msg.Level, "message", msg.Message)

	// También podríamos enviar esto a un actor de notificaciones
	fmt.Printf("\n%s %s: %s\n\n", emoji, msg.Level, msg.Message)
}

func (m *MonitorActor) handleGetStatus(c *actor.Context) {
	checks := m.repository.GetChecks()
	slog.Info("Current status", "total_checks", len(checks))
}

package actors

import (
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/application/events"
	"github.com/luispfcanales/daemon-daa/internal/core/domain"
	"github.com/luispfcanales/daemon-daa/internal/core/ports"

	"github.com/anthdm/hollywood/actor"
)

type MonitorActor struct {
	repository   ports.DomainRepository
	emailService ports.EmailService
	monitoring   bool
	interval     time.Duration
	stopCh       chan struct{}
	eventBus     *events.EventBus
	startedAt    time.Time
	recipients   []string
}

func NewMonitorActor(
	repo ports.DomainRepository,
	eventBus *events.EventBus,
	emailService ports.EmailService,
	recipients []string,
) actor.Producer {
	return func() actor.Receiver {
		return &MonitorActor{
			repository:   repo,
			stopCh:       make(chan struct{}),
			eventBus:     eventBus,
			emailService: emailService,
			startedAt:    time.Time{},
			recipients:   recipients,
		}
	}
}

func (m *MonitorActor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("MonitorActor started", "pid", c.PID())

	case StartMonitoring:
		m.handleStartMonitoring(c, msg)

	case StopMonitoring:
		m.handleStopMonitoring(c)

	case CheckAllDomains:
		m.handleCheckAllDomains(c) // ¡MEJORADO!

	case DomainChecked:
		m.handleDomainChecked(c, msg)

	case Alert:
		m.handleAlert(c, msg)

	case GetMonitoringStatus:
		m.handleGetMonitoringStatus(c)

	case GetStatsDomain:
		m.handleStatsDomain(c, msg)

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

func (m *MonitorActor) handleStartMonitoring(c *actor.Context, msg StartMonitoring) {
	if m.monitoring {
		slog.Warn("Monitoring already started", "interval", m.interval)
		// Responder que ya está corriendo
		if c.Sender() != nil {
			c.Send(c.Sender(), MonitoringStatus{
				IsRunning: true,
				Interval:  m.interval,
				StartedAt: m.startedAt,
				Message:   "Monitoring already running",
			})
		}
		return
	}

	m.interval = time.Duration(msg.Interval) * time.Second
	m.monitoring = true
	m.startedAt = time.Now()

	m.sendMonitoringNotification(c)
	slog.Info("Starting concurrent domain monitoring", "interval", m.interval)

	go m.monitoringLoop(c)
	// Responder éxito
	if c.Sender() != nil {
		c.Send(c.Sender(), MonitoringStatus{
			IsRunning: true,
			Interval:  m.interval,
			StartedAt: m.startedAt,
			Message:   "Monitoring started",
		})
	}
}

func (m *MonitorActor) sendMonitoringNotification(c *actor.Context) {
	if m.emailService == nil || len(m.recipients) == 0 {
		slog.Warn("Email service not configured or no recipients, skipping notification")
		return
	}

	status := domain.MonitoringStatus{
		IsRunning: m.monitoring,
		Interval:  m.interval,
		StartedAt: m.startedAt,
	}

	if m.monitoring {
		status.Message = fmt.Sprintf("Monitoreo iniciado con intervalo de %v. Verificando dominios configurados.", m.interval)
	} else {
		status.Message = "Monitoreo detenido"
	}

	go func() {
		err := m.emailService.SendMonitoringNotification(m.recipients, status)
		if err != nil {
			slog.Error("Error sending monitoring start notification", "error", err)
			c.Send(c.PID(), Alert{
				Level:   "ERROR",
				Message: fmt.Sprintf("Error enviando notificación por email: %v", err),
			})
		} else {
			slog.Info("Monitoring start notification sent successfully")
		}
	}()
}

func (m *MonitorActor) handleStopMonitoring(c *actor.Context) {
	if m.monitoring {
		slog.Info("Stopping domain monitoring")
		m.monitoring = false
		close(m.stopCh)
		m.stopCh = make(chan struct{})

		m.sendMonitoringNotification(c)
		// Responder éxito si hay un solicitante
		if c.Sender() != nil {
			c.Send(c.Sender(), MonitoringStatus{
				IsRunning: false,
				Interval:  0,
				Message:   "Monitoring stopped",
			})
		}
	} else if c.Sender() != nil {
		// Ya está detenido
		c.Send(c.Sender(), MonitoringStatus{
			IsRunning: false,
			Interval:  0,
			Message:   "Monitoring already stopped",
		})
	}
}

func (m *MonitorActor) handleGetMonitoringStatus(c *actor.Context) {
	status := MonitoringStatus{
		IsRunning: m.monitoring,
		Interval:  m.interval,
	}

	if m.monitoring {
		status.StartedAt = m.startedAt
	}

	c.Send(c.Sender(), status)
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

func (m *MonitorActor) handleDomainChecked(_ *actor.Context, msg DomainChecked) {

	// Guardar en el repositorio
	m.repository.SaveDomainCheck(msg.Check)

	// Log del resultado
	status := "✅ VÁLIDO"
	if !msg.Check.IsValid {
		status = "❌ INVÁLIDO"
	}

	//duration := time.Since(msg.Check.Timestamp).String()
	duration := float64(time.Since(msg.Check.Timestamp).Microseconds()) / 1000.0
	msg.Check.RequestTime = duration

	m.eventBus.Broadcast(events.Event{
		Type:      "monitoring_ip",
		Data:      msg,
		Timestamp: time.Time{},
	})
	slog.Info("Domain check completed",
		"domain", msg.Check.Domain,
		"status", status,
		"expected", msg.Check.ExpectedIP,
		"actual", msg.Check.ActualIPs,
		"duration", duration,
	)
}

func (m *MonitorActor) handleAlert(_ *actor.Context, msg Alert) {
	emoji := "⚠️"
	if msg.Level == "ERROR" {
		emoji = "🚨"
	}

	slog.Warn("ALERTA", "level", msg.Level, "message", msg.Message)

	// También podríamos enviar esto a un actor de notificaciones
	fmt.Printf("\n%s %s: %s\n\n", emoji, msg.Level, msg.Message)
}

func (m *MonitorActor) handleStatsDomain(_ *actor.Context, msg GetStatsDomain) {

	data, err := m.repository.GetDomainStats("intranet.unamad.edu.pe")
	if err != nil {
		slog.Error("Error getting info Stats", "error", err)
	}
	log.Println(data)
}

func (m *MonitorActor) handleGetStatus(_ *actor.Context) {
	checks, _ := m.repository.GetChecks()
	slog.Info("Current status", "total_checks", len(checks))
}

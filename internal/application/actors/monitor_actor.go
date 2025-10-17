package actors

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/application/events"
	"github.com/luispfcanales/daemon-daa/internal/core/domain"
	"github.com/luispfcanales/daemon-daa/internal/core/ports"

	"github.com/anthdm/hollywood/actor"
)

type MonitorActor struct {
	repository      ports.DomainRepository
	emailService    ports.EmailService
	monitoring      bool
	interval        time.Duration
	cancelFunc      context.CancelFunc
	eventBus        *events.EventBus
	startedAt       time.Time
	recipientsEmail []string
	domainCheckers  map[string]*actor.PID
}

func NewMonitorActor(
	repo ports.DomainRepository,
	eventBus *events.EventBus,
	emailService ports.EmailService,
	recipients []string,
) actor.Producer {
	return func() actor.Receiver {
		return &MonitorActor{
			repository:      repo,
			eventBus:        eventBus,
			emailService:    emailService,
			startedAt:       time.Time{},
			recipientsEmail: recipients,
			domainCheckers:  make(map[string]*actor.PID),
		}
	}
}

func (m *MonitorActor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("MonitorActor started", "pid", c.PID())
		m.initializeDomainCheckers(c)

	case StartMonitoring:
		m.handleStartMonitoring(c, msg)

	case StopMonitoring:
		m.handleStopMonitoring(c)

	case CheckAllDomains:
		m.triggerDomainChecks(c)

	case DomainChecked:
		m.handleDomainChecked(c, msg)

	case Alert:
		m.handleAlert(c, msg)

	case GetMonitoringStatus:
		m.handleGetMonitoringStatus(c)

	case GetAllCachedStats:
		m.handleGetAllCachedStats(c)

	case actor.Stopped:
		m.handleStopMonitoring(c)
		slog.Info("MonitorActor stopped", "pid", c.PID())
	}
}

func (m *MonitorActor) initializeDomainCheckers(c *actor.Context) {
	configs, err := m.repository.GetDomainConfigs()
	if err != nil {
		slog.Error("Error getting domain configs", "error", err)
		return
	}

	for _, config := range configs {
		checkerPID := c.SpawnChild(
			NewSingleDomainChecker(
				config,
				m.repository,
				m.eventBus,
			),
			"checker-"+config.Domain,
			actor.WithInboxSize(1024),
		)
		m.domainCheckers[config.Domain] = checkerPID
	}
}

func (m *MonitorActor) triggerDomainChecks(c *actor.Context) {
	slog.Info("Starting concurrent domain check", "domains", len(m.domainCheckers))
	for _, pid := range m.domainCheckers {
		c.Send(pid, CheckDomain{})
	}
}

func (m *MonitorActor) handleGetAllCachedStats(c *actor.Context) {
	slog.Debug("Collecting cached stats from all domain checkers",
		"total_domains", len(m.domainCheckers))

	if len(m.domainCheckers) == 0 {
		slog.Warn("No domain checkers available")
		if c.Sender() != nil {
			c.Send(c.Sender(), AllCachedStatsResponse{
				Stats: []*domain.StatsDomain{},
			})
		}
		return
	}

	statsMap := []*domain.StatsDomain{}
	type domainResponse struct {
		domain   string
		response CachedStatsResponse
	}
	responseChan := make(chan domainResponse, len(m.domainCheckers))

	for domainName, pid := range m.domainCheckers {
		domain := domainName

		go func(d string, p *actor.PID) {
			response := c.Request(p, GetCachedStats{}, 1*time.Second)

			result, err := response.Result()

			if err != nil {
				slog.Warn("Error requesting stats",
					"domain", d,
					"error", err)
				responseChan <- domainResponse{
					domain:   d,
					response: CachedStatsResponse{Found: false},
				}
				return
			}

			if cachedStats, ok := result.(CachedStatsResponse); ok {
				responseChan <- domainResponse{
					domain:   d,
					response: cachedStats,
				}
			} else {
				slog.Warn("Invalid response type",
					"domain", d,
					"type", fmt.Sprintf("%T", result))
				responseChan <- domainResponse{
					domain:   d,
					response: CachedStatsResponse{Found: false},
				}
			}
		}(domain, pid)
	}

	// Recolectar respuestas con timeout global
	timeout := time.After(3 * time.Second)
	collected := 0
	expectedCount := len(m.domainCheckers)

	for collected < expectedCount {
		select {
		case result := <-responseChan:
			if result.response.Found && result.response.Stats != nil {
				statsMap = append(statsMap, result.response.Stats)
				slog.Debug("Stats collected",
					"domain", result.domain,
					"total_checks", result.response.Stats.TotalChecks)
			} else {
				slog.Debug("No stats available for domain", "domain", result.domain)
			}
			collected++

		case <-timeout:
			slog.Warn("Timeout collecting cached stats",
				"collected", collected,
				"expected", expectedCount,
				"missing", expectedCount-collected)
			goto sendResponse
		}
	}

sendResponse:
	slog.Info("Cached stats collection complete",
		"domains_with_stats", len(statsMap),
		"total_domains", expectedCount)

	if c.Sender() != nil {
		c.Send(c.Sender(), AllCachedStatsResponse{Stats: statsMap})
	}
}

func (m *MonitorActor) handleStartMonitoring(c *actor.Context, msg StartMonitoring) {
	if m.monitoring {
		slog.Warn("Monitoring already started", "interval", m.interval)

		if c.Sender() != nil {
			c.Send(c.Sender(), domain.MonitoringStatus{
				IsRunning: true,
				Interval:  m.interval,
				StartedAt: m.startedAt,
				Message:   "Monitoring already running",
			})
		}
		return
	}

	var ctx context.Context
	ctx, m.cancelFunc = context.WithCancel(context.Background())

	m.interval = time.Duration(msg.Interval) * time.Second
	m.monitoring = true
	m.startedAt = time.Now()

	m.sendMonitoringNotification(c)
	slog.Info("Starting concurrent domain monitoring", "interval", m.interval)

	go m.monitoringLoop(c, ctx)

	if c.Sender() != nil {
		c.Send(c.Sender(), domain.MonitoringStatus{
			IsRunning: true,
			Interval:  m.interval,
			StartedAt: m.startedAt,
			Message:   "Monitoring started",
		})
	}
}

func (m *MonitorActor) sendMonitoringNotification(c *actor.Context) {
	if m.emailService == nil || len(m.recipientsEmail) == 0 {
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
		err := m.emailService.SendMonitoringNotification(m.recipientsEmail, status)
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
	if m.monitoring && m.cancelFunc != nil {
		slog.Info("Stopping domain monitoring")
		m.monitoring = false

		m.cancelFunc()
		m.cancelFunc = nil

		m.sendMonitoringNotification(c)

		if c.Sender() != nil {
			c.Send(c.Sender(), domain.MonitoringStatus{
				IsRunning: false,
				Interval:  0,
				Message:   "Monitoring stopped",
			})
		}
	} else if c.Sender() != nil {

		c.Send(c.Sender(), domain.MonitoringStatus{
			IsRunning: false,
			Interval:  0,
			Message:   "Monitoring already stopped",
		})
	}
}

func (m *MonitorActor) handleGetMonitoringStatus(c *actor.Context) {
	status := domain.MonitoringStatus{
		IsRunning: m.monitoring,
		Interval:  m.interval,
	}

	if m.monitoring {
		status.StartedAt = m.startedAt
	}

	c.Send(c.Sender(), status)
}

func (m *MonitorActor) monitoringLoop(c *actor.Context, ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !m.monitoring {
				return
			}
			slog.Debug("Monitoring tick - checking all domains")
			c.Send(c.PID(), CheckAllDomains{})

		case <-ctx.Done():
			return
		}
	}
}

func (m *MonitorActor) handleDomainChecked(_ *actor.Context, msg DomainChecked) {
	// Log del resultado
	status := "✅ VÁLIDO"
	if !msg.Check.IsValid {
		status = "❌ INVÁLIDO"
	}

	duration := float64(time.Since(msg.Check.Timestamp).Microseconds()) / 1000.0
	msg.Check.RequestTime = duration

	m.eventBus.Broadcast(events.Event{
		Type: "monitoring_ip",
		Data: msg,
	})

	slog.Info("Domain check completed",
		"domain", msg.Check.Domain,
		"status", status,
		"expected", msg.Check.ExpectedIP,
		"actual", msg.Check.ActualIPs,
		"duration_request", duration,
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

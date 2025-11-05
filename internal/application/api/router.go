package api

import (
	"net/http"

	"github.com/anthdm/hollywood/actor"
	"github.com/luispfcanales/daemon-daa/internal/application/events"
	"github.com/luispfcanales/daemon-daa/internal/core/ports"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/services"
)

type Router struct {
	handler *APIHandler
}

func NewRouter(
	engine *actor.Engine,
	monitorPID *actor.PID,
	iisService *services.IISService,
	eventBus *events.EventBus,
	ipService ports.IPService,
	emailService ports.IEmailService,
) *Router {
	return &Router{
		handler: NewAPIHandler(
			engine,
			monitorPID,
			iisService,
			eventBus,
			ipService,
			emailService,
		),
	}
}

func (r *Router) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Eventos en tiempo real
	mux.HandleFunc("GET /monitoring/events", r.handler.MonitoringEvents)

	// Control del sistema
	mux.HandleFunc("POST /monitoring/control", r.handler.ControlMonitoring)
	mux.HandleFunc("POST /iis/control", r.handler.ControlIIS)
	mux.HandleFunc("GET /iis/sites", r.handler.GetIISSites)

	//control de dominios
	mux.HandleFunc("GET /domain/list", r.handler.GetDomainsList)
	mux.HandleFunc("POST /domain/add", r.handler.AddDomain)
	mux.HandleFunc("DELETE /domain/delete/{dns}", r.handler.DeleteDomain)
	mux.HandleFunc("PUT /domain/update/{id}", r.handler.UpdateDomain)

	//Notify SMS
	mux.HandleFunc("POST /notify/sms", r.handler.handleSendSMS)

	// Rutas de Email (siguiendo el mismo patrón que DNS)
	mux.HandleFunc("GET /api/email/sender-config", r.handler.GetSenderConfig)
	mux.HandleFunc("POST /api/email/sender-config", r.handler.UpdateSenderConfig)
	mux.HandleFunc("GET /api/email/notification-emails", r.handler.GetNotificationEmails)
	mux.HandleFunc("POST /api/email/notification-emails", r.handler.AddNotificationEmail)
	mux.HandleFunc("DELETE /api/email/notification-emails", r.handler.RemoveNotificationEmail)

	// Ruta por defecto
	mux.HandleFunc("/", r.handler.NotFound)

	return mux
}

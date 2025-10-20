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
) *Router {
	return &Router{
		handler: NewAPIHandler(
			engine,
			monitorPID,
			iisService,
			eventBus,
			ipService,
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

	// Ruta por defecto
	mux.HandleFunc("/", r.handler.NotFound)

	return mux
}

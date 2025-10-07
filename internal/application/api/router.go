package api

import (
	"net/http"

	"github.com/anthdm/hollywood/actor"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/services"
)

// Router configura las rutas del API
type Router struct {
	handler *APIHandler
}

func NewRouter(engine *actor.Engine, monitorPID *actor.PID, iisService *services.IISService) *Router {
	return &Router{
		handler: NewAPIHandler(engine, monitorPID, iisService),
	}
}

// SetupRoutes configura todas las rutas del API
func (r *Router) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Rutas del API
	mux.HandleFunc("POST /monitoring/control", r.handler.ControlMonitoring)
	mux.HandleFunc("POST /iis/control", r.handler.ControlIIS)
	mux.HandleFunc("GET /iis/sites", r.handler.GetIISSites)

	mux.HandleFunc("/", r.handler.NotFound)

	return mux
}

// NotFound maneja rutas no encontradas
func (h *APIHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	h.sendError(w, "Ruta no encontrada", http.StatusNotFound)
}

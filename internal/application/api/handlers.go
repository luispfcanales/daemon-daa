package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/application/actors"
	"github.com/luispfcanales/daemon-daa/internal/application/events"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/services"

	"github.com/anthdm/hollywood/actor"
)

type APIHandler struct {
	engine     *actor.Engine
	monitorPID *actor.PID
	iisService *services.IISService
	eventBus   *events.EventBus
}

func NewAPIHandler(engine *actor.Engine, monitorPID *actor.PID, iisService *services.IISService, eventBus *events.EventBus) *APIHandler {
	return &APIHandler{
		engine:     engine,
		monitorPID: monitorPID,
		iisService: iisService,
		eventBus:   eventBus,
	}
}

type MonitoringControlRequest struct {
	Action   string `json:"action"`
	Interval int    `json:"interval,omitempty"`
}

type IISControlRequest struct {
	SiteName string `json:"site_name"`
	Action   string `json:"action"`
}

type ErrorResponse struct {
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

// MonitoringEvents maneja las conexiones SSE para monitoreo en tiempo real
func (h *APIHandler) MonitoringEvents(w http.ResponseWriter, r *http.Request) {
	// Configurar headers para SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE no soportado", http.StatusInternalServerError)
		return
	}

	// Notificar conexión inmediatamente
	connectedEvent := events.Event{
		Type:      "connected",
		Data:      map[string]interface{}{"status": "connected"},
		Timestamp: time.Now(),
	}
	if data, err := json.Marshal(connectedEvent); err == nil {
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Suscribir cliente
	client := h.eventBus.Subscribe()
	defer func() {
		h.eventBus.Unsubscribe(client)
		slog.Info("Cliente SSE desconectado", "remote_addr", r.RemoteAddr)
	}()

	slog.Info("Cliente SSE conectado", "remote_addr", r.RemoteAddr)

	// Crear contexto con timeout para obtener estado inicial
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Enviar estado inicial
	go func() {
		time.Sleep(50 * time.Millisecond)

		if status, err := h.getCurrentMonitoringStatusWithContext(ctx); err == nil {
			initialEvent := events.Event{
				Type: "initial_status",
				Data: map[string]interface{}{
					"is_running": status.IsRunning,
					"interval":   int(status.Interval.Seconds()),
				},
				Timestamp: time.Now(),
			}

			// Enviar directamente al cliente
			select {
			case client <- initialEvent:
				slog.Info("Estado inicial enviado", "is_running", status.IsRunning)
			default:
				slog.Warn("No se pudo enviar estado inicial (cliente ocupado)")
			}
		} else {
			slog.Error("Error obteniendo estado inicial", "error", err)
		}
	}()

	// Heartbeat ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Loop principal
	done := r.Context().Done()

	for {
		select {
		case event, ok := <-client:
			if !ok {
				slog.Info("Canal cerrado")
				return
			}

			data, err := json.Marshal(event)
			if err != nil {
				slog.Error("Error serializando evento", "error", err)
				continue
			}

			n, err := fmt.Fprintf(w, "data: %s\n\n", data)
			if err != nil || n == 0 {
				slog.Error("Error escribiendo evento", "error", err)
				return
			}

			flusher.Flush()

		case <-ticker.C:
			n, err := fmt.Fprintf(w, ": heartbeat\n\n")
			if err != nil || n == 0 {
				slog.Error("Error enviando heartbeat", "error", err)
				return
			}
			flusher.Flush()

		case <-done:
			slog.Info("Cliente cerró conexión", "remote_addr", r.RemoteAddr)
			return
		}
	}
}

// getCurrentMonitoringStatusWithContext obtiene el estado con contexto
func (h *APIHandler) getCurrentMonitoringStatusWithContext(ctx context.Context) (*actors.MonitoringStatus, error) {
	future := h.engine.Request(h.monitorPID, actors.GetMonitoringStatus{}, 2*time.Second)

	result, err := future.Result()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo estado: %v", err)
	}

	if status, ok := result.(actors.MonitoringStatus); ok {
		return &status, nil
	}

	return nil, fmt.Errorf("tipo de respuesta inesperado")
}

func (h *APIHandler) ControlMonitoring(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req MonitoringControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	currentStatus, err := h.getCurrentMonitoringStatus()
	if err != nil {
		h.sendError(w, fmt.Sprintf("Error verificando estado: %v", err), http.StatusInternalServerError)
		return
	}

	switch req.Action {
	case "start":
		h.handleStartWithCheck(w, req.Interval, currentStatus)
	case "stop":
		h.handleStopWithCheck(w, currentStatus)
	case "status":
		h.handleStatusResponse(w, currentStatus)
	default:
		h.sendError(w, "Acción no válida. Use: start, stop, status", http.StatusBadRequest)
	}
}

func (h *APIHandler) handleStartWithCheck(w http.ResponseWriter, interval int, currentStatus *actors.MonitoringStatus) {
	if interval <= 0 {
		interval = 30
	}

	if currentStatus.IsRunning {
		response := map[string]interface{}{
			"success":    false,
			"action":     "start",
			"is_running": true,
			"interval":   currentStatus.Interval.Seconds(),
			"message":    "❌ El monitoreo YA está ejecutándose",
		}

		if !currentStatus.StartedAt.IsZero() {
			response["started_at"] = currentStatus.StartedAt.Format(time.RFC3339)
			response["running_for"] = time.Since(currentStatus.StartedAt).String()
		}

		slog.Warn("Intento de iniciar monitoreo activo")
		h.sendJSON(w, response, http.StatusConflict)
		return
	}

	slog.Info("Iniciando monitoreo", "interval", interval)
	h.engine.Send(h.monitorPID, actors.StartMonitoring{Interval: interval})

	h.eventBus.Broadcast(events.Event{
		Type: "monitoring_started",
		Data: map[string]interface{}{
			"interval":   interval,
			"is_running": true,
			"message":    fmt.Sprintf("✅ Monitoreo iniciado cada %d segundos", interval),
		},
		Timestamp: time.Now(),
	})

	response := map[string]interface{}{
		"success":    true,
		"action":     "start",
		"is_running": true,
		"interval":   interval,
		"message":    fmt.Sprintf("✅ Monitoreo iniciado cada %d segundos", interval),
	}

	h.sendJSON(w, response, http.StatusOK)
}

func (h *APIHandler) handleStopWithCheck(w http.ResponseWriter, currentStatus *actors.MonitoringStatus) {
	if !currentStatus.IsRunning {
		response := map[string]interface{}{
			"success":    false,
			"action":     "stop",
			"is_running": false,
			"message":    "❌ El monitoreo YA está detenido",
		}

		slog.Warn("Intento de detener monitoreo inactivo")
		h.sendJSON(w, response, http.StatusConflict)
		return
	}

	slog.Info("Deteniendo monitoreo")
	h.engine.Send(h.monitorPID, actors.StopMonitoring{})

	h.eventBus.Broadcast(events.Event{
		Type: "monitoring_stopped",
		Data: map[string]interface{}{
			"is_running": false,
			"interval":   0,
			"message":    "✅ Monitoreo detenido",
		},
		Timestamp: time.Now(),
	})

	response := map[string]interface{}{
		"success":    true,
		"action":     "stop",
		"is_running": false,
		"message":    "✅ Monitoreo detenido",
	}

	h.sendJSON(w, response, http.StatusOK)
}

func (h *APIHandler) handleStatusResponse(w http.ResponseWriter, status *actors.MonitoringStatus) {
	response := map[string]interface{}{
		"success":    true,
		"is_running": status.IsRunning,
		"interval":   status.Interval.Seconds(),
	}

	if status.IsRunning {
		response["message"] = "🟢 Monitoreo ACTIVO"
		if !status.StartedAt.IsZero() {
			response["started_at"] = status.StartedAt.Format(time.RFC3339)
			response["running_for"] = time.Since(status.StartedAt).String()
		}
	} else {
		response["message"] = "🔴 Monitoreo INACTIVO"
	}

	h.sendJSON(w, response, http.StatusOK)
}

func (h *APIHandler) getCurrentMonitoringStatus() (*actors.MonitoringStatus, error) {
	future := h.engine.Request(h.monitorPID, actors.GetMonitoringStatus{}, 3*time.Second)

	result, err := future.Result()
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener estado: %v", err)
	}

	if status, ok := result.(actors.MonitoringStatus); ok {
		return &status, nil
	}

	return nil, fmt.Errorf("respuesta inesperada del monitor")
}

func (h *APIHandler) GetIISSites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	sites, err := h.iisService.GetAllSites()
	if err != nil {
		h.sendError(w, fmt.Sprintf("Error obteniendo sitios IIS: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"sites":   sites,
		"count":   len(sites),
	}

	h.sendJSON(w, response, http.StatusOK)
}

func (h *APIHandler) ControlIIS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req IISControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if req.SiteName == "" {
		h.sendError(w, "site_name es requerido", http.StatusBadRequest)
		return
	}

	if req.Action != "start" && req.Action != "stop" && req.Action != "restart" {
		h.sendError(w, "acción no válida. Use: start, stop, restart", http.StatusBadRequest)
		return
	}

	slog.Info("Control IIS", "site", req.SiteName, "action", req.Action)

	result, err := h.iisService.ControlSite(req.SiteName, req.Action)
	if err != nil {
		h.sendError(w, fmt.Sprintf("Error ejecutando comando: %v", err), http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, result, http.StatusOK)
}

func (h *APIHandler) sendJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Error encoding JSON", "error", err)
	}
}

func (h *APIHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := ErrorResponse{
		Error:     message,
		Timestamp: time.Now(),
	}

	json.NewEncoder(w).Encode(errorResponse)
}

func (h *APIHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	h.sendError(w, "Ruta no encontrada", http.StatusNotFound)
}

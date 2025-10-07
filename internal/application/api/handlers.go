package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/application/actors"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/services"

	"github.com/anthdm/hollywood/actor"
)

type APIHandler struct {
	engine     *actor.Engine
	monitorPID *actor.PID
	iisService *services.IISService
}

func NewAPIHandler(engine *actor.Engine, monitorPID *actor.PID, iisService *services.IISService) *APIHandler {
	return &APIHandler{
		engine:     engine,
		monitorPID: monitorPID,
		iisService: iisService,
	}
}

// Estructuras de request/response
type MonitoringControlRequest struct {
	Action   string `json:"action"`             // "start", "stop", "status"
	Interval int    `json:"interval,omitempty"` // segundos, solo para "start"
}

// IISControlRequest representa la solicitud para controlar IIS
type IISControlRequest struct {
	SiteName string `json:"site_name"`
	Action   string `json:"action"` // "start", "stop", "restart"
}

// IISControlResponse representa la respuesta del control IIS
type IISControlResponse struct {
	Success   bool      `json:"success"`
	Action    string    `json:"action"`
	Message   string    `json:"message"`
	Output    string    `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// DomainCheckResponse representa la respuesta de verificación de dominios
type DomainCheckResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Results   interface{} `json:"results,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ErrorResponse representa una respuesta de error
type ErrorResponse struct {
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
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

	// 1. PRIMERO obtener estado actual
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

// ✅ Iniciar monitoreo
func (h *APIHandler) handleStartWithCheck(w http.ResponseWriter, interval int, currentStatus *actors.MonitoringStatus) {
	if interval <= 0 {
		interval = 30
	}

	// Verificar si ya está ejecutándose
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

		slog.Warn("Intento de iniciar monitoreo que ya está ejecutándose")
		h.sendJSON(w, response, http.StatusConflict)
		return
	}

	// Iniciar monitoreo
	slog.Info("Iniciando monitoreo", "interval", interval)
	h.engine.Send(h.monitorPID, actors.StartMonitoring{Interval: interval})

	response := map[string]interface{}{
		"success":  true,
		"action":   "start",
		"interval": interval,
		"message":  fmt.Sprintf("✅ Monitoreo iniciado cada %d segundos", interval),
	}

	h.sendJSON(w, response, http.StatusOK)
}

// ✅ Detener monitoreo
func (h *APIHandler) handleStopWithCheck(w http.ResponseWriter, currentStatus *actors.MonitoringStatus) {
	// Verificar si ya está detenido
	if !currentStatus.IsRunning {
		response := map[string]interface{}{
			"success":    false,
			"action":     "stop",
			"is_running": false,
			"message":    "❌ El monitoreo YA está detenido",
		}

		slog.Warn("Intento de detener monitoreo que ya está detenido")
		h.sendJSON(w, response, http.StatusConflict)
		return
	}

	// Detener monitoreo
	slog.Info("Deteniendo monitoreo")
	h.engine.Send(h.monitorPID, actors.StopMonitoring{})

	response := map[string]interface{}{
		"success": true,
		"action":  "stop",
		"message": "✅ Monitoreo detenido",
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
	// Usar Request que es más confiable
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

// ControlIIS maneja las solicitudes para controlar IIS
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

	slog.Info("Control IIS request", "site", req.SiteName, "action", req.Action)

	result, err := h.iisService.ControlSite(req.SiteName, req.Action)
	if err != nil {
		h.sendError(w, fmt.Sprintf("Error ejecutando comando: %v", err), http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, result, http.StatusOK)
}

// sendJSON envía una respuesta JSON
func (h *APIHandler) sendJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Error encoding JSON response", "error", err)
	}
}

// sendError envía una respuesta de error JSON
func (h *APIHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := ErrorResponse{
		Error:     message,
		Timestamp: time.Now(),
	}

	json.NewEncoder(w).Encode(errorResponse)
}

package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

type EmailSenderRequest struct {
	Email            string `json:"email,omitempty"`
	GmailAppPassword string `json:"gmail_app_password,omitempty"`
}

type NotificationEmailRequest struct {
	Email string `json:"email,omitempty"`
}

func (h *APIHandler) GetSenderConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.emailService.GetSenderConfig()
	if err != nil {
		h.sendError(
			w,
			"Error al obtener configuración del remitente",
			http.StatusInternalServerError,
		)
		return
	}

	if config == nil {
		h.sendJSON(w, map[string]any{
			"success": true,
			"config":  nil,
		}, http.StatusOK)
		return
	}

	h.sendJSON(w, map[string]any{
		"success": true,
		"config":  config,
	}, http.StatusOK)
}

func (h *APIHandler) UpdateSenderConfig(w http.ResponseWriter, r *http.Request) {
	var req EmailSenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.GmailAppPassword == "" {
		h.sendError(w, "Email y contraseña de aplicación son requeridos", http.StatusBadRequest)
		return
	}

	if !strings.Contains(req.Email, "@") {
		h.sendError(w, "Formato de email inválido", http.StatusBadRequest)
		return
	}

	config := &domain.EmailConfig{
		Email:            req.Email,
		GmailAppPassword: req.GmailAppPassword,
	}

	err := h.emailService.SaveSenderConfig(config)
	if err != nil {
		h.sendError(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	h.sendJSON(w, map[string]any{
		"success": true,
		"message": "Configuración guardada exitosamente",
	}, http.StatusCreated)
}

func (h *APIHandler) GetNotificationEmails(w http.ResponseWriter, r *http.Request) {
	emails, err := h.emailService.GetNotificationEmails()
	if err != nil {
		h.sendError(
			w,
			"Error al obtener lista de correos",
			http.StatusInternalServerError,
		)
		return
	}

	h.sendJSON(w, map[string]any{
		"success": true,
		"emails":  emails,
	}, http.StatusOK)
}

func (h *APIHandler) AddNotificationEmail(w http.ResponseWriter, r *http.Request) {
	var req NotificationEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		h.sendError(w, "Email es requerido", http.StatusBadRequest)
		return
	}

	if !strings.Contains(req.Email, "@") {
		h.sendError(w, "Formato de email inválido", http.StatusBadRequest)
		return
	}

	err := h.emailService.AddNotificationEmail(req.Email)
	if err != nil {
		h.sendError(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	h.sendJSON(w, map[string]any{
		"success": true,
		"message": "Correo agregado exitosamente",
	}, http.StatusCreated)
}

func (h *APIHandler) RemoveNotificationEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		h.sendError(w, "Email es requerido", http.StatusBadRequest)
		return
	}

	err := h.emailService.RemoveNotificationEmail(email)
	if err != nil {
		h.sendError(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	h.sendJSON(w, map[string]any{
		"success": true,
		"message": "Correo eliminado exitosamente",
	}, http.StatusOK)
}

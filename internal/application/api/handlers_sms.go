package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/application/events"
)

type SMSRequest struct {
	Number  string `json:"number,omitempty"`
	Message string `json:"message,omitempty"`
}

func (h *APIHandler) handleSendSMS(w http.ResponseWriter, r *http.Request) {
	var req SMSRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	h.eventBus.Broadcast(events.Event{
		Type:      "notify_sms",
		Data:      req,
		Timestamp: time.Now(),
	})

	h.sendJSON(w, "ready send message to sms", http.StatusOK)
}

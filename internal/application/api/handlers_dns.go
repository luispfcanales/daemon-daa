package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

type DNSRequest struct {
	ID         string `json:"id,omitempty"`
	DNS        string `json:"dns,omitempty"`
	ExpectedIP string `json:"expected_ip,omitempty"`
	Status     bool   `json:"status"`
}

func (h *APIHandler) GetDomainsList(w http.ResponseWriter, r *http.Request) {
	var res []DNSRequest
	list, err := h.ipService.ListDomains()
	if err != nil {
		h.sendError(
			w,
			"No hay dominios registrados",
			http.StatusNotFound,
		)
		return
	}

	for _, value := range list {
		res = append(res, DNSRequest{
			ID:         value.ID,
			DNS:        value.Domain,
			ExpectedIP: value.ExpectedIP,
			Status:     value.Status,
		})
	}

	h.sendJSON(w, res, http.StatusOK)
}

func (h *APIHandler) AddDomain(w http.ResponseWriter, r *http.Request) {
	var req domain.DomainConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	req.ID = uuid.NewString()
	err := h.ipService.AddDomain(req)
	if err != nil {
		h.sendError(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	h.sendJSON(w, req, http.StatusCreated)
}

func (h *APIHandler) UpdateDomain(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("id")
	if key == "" {
		h.sendError(w, "id no especificado", http.StatusBadRequest)
		return
	}

	var req domain.DomainConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	req.ID = key

	err := h.ipService.UpdateDomainIP(req)
	if err != nil {
		h.sendError(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	h.sendJSON(w, req, http.StatusCreated)
}

func (h *APIHandler) DeleteDomain(w http.ResponseWriter, r *http.Request) {
	domainName := r.PathValue("dns")
	if domainName == "" {
		h.sendError(w, "Dominio no especificado", http.StatusBadRequest)
		return
	}

	err := h.ipService.DeleteDomainIP(domainName)
	if err != nil {
		h.sendError(
			w,
			err.Error(),
			http.StatusNotFound,
		)
		return
	}

	h.sendJSON(w, "Dominio eliminado", http.StatusOK)
}

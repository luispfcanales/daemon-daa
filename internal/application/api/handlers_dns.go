package api

import (
	"encoding/json"
	"net/http"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

type DNSRequest struct {
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
	err := h.ipService.AddDomain(req)
	if err != nil {
		h.sendError(
			w,
			err.Error(),
			http.StatusNotFound,
		)
		return
	}

	h.sendJSON(w, req, http.StatusCreated)
}

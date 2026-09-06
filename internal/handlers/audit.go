package handlers

import (
	"net/http"
	"strconv"

	"supermarket/internal/services"
)

type AuditHandler struct{ service *services.AuditService }

func NewAuditHandler(s *services.AuditService) *AuditHandler { return &AuditHandler{service: s} }

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
	v, err := h.service.List(r.Context(), r.URL.Query().Get("from"), r.URL.Query().Get("to"), r.URL.Query().Get("module"), userID)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"audit": v})
}

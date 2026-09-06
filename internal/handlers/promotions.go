package handlers

import (
	"net/http"
	"strconv"

	"supermarket/internal/models"
	"supermarket/internal/services"
)

type PromotionHandler struct{ service *services.PromotionService }

func NewPromotionHandler(s *services.PromotionService) *PromotionHandler {
	return &PromotionHandler{service: s}
}

func (h *PromotionHandler) List(w http.ResponseWriter, r *http.Request) {
	productID, _ := strconv.ParseInt(r.URL.Query().Get("product_id"), 10, 64)
	activeOnly := r.URL.Query().Get("active") != "false"
	v, err := h.service.List(r.Context(), productID, activeOnly)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"promotions": v})
}

func (h *PromotionHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	r.URL.RawQuery = "active=true&" + r.URL.RawQuery
	h.List(w, r)
}

func (h *PromotionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		middlewareWriteBadID(w)
		return
	}
	v, err := h.service.Get(r.Context(), id)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *PromotionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var p models.Promotion
	if !decodeJSON(w, r, &p) {
		return
	}
	v, err := h.service.Create(r.Context(), p)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (h *PromotionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		middlewareWriteBadID(w)
		return
	}
	var p models.Promotion
	if !decodeJSON(w, r, &p) {
		return
	}
	if err = h.service.Update(r.Context(), id, p); err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *PromotionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		middlewareWriteBadID(w)
		return
	}
	if err = h.service.Delete(r.Context(), id); err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *PromotionHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		middlewareWriteBadID(w)
		return
	}
	if err = h.service.Activate(r.Context(), id); err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "active": true})
}

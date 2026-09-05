package handlers

import (
	"net/http"
	"strconv"

	"supermarket/internal/middleware"
	"supermarket/internal/models"
	"supermarket/internal/services"
)

type PurchasingHandler struct{ service *services.PurchasingService }

func NewPurchasingHandler(s *services.PurchasingService) *PurchasingHandler {
	return &PurchasingHandler{service: s}
}
func currentUserID(r *http.Request) int64 {
	u, _ := middleware.UserFromContext(r.Context())
	return u.ID
}

func (h *PurchasingHandler) ListDealers(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.ListDealers(r.Context())
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"dealers": v})
}
func (h *PurchasingHandler) CreateDealer(w http.ResponseWriter, r *http.Request) {
	var v models.Dealer
	if !decodeJSON(w, r, &v) {
		return
	}
	v, e := h.service.CreateDealer(r.Context(), v)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (h *PurchasingHandler) UpdateDealer(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	var v models.Dealer
	if !decodeJSON(w, r, &v) {
		return
	}
	if e = h.service.UpdateDealer(r.Context(), id, v); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
func (h *PurchasingHandler) DeleteDealer(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	if e = h.service.DeleteDealer(r.Context(), id); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
func (h *PurchasingHandler) ActivateDealer(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	if e = h.service.ActivateDealer(r.Context(), id); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "active": true})
}

type purchaseRequest struct {
	DealerID            int64                 `json:"dealer_id"`
	DealerInvoiceNumber string                `json:"dealer_invoice_number"`
	Discount            float64               `json:"discount"`
	Items               []models.PurchaseItem `json:"items"`
}

func (h *PurchasingHandler) ListPurchases(w http.ResponseWriter, r *http.Request) {
	dealerID, _ := strconv.ParseInt(r.URL.Query().Get("dealer_id"), 10, 64)
	v, e := h.service.ListPurchases(r.Context(), dealerID)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"purchases": v})
}
func (h *PurchasingHandler) GetPurchase(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	v, e := h.service.GetPurchase(r.Context(), id)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (h *PurchasingHandler) CreatePurchase(w http.ResponseWriter, r *http.Request) {
	var req purchaseRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p := models.Purchase{DealerID: req.DealerID, DealerInvoiceNumber: req.DealerInvoiceNumber, Discount: req.Discount, Items: req.Items}
	v, e := h.service.CreatePurchase(r.Context(), currentUserID(r), p)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 201, v)
}

type paymentRequest struct {
	Amount float64 `json:"amount"`
	Method string  `json:"method"`
}

func (h *PurchasingHandler) PayPurchase(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	var req paymentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	v, e := h.service.PayPurchase(r.Context(), id, req.Amount, req.Method)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (h *PurchasingHandler) ReturnPurchase(w http.ResponseWriter, r *http.Request) {
	var v models.PurchaseReturn
	if !decodeJSON(w, r, &v) {
		return
	}
	v, e := h.service.ReturnPurchase(r.Context(), currentUserID(r), v)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 201, v)
}

func (h *PurchasingHandler) ListReturns(w http.ResponseWriter, r *http.Request) {
	purchaseID, _ := strconv.ParseInt(r.URL.Query().Get("purchase_id"), 10, 64)
	v, err := h.service.ListReturns(r.Context(), purchaseID)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"returns": v})
}

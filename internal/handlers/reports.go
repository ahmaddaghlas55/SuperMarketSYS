package handlers

import (
	"net/http"

	"supermarket/internal/services"
)

type ReportsHandler struct {
	service *services.ReportsService
	audit   *services.AuditService
}

func NewReportsHandler(s *services.ReportsService, audit *services.AuditService) *ReportsHandler {
	return &ReportsHandler{service: s, audit: audit}
}

func (h *ReportsHandler) Sales(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.Sales(r.Context(), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (h *ReportsHandler) Purchases(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.Purchases(r.Context(), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (h *ReportsHandler) Stock(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.Stock(r.Context())
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (h *ReportsHandler) Profit(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.Profit(r.Context(), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (h *ReportsHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.Dashboard(r.Context(), r.URL.Query().Get("date"))
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *ReportsHandler) export(w http.ResponseWriter, r *http.Request, name string, fn func() ([]byte, error)) {
	data, err := fn()
	if err != nil {
		serviceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.xlsx"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
func (h *ReportsHandler) ExportSales(w http.ResponseWriter, r *http.Request) {
	h.export(w, r, "sales", func() ([]byte, error) {
		return h.service.ExportSales(r.Context(), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	})
}
func (h *ReportsHandler) ExportDebt(w http.ResponseWriter, r *http.Request) {
	h.export(w, r, "debt", func() ([]byte, error) { return h.service.ExportDebt(r.Context()) })
}
func (h *ReportsHandler) ExportPurchases(w http.ResponseWriter, r *http.Request) {
	h.export(w, r, "dealers-purchases", func() ([]byte, error) {
		return h.service.ExportPurchases(r.Context(), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	})
}
func (h *ReportsHandler) ExportExpenses(w http.ResponseWriter, r *http.Request) {
	h.export(w, r, "expenses", func() ([]byte, error) {
		return h.service.ExportExpenses(r.Context(), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	})
}
func (h *ReportsHandler) ExportReturns(w http.ResponseWriter, r *http.Request) {
	h.export(w, r, "returns", func() ([]byte, error) {
		return h.service.ExportReturns(r.Context(), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	})
}

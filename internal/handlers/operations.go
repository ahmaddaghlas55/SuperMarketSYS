package handlers

import (
	"net/http"
	"strconv"

	"supermarket/internal/models"
	"supermarket/internal/services"
)

type OperationsHandler struct{ service *services.OperationsService }

func NewOperationsHandler(s *services.OperationsService) *OperationsHandler {
	return &OperationsHandler{service: s}
}
func (h *OperationsHandler) CreateSale(w http.ResponseWriter, r *http.Request) {
	var in services.SaleInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, e := h.service.CreateSale(r.Context(), currentUserID(r), in)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}
func (h *OperationsHandler) GetSale(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	v, e := h.service.GetSale(r.Context(), id)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (h *OperationsHandler) ListSales(w http.ResponseWriter, r *http.Request) {
	customerID, _ := strconv.ParseInt(r.URL.Query().Get("customer_id"), 10, 64)
	v, e := h.service.ListSales(r.Context(), customerID, r.URL.Query().Get("date"), r.URL.Query().Get("payment_status"))
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sales": v})
}
func middlewareWriteBadID(w http.ResponseWriter) {
	writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "invalid_id"}})
}
func (h *OperationsHandler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.ListCustomers(r.Context())
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"customers": v})
}
func (h *OperationsHandler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	c, _, e := h.service.CustomerDebt(r.Context(), id)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
func (h *OperationsHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var c models.Customer
	if !decodeJSON(w, r, &c) {
		return
	}
	v, e := h.service.CreateCustomer(r.Context(), c)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (h *OperationsHandler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	var c models.Customer
	if !decodeJSON(w, r, &c) {
		return
	}
	if e = h.service.UpdateCustomer(r.Context(), id, c); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
func (h *OperationsHandler) DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	if e = h.service.DeactivateCustomer(r.Context(), id); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
func (h *OperationsHandler) ActivateCustomer(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	if e = h.service.ActivateCustomer(r.Context(), id); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "active": true})
}
func (h *OperationsHandler) CustomerDebt(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	c, s, e := h.service.CustomerDebt(r.Context(), id)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"customer": c, "sales": s})
}

type debtPaymentInput struct {
	SaleID int64   `json:"sale_id"`
	Amount float64 `json:"amount"`
	Method string  `json:"method"`
}

func (h *OperationsHandler) PayDebt(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	var in debtPaymentInput
	if !decodeJSON(w, r, &in) {
		return
	}
	amount, e := h.service.PayDebt(r.Context(), currentUserID(r), id, in.SaleID, in.Amount, in.Method)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"customer_id": id, "applied_amount": amount})
}
func (h *OperationsHandler) CreateReturnForSale(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	var in services.ReturnInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, e := h.service.ReturnSale(r.Context(), currentUserID(r), id, in)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}
func (h *OperationsHandler) CreateReturn(w http.ResponseWriter, r *http.Request) {
	var in services.ReturnInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, e := h.service.CreateSalesReturn(r.Context(), currentUserID(r), in)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 201, v)
}

type openShiftInput struct {
	OpeningBalance float64 `json:"opening_balance"`
	OpeningNotes   string  `json:"opening_notes"`
}

func (h *OperationsHandler) OpenShift(w http.ResponseWriter, r *http.Request) {
	var in openShiftInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, e := h.service.OpenShift(r.Context(), currentUserID(r), in.OpeningBalance, in.OpeningNotes)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (h *OperationsHandler) CurrentShift(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.CurrentShift(r.Context())
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

type closeShiftInput struct {
	ActualClosing float64 `json:"actual_closing"`
	ClosingNotes  string  `json:"closing_notes"`
}

func (h *OperationsHandler) CloseShift(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	var in closeShiftInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, e := h.service.CloseShift(r.Context(), currentUserID(r), id, in.ActualClosing, in.ClosingNotes)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (h *OperationsHandler) ListShifts(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.ListShifts(r.Context())
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"shifts": v})
}
func (h *OperationsHandler) ListCashMovements(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	v, e := h.service.ListCashMovements(r.Context(), id)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"movements": v})
}
func (h *OperationsHandler) CreateManualCashMovement(w http.ResponseWriter, r *http.Request) {
	var in services.ManualCashMovementInput
	if !decodeJSON(w, r, &in) {
		return
	}
	v, e := h.service.CreateManualCashMovement(r.Context(), currentUserID(r), in)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}
func (h *OperationsHandler) CreateExpenseCategory(w http.ResponseWriter, r *http.Request) {
	var c models.ExpenseCategory
	if !decodeJSON(w, r, &c) {
		return
	}
	v, e := h.service.CreateExpenseCategory(r.Context(), c)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (h *OperationsHandler) UpdateExpenseCategory(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	var c models.ExpenseCategory
	if !decodeJSON(w, r, &c) {
		return
	}
	if e = h.service.UpdateExpenseCategory(r.Context(), id, c); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
func (h *OperationsHandler) DeleteExpenseCategory(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	if e = h.service.DeactivateExpenseCategory(r.Context(), id); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
func (h *OperationsHandler) ActivateExpenseCategory(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middlewareWriteBadID(w)
		return
	}
	if e = h.service.ActivateExpenseCategory(r.Context(), id); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "active": true})
}
func (h *OperationsHandler) ListExpenseCategories(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.ListExpenseCategories(r.Context())
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"categories": v})
}
func (h *OperationsHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	var e models.Expense
	if !decodeJSON(w, r, &e) {
		return
	}
	v, err := h.service.CreateExpense(r.Context(), currentUserID(r), e)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, 201, v)
}
func (h *OperationsHandler) CreateOwnerExpense(w http.ResponseWriter, r *http.Request) {
	var e models.InventoryOwnerExpense
	if !decodeJSON(w, r, &e) {
		return
	}
	v, err := h.service.CreateOwnerExpense(r.Context(), currentUserID(r), e)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, 201, v)
}
func ParseID(r *http.Request) int64 { id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64); return id }

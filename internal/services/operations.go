package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"supermarket/internal/models"
	"supermarket/internal/repository"
)

type OperationsService struct {
	repo  *repository.OperationsRepository
	audit *repository.AuditRepository
}

func NewOperationsService(r *repository.OperationsRepository, audits ...*repository.AuditRepository) *OperationsService {
	var audit *repository.AuditRepository
	if len(audits) > 0 {
		audit = audits[0]
	}
	return &OperationsService{repo: r, audit: audit}
}
func (s *OperationsService) GetSale(ctx context.Context, id int64) (models.Sale, error) {
	return s.repo.LoadSale(ctx, id)
}
func (s *OperationsService) ListSales(ctx context.Context, customerID int64, date, paymentStatus string) ([]models.Sale, error) {
	if paymentStatus != "" && paymentStatus != "paid" && paymentStatus != "unpaid" && paymentStatus != "partial" {
		return nil, fmt.Errorf("%w: payment_status", ErrValidation)
	}
	return s.repo.ListSales(ctx, customerID, date, paymentStatus)
}
func money(v float64) float64 { return float64(int64(v*100+0.5)) / 100 }

type SaleInput struct {
	RequestID      string            `json:"request_id"`
	CustomerID     *int64            `json:"customer_id"`
	Discount       float64           `json:"discount"`
	AmountReceived *float64          `json:"amount_received"`
	AmountPaid     *float64          `json:"amount_paid"`
	PaymentMethod  string            `json:"payment_method"`
	Items          []models.SaleItem `json:"items"`
}

func (s *OperationsService) CreateSale(ctx context.Context, userID int64, in SaleInput) (models.Sale, error) {
	if strings.TrimSpace(in.RequestID) == "" || len(in.Items) == 0 {
		return models.Sale{}, fmt.Errorf("%w: request_id and items required", ErrValidation)
	}
	if in.PaymentMethod != "cash" && in.PaymentMethod != "card" {
		return models.Sale{}, fmt.Errorf("%w: payment_method", ErrValidation)
	}
	if old, e := s.repo.ExistingSale(ctx, in.RequestID); e == nil {
		return old, nil
	} else if !errors.Is(e, repository.ErrNotFound) {
		return models.Sale{}, e
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return models.Sale{}, err
	}
	defer tx.Rollback()
	var subtotal float64
	for i := range in.Items {
		if in.Items[i].ProductID <= 0 || in.Items[i].Quantity <= 0 {
			return models.Sale{}, fmt.Errorf("%w: sale item", ErrValidation)
		}
		p, e := s.repo.SaleProductTx(ctx, tx, in.Items[i].ProductID)
		if e != nil {
			return models.Sale{}, e
		}
		unit, cost := float64(0), p.PurchasePrice
		if p.UnitType == "weight" {
			unit = p.PricePerKg
			cost = p.PurchasePricePerKg
		} else if p.UnitType == "piece" {
			unit = p.SalePrice
		} else if p.UnitType == "carton" {
			unit = p.PricePerPiece
			if p.PiecesPerCarton > 0 && int(in.Items[i].Quantity)%int(p.PiecesPerCarton) == 0 && in.Items[i].Quantity >= float64(p.PiecesPerCarton) {
				unit = p.PricePerCarton / float64(p.PiecesPerCarton)
			}
		}
		if promo, promoErr := s.repo.ActivePromotionTx(ctx, tx, in.Items[i].ProductID, time.Now().UTC(), p.Quantity); promoErr == nil {
			unit = PromotionPrice(promo, unit, in.Items[i].Quantity)
		}
		manualOverride := in.Items[i].PriceOverride != nil || in.Items[i].LineTotal > 0
		if manualOverride {
			in.Items[i].IsOverride = true
		}
		if in.Items[i].PriceOverride != nil {
			if *in.Items[i].PriceOverride < 0 {
				return models.Sale{}, fmt.Errorf("%w: negative price override", ErrValidation)
			}
			unit = *in.Items[i].PriceOverride
		}
		if unit < 0 || cost < 0 {
			return models.Sale{}, fmt.Errorf("%w: product pricing", ErrValidation)
		}
		in.Items[i].UnitPrice = money(unit)
		in.Items[i].CostPrice = money(cost)
		computedLineTotal := money(unit * in.Items[i].Quantity)
		if manualOverride && in.Items[i].LineTotal != 0 {
			if in.Items[i].LineTotal < 0 || (computedLineTotal > 0 && in.Items[i].LineTotal > computedLineTotal*10+0.000001) {
				return models.Sale{}, fmt.Errorf("%w: unreasonable line override", ErrValidation)
			}
			in.Items[i].LineTotal = money(in.Items[i].LineTotal)
			in.Items[i].UnitPrice = money(in.Items[i].LineTotal / in.Items[i].Quantity)
		} else {
			in.Items[i].LineTotal = computedLineTotal
		}
		subtotal += in.Items[i].LineTotal
	}
	total := money(subtotal - in.Discount)
	if in.Discount < 0 || total < 0 {
		return models.Sale{}, fmt.Errorf("%w: discount", ErrValidation)
	}
	received := total
	if in.AmountReceived != nil {
		received = *in.AmountReceived
		if received < 0 {
			return models.Sale{}, fmt.Errorf("%w: amount_received", ErrValidation)
		}
	} else if in.PaymentMethod == "cash" && in.AmountPaid != nil {
		received = *in.AmountPaid
	}
	paid := total
	if in.AmountPaid != nil {
		paid = *in.AmountPaid
	} else if in.PaymentMethod == "cash" {
		paid = received
		if paid > total {
			paid = total
		}
	}
	if paid < 0 || paid > total+0.000001 {
		return models.Sale{}, fmt.Errorf("%w: amount_paid", ErrValidation)
	}
	if in.PaymentMethod == "cash" {
		if in.CustomerID == nil && received+0.000001 < total {
			return models.Sale{}, fmt.Errorf("%w: walk-in cash must be fully paid", ErrValidation)
		}
		if in.AmountPaid != nil && paid > received+0.000001 {
			return models.Sale{}, fmt.Errorf("%w: amount_paid exceeds cash received", ErrValidation)
		}
		if in.AmountPaid == nil {
			paid = received
			if paid > total {
				paid = total
			}
		}
	} else {
		if in.AmountReceived != nil && received+0.000001 < total {
			return models.Sale{}, fmt.Errorf("%w: card amount_received must cover total", ErrValidation)
		}
		if in.AmountReceived != nil && received > total+0.000001 {
			return models.Sale{}, fmt.Errorf("%w: card amount_received cannot exceed total", ErrValidation)
		}
		if in.AmountPaid != nil && paid+0.000001 < total {
			return models.Sale{}, fmt.Errorf("%w: card payment must be complete", ErrValidation)
		}
		paid = total
		received = total
	}
	remaining := money(total - paid)
	if remaining > 0 && in.CustomerID == nil {
		return models.Sale{}, fmt.Errorf("%w: customer required for debt", ErrValidation)
	}
	if in.PaymentMethod == "card" && remaining > 0 {
		return models.Sale{}, fmt.Errorf("%w: card sales must be fully paid", ErrValidation)
	}
	var change float64
	if in.PaymentMethod == "cash" && received > total {
		change = money(received - total)
	}
	var shiftID int64
	if in.PaymentMethod == "cash" && received > 0 {
		shiftID, err = s.repo.AnyOpenShiftTx(ctx, tx)
		if err != nil {
			return models.Sale{}, fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}
	sale := models.Sale{RequestID: in.RequestID, CustomerID: in.CustomerID, UserID: userID, Subtotal: money(subtotal), Discount: money(in.Discount), Total: total, AmountPaid: money(paid), Remaining: remaining, PaymentMethod: in.PaymentMethod}
	sale.AmountReceived = &received
	sale.ChangeGiven = &change
	sale.InvoiceNumber = fmt.Sprintf("S-%d", time.Now().UnixNano())
	id, err := s.repo.InsertSaleTx(ctx, tx, sale)
	if err != nil {
		return models.Sale{}, err
	}
	sale.ID = id
	for _, item := range in.Items {
		if err := s.repo.InsertSaleItemTx(ctx, tx, id, item); err != nil {
			return models.Sale{}, err
		}
		if _, err := s.repo.ChangeStockTx(ctx, tx, item.ProductID, -item.Quantity, "sale", id, userID, ""); err != nil {
			return models.Sale{}, err
		}
	}
	if in.PaymentMethod == "cash" && received > 0 {
		ref := id
		if err := s.repo.AddCashMovementTx(ctx, tx, models.CashMovement{ShiftID: shiftID, Type: "sale", Direction: "in", Amount: received, ReferenceType: "sale", ReferenceID: &ref}); err != nil {
			return models.Sale{}, err
		}
		if change > 0 {
			if err := s.repo.AddCashMovementTx(ctx, tx, models.CashMovement{ShiftID: shiftID, Type: "sale", Direction: "out", Amount: change, ReferenceType: "sale", ReferenceID: &ref, Notes: "change"}); err != nil {
				return models.Sale{}, err
			}
		}
	}
	if s.audit != nil {
		record := id
		if err := s.audit.InsertTx(ctx, tx, models.AuditEntry{
			UserID: userID, Action: "sale_created", Module: "sales", RecordID: &record,
			Description: fmt.Sprintf("invoice %s total %.2f", sale.InvoiceNumber, sale.Total),
		}); err != nil {
			return models.Sale{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return models.Sale{}, err
	}
	return s.repo.LoadSale(ctx, id)
}

func (s *OperationsService) ListCustomers(ctx context.Context) ([]models.Customer, error) {
	return s.repo.ListCustomers(ctx)
}
func (s *OperationsService) UpdateCustomer(ctx context.Context, id int64, c models.Customer) error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("%w: customer name", ErrValidation)
	}
	c.Name = strings.TrimSpace(c.Name)
	return s.repo.UpdateCustomer(ctx, id, c)
}
func (s *OperationsService) DeactivateCustomer(ctx context.Context, id int64) error {
	return s.repo.DeactivateCustomer(ctx, id)
}
func (s *OperationsService) ActivateCustomer(ctx context.Context, id int64) error {
	return s.repo.ActivateCustomer(ctx, id)
}
func (s *OperationsService) CreateCustomer(ctx context.Context, c models.Customer) (models.Customer, error) {
	if strings.TrimSpace(c.Name) == "" {
		return c, fmt.Errorf("%w: customer name", ErrValidation)
	}
	c.Name = strings.TrimSpace(c.Name)
	return s.repo.CreateCustomer(ctx, c)
}
func (s *OperationsService) CustomerDebt(ctx context.Context, id int64) (models.Customer, []models.Sale, error) {
	return s.repo.GetCustomerDebt(ctx, id)
}
func (s *OperationsService) PayDebt(ctx context.Context, userID, customerID int64, saleID int64, amount float64, method string) (float64, error) {
	if amount <= 0 || (method != "cash" && method != "card") {
		return 0, fmt.Errorf("%w: payment", ErrValidation)
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var shift *int64
	if method == "cash" {
		id, e := s.repo.AnyOpenShiftTx(ctx, tx)
		if e != nil {
			return 0, fmt.Errorf("%w: %v", ErrValidation, e)
		}
		shift = &id
	}
	var applied float64
	if saleID > 0 {
		var owner int64
		var remaining float64
		if err := tx.QueryRowContext(ctx, `SELECT customer_id,remaining FROM sales WHERE id=?`, saleID).Scan(&owner, &remaining); err != nil {
			return 0, repository.ErrNotFound
		}
		if owner != customerID || amount > remaining+0.000001 {
			return 0, fmt.Errorf("%w: payment exceeds debt", ErrValidation)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE sales SET amount_paid=amount_paid+?,remaining=remaining-? WHERE id=?`, amount, amount, saleID); err != nil {
			return 0, err
		}
		if err := s.repo.InsertCustomerPaymentTx(ctx, tx, models.DebtPayment{SaleID: saleID, CustomerID: customerID, Amount: amount, Method: method}, shift); err != nil {
			return 0, err
		}
		applied = amount
	} else {
		outstanding, e := s.repo.CustomerOutstandingTx(ctx, tx, customerID)
		if e != nil {
			return 0, e
		}
		if amount > outstanding+0.000001 {
			return 0, fmt.Errorf("%w: payment exceeds outstanding balance", ErrValidation)
		}
		applied, err = s.repo.ApplyDebtPaymentTx(ctx, tx, customerID, amount, method, shift)
		if err != nil {
			return 0, err
		}
	}
	if method == "cash" && applied > 0 {
		if err := s.repo.AddCashMovementTx(ctx, tx, models.CashMovement{ShiftID: *shift, Type: "customer_payment", Direction: "in", Amount: applied, ReferenceType: "customer", ReferenceID: &customerID}); err != nil {
			return 0, err
		}
	}
	if s.audit != nil {
		record := customerID
		if err := s.audit.InsertTx(ctx, tx, models.AuditEntry{UserID: userID, Action: "customer_payment", Module: "customer_payments", RecordID: &record, Description: fmt.Sprintf("%.2f", applied)}); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return applied, nil
}

type ReturnInput struct {
	SaleID        int64                    `json:"sale_id"`
	CustomerID    *int64                   `json:"customer_id"`
	RefundType    string                   `json:"refund_type"`
	Notes         string                   `json:"notes"`
	Items         []models.SalesReturnItem `json:"items"`
	ExchangeItems []models.SaleItem        `json:"exchange_items"`
}

func (s *OperationsService) ReturnSale(ctx context.Context, userID, saleID int64, in ReturnInput) (models.SalesReturn, error) {
	in.SaleID = saleID
	return s.CreateSalesReturn(ctx, userID, in)
}

func (s *OperationsService) CreateSalesReturn(ctx context.Context, userID int64, in ReturnInput) (models.SalesReturn, error) {
	if in.SaleID <= 0 || len(in.Items) == 0 || (in.RefundType != "debt_removed" && in.RefundType != "cash_refund" && in.RefundType != "exchange") {
		return models.SalesReturn{}, ErrInvalidReturnState
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return models.SalesReturn{}, err
	}
	defer tx.Rollback()
	var saleCustomer sqlNullInt64
	var saleTotal, saleRemaining float64
	var salePaymentMethod string
	if err := tx.QueryRowContext(ctx, `SELECT customer_id,total,remaining,payment_method FROM sales WHERE id=?`, in.SaleID).Scan(&saleCustomer, &saleTotal, &saleRemaining, &salePaymentMethod); err != nil {
		return models.SalesReturn{}, repository.ErrNotFound
	}
	if in.CustomerID == nil && saleCustomer.Valid {
		in.CustomerID = &saleCustomer.Int64
	}
	var total float64
	for i := range in.Items {
		q, price, e := s.repo.SaleLineTx(ctx, tx, in.SaleID, in.Items[i].ProductID)
		if e != nil {
			return models.SalesReturn{}, e
		}
		returned, e := s.repo.ReturnedSaleQtyTx(ctx, tx, in.SaleID, in.Items[i].ProductID)
		if e != nil {
			return models.SalesReturn{}, e
		}
		if in.Items[i].Quantity <= 0 {
			return models.SalesReturn{}, fmt.Errorf("%w: quantity must be positive", ErrInvalidReturnState)
		}
		if q-returned <= 0.000001 {
			return models.SalesReturn{}, ErrAlreadyReturned
		}
		if in.Items[i].Quantity > q-returned+0.000001 {
			return models.SalesReturn{}, ErrReturnQuantityExceeded
		}
		in.Items[i].UnitPrice = price
		in.Items[i].LineTotal = money(price * in.Items[i].Quantity)
		total += in.Items[i].LineTotal
	}
	total = money(total)
	if in.RefundType == "debt_removed" && saleRemaining+0.000001 < total {
		return models.SalesReturn{}, fmt.Errorf("%w: return exceeds debt", ErrInvalidReturnState)
	}
	if in.RefundType == "cash_refund" && saleRemaining > 0.000001 {
		return models.SalesReturn{}, fmt.Errorf("%w: sale is not fully paid", ErrInvalidReturnState)
	}
	if (in.RefundType == "cash_refund" || in.RefundType == "exchange") && (salePaymentMethod != "cash" || saleRemaining > 0.000001) {
		return models.SalesReturn{}, fmt.Errorf("%w: cash returns require a fully paid cash sale", ErrInvalidReturnState)
	}
	var shiftID int64
	if in.RefundType == "cash_refund" {
		shiftID, err = s.repo.AnyOpenShiftTx(ctx, tx)
		if err != nil {
			return models.SalesReturn{}, fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}
	v := models.SalesReturn{SaleID: in.SaleID, CustomerID: in.CustomerID, UserID: userID, RefundType: in.RefundType, TotalValue: total, Notes: in.Notes, Items: in.Items}
	var exchangeSaleID *int64
	var netDifference float64
	if in.RefundType == "exchange" {
		if len(in.ExchangeItems) == 0 {
			return v, fmt.Errorf("%w: exchange items required", ErrInvalidReturnState)
		}
		ex, exchangeTotal, ee := s.createExchangeSaleTx(ctx, tx, userID, in.CustomerID, in.ExchangeItems)
		if ee != nil {
			return v, ee
		}
		exchangeSaleID = &ex
		netDifference = money(exchangeTotal - total)
		if netDifference != 0 {
			exchangeShift, se := s.repo.AnyOpenShiftTx(ctx, tx)
			if se != nil {
				return v, fmt.Errorf("%w: %v", ErrValidation, se)
			}
			direction := "in"
			amount := netDifference
			if amount < 0 {
				direction = "out"
				amount = -amount
			}
			ref := ex
			if se = s.repo.AddCashMovementTx(ctx, tx, models.CashMovement{ShiftID: exchangeShift, Type: "sales_return_refund", Direction: direction, Amount: amount, ReferenceType: "exchange", ReferenceID: &ref}); se != nil {
				return v, se
			}
		}
	}
	v.NetDifference = netDifference
	v.LinkedSaleID = exchangeSaleID
	id, err := s.repo.InsertSalesReturnTx(ctx, tx, v)
	if err != nil {
		return v, err
	}
	if s.audit != nil {
		record := id
		if err := s.audit.InsertTx(ctx, tx, models.AuditEntry{UserID: userID, Action: "sales_return", Module: "sales_returns", RecordID: &record, Description: in.RefundType}); err != nil {
			return v, err
		}
	}
	v.ID = id
	for _, item := range in.Items {
		if err := s.repo.InsertSalesReturnItemTx(ctx, tx, id, item); err != nil {
			return v, err
		}
		if _, err := s.repo.ChangeStockTx(ctx, tx, item.ProductID, item.Quantity, "return_sale", id, userID, ""); err != nil {
			return v, err
		}
	}
	if in.RefundType == "debt_removed" {
		_, err = tx.ExecContext(ctx, `UPDATE sales SET total=total-?,remaining=remaining-? WHERE id=?`, total, total, in.SaleID)
	} else if in.RefundType == "cash_refund" {
		ref := id
		err = s.repo.AddCashMovementTx(ctx, tx, models.CashMovement{ShiftID: shiftID, Type: "sales_return_refund", Direction: "out", Amount: total, ReferenceType: "sales_return", ReferenceID: &ref})
	} else if in.RefundType == "exchange" && saleRemaining > 0 {
		_, err = tx.ExecContext(ctx, `UPDATE sales SET total=total-?,remaining=MAX(0,remaining-?) WHERE id=?`, total, total, in.SaleID)
	}
	if err != nil {
		return v, err
	}
	if err := tx.Commit(); err != nil {
		return v, err
	}
	v.CreatedAt = time.Now().UTC()
	return v, nil
}

type sqlNullInt64 struct {
	Int64 int64
	Valid bool
}

func (n *sqlNullInt64) Scan(v any) error {
	switch x := v.(type) {
	case nil:
		n.Valid = false
		return nil
	case int64:
		n.Int64 = x
		n.Valid = true
		return nil
	case int:
		n.Int64 = int64(x)
		n.Valid = true
		return nil
	}
	return fmt.Errorf("invalid nullable integer")
}
func (s *OperationsService) createExchangeSaleTx(ctx context.Context, tx *sql.Tx, userID int64, customerID *int64, items []models.SaleItem) (int64, float64, error) {
	var total float64
	for i := range items {
		p, e := s.repo.SaleProductTx(ctx, tx, items[i].ProductID)
		if e != nil {
			return 0, 0, e
		}
		unit, cost := p.SalePrice, p.PurchasePrice
		if p.UnitType == "weight" {
			unit, cost = p.PricePerKg, p.PurchasePricePerKg
		}
		if p.UnitType == "carton" {
			unit = p.PricePerPiece
			if p.PiecesPerCarton > 0 && int(items[i].Quantity)%int(p.PiecesPerCarton) == 0 {
				unit = p.PricePerCarton / float64(p.PiecesPerCarton)
			}
		}
		if promo, promoErr := s.repo.ActivePromotionTx(ctx, tx, items[i].ProductID, time.Now().UTC(), p.Quantity); promoErr == nil {
			unit = PromotionPrice(promo, unit, items[i].Quantity)
		}
		manualOverride := items[i].PriceOverride != nil || items[i].LineTotal > 0
		if items[i].PriceOverride != nil {
			if *items[i].PriceOverride < 0 {
				return 0, 0, fmt.Errorf("%w: negative price override", ErrValidation)
			}
			unit = *items[i].PriceOverride
		}
		items[i].UnitPrice = unit
		items[i].CostPrice = cost
		if manualOverride && items[i].LineTotal > 0 {
			items[i].LineTotal = money(items[i].LineTotal)
			unit = items[i].LineTotal / items[i].Quantity
			items[i].UnitPrice = money(unit)
		} else {
			items[i].LineTotal = money(unit * items[i].Quantity)
		}
		total += items[i].LineTotal
	}
	recv := money(total)
	sale := models.Sale{InvoiceNumber: fmt.Sprintf("S-%d", time.Now().UnixNano()), RequestID: fmt.Sprintf("exchange-%d", time.Now().UnixNano()), CustomerID: customerID, UserID: userID, Subtotal: money(total), Total: money(total), AmountPaid: money(total), PaymentMethod: "cash", AmountReceived: &recv}
	id, e := s.repo.InsertSaleTx(ctx, tx, sale)
	if e != nil {
		return 0, 0, e
	}
	for _, item := range items {
		if e = s.repo.InsertSaleItemTx(ctx, tx, id, item); e != nil {
			return 0, 0, e
		}
		if _, e = s.repo.ChangeStockTx(ctx, tx, item.ProductID, -item.Quantity, "sale", id, userID, "exchange"); e != nil {
			return 0, 0, e
		}
	}
	return id, money(total), nil
}

func (s *OperationsService) OpenShift(ctx context.Context, userID int64, balance float64, notes string) (models.CashShift, error) {
	if balance < 0 {
		return models.CashShift{}, fmt.Errorf("%w: opening balance", ErrValidation)
	}
	tx, e := s.repo.Begin(ctx)
	if e != nil {
		return models.CashShift{}, e
	}
	defer tx.Rollback()
	if _, e = s.repo.OpenShiftTx(ctx, tx, userID); e != nil {
		return models.CashShift{}, fmt.Errorf("%w: %v", ErrValidation, e)
	}
	id, e := s.repo.CreateShiftTx(ctx, tx, models.CashShift{UserID: userID, OpeningBalance: balance, OpeningNotes: notes})
	if e != nil {
		return models.CashShift{}, e
	}
	if s.audit != nil {
		record := id
		if e = s.audit.InsertTx(ctx, tx, models.AuditEntry{UserID: userID, Action: "shift_opened", Module: "cash_shifts", RecordID: &record}); e != nil {
			return models.CashShift{}, e
		}
	}
	if e = tx.Commit(); e != nil {
		return models.CashShift{}, e
	}
	return s.repo.GetShift(ctx, id)
}
func (s *OperationsService) CloseShift(ctx context.Context, userID, id int64, actual float64, notes string) (models.CashShift, error) {
	if actual < 0 {
		return models.CashShift{}, fmt.Errorf("%w: closing balance", ErrValidation)
	}
	tx, e := s.repo.Begin(ctx)
	if e != nil {
		return models.CashShift{}, e
	}
	defer tx.Rollback()
	var owner int64
	if e = tx.QueryRowContext(ctx, `SELECT user_id FROM cash_shifts WHERE id=? AND closed_at IS NULL`, id).Scan(&owner); e != nil {
		return models.CashShift{}, repository.ErrNotFound
	}
	if owner != userID {
		return models.CashShift{}, fmt.Errorf("shift belongs to another user")
	}
	if _, e = s.repo.CloseShiftTx(ctx, tx, id, actual, notes); e != nil {
		return models.CashShift{}, e
	}
	if s.audit != nil {
		record := id
		if e = s.audit.InsertTx(ctx, tx, models.AuditEntry{UserID: userID, Action: "shift_closed", Module: "cash_shifts", RecordID: &record}); e != nil {
			return models.CashShift{}, e
		}
	}
	if e = tx.Commit(); e != nil {
		return models.CashShift{}, e
	}
	return s.repo.GetShift(ctx, id)
}
func (s *OperationsService) ListShifts(ctx context.Context) ([]models.CashShift, error) {
	return s.repo.ListShifts(ctx)
}
func (s *OperationsService) CurrentShift(ctx context.Context) (models.CashShift, error) {
	return s.repo.GetCurrentShift(ctx)
}
func (s *OperationsService) ListCashMovements(ctx context.Context, shiftID int64) ([]models.CashMovement, error) {
	return s.repo.ListCashMovements(ctx, shiftID)
}

type ManualCashMovementInput struct {
	Direction string  `json:"direction"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Notes     string  `json:"notes"`
}

func (s *OperationsService) CreateManualCashMovement(ctx context.Context, userID int64, in ManualCashMovementInput) (models.CashMovement, error) {
	if in.Direction == "" {
		switch in.Type {
		case "manual_in":
			in.Direction = "in"
		case "manual_out":
			in.Direction = "out"
		}
	}
	if (in.Direction != "in" && in.Direction != "out") || in.Amount <= 0 {
		return models.CashMovement{}, fmt.Errorf("%w: manual cash movement", ErrValidation)
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return models.CashMovement{}, err
	}
	defer tx.Rollback()
	shiftID, err := s.repo.AnyOpenShiftTx(ctx, tx)
	if err != nil {
		return models.CashMovement{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	typ := "manual_" + in.Direction
	m := models.CashMovement{ShiftID: shiftID, Type: typ, Direction: in.Direction, Amount: money(in.Amount), Notes: in.Notes}
	id, err := s.repo.InsertManualCashMovementTx(ctx, tx, m)
	if err != nil {
		return models.CashMovement{}, err
	}
	if s.audit != nil {
		record := id
		if err = s.audit.InsertTx(ctx, tx, models.AuditEntry{UserID: userID, Action: "manual_cash_movement", Module: "cash_movements", RecordID: &record, Description: in.Direction}); err != nil {
			return models.CashMovement{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return models.CashMovement{}, err
	}
	m.ID = id
	m.CreatedAt = time.Now().UTC()
	return m, nil
}
func (s *OperationsService) CreateExpenseCategory(ctx context.Context, c models.ExpenseCategory) (models.ExpenseCategory, error) {
	if strings.TrimSpace(c.Name) == "" {
		return c, fmt.Errorf("%w: category name", ErrValidation)
	}
	c.Name = strings.TrimSpace(c.Name)
	return s.repo.InsertExpenseCategory(ctx, c)
}
func (s *OperationsService) UpdateExpenseCategory(ctx context.Context, id int64, c models.ExpenseCategory) error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("%w: category name", ErrValidation)
	}
	c.Name = strings.TrimSpace(c.Name)
	return s.repo.UpdateExpenseCategory(ctx, id, c)
}
func (s *OperationsService) DeactivateExpenseCategory(ctx context.Context, id int64) error {
	return s.repo.DeactivateExpenseCategory(ctx, id)
}
func (s *OperationsService) ActivateExpenseCategory(ctx context.Context, id int64) error {
	return s.repo.ActivateExpenseCategory(ctx, id)
}
func (s *OperationsService) ListExpenseCategories(ctx context.Context) ([]models.ExpenseCategory, error) {
	return s.repo.ListExpenseCategories(ctx)
}
func (s *OperationsService) CreateExpense(ctx context.Context, userID int64, e models.Expense) (models.Expense, error) {
	if e.Amount <= 0 || (e.PaymentMethod != "cash" && e.PaymentMethod != "card" && e.PaymentMethod != "bank") || e.ExpenseCategoryID <= 0 {
		return e, fmt.Errorf("%w: expense", ErrValidation)
	}
	e.UserID = userID
	if e.ExpenseDate == "" {
		e.ExpenseDate = time.Now().Format("2006-01-02")
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return e, err
	}
	defer tx.Rollback()
	if e.PaymentMethod == "cash" {
		id, ee := s.repo.AnyOpenShiftTx(ctx, tx)
		if ee != nil {
			return e, fmt.Errorf("%w: %v", ErrValidation, ee)
		}
		e.ShiftID = &id
	}
	id, err := s.repo.InsertExpenseTx(ctx, tx, e)
	if err != nil {
		return e, err
	}
	if e.PaymentMethod == "cash" {
		ref := id
		if err = s.repo.AddCashMovementTx(ctx, tx, models.CashMovement{ShiftID: *e.ShiftID, Type: "expense", Direction: "out", Amount: e.Amount, ReferenceType: "expense", ReferenceID: &ref}); err != nil {
			return e, err
		}
	}
	if s.audit != nil {
		record := id
		if err = s.audit.InsertTx(ctx, tx, models.AuditEntry{UserID: userID, Action: "expense_created", Module: "expenses", RecordID: &record}); err != nil {
			return e, err
		}
	}
	if err = tx.Commit(); err != nil {
		return e, err
	}
	e.ID = id
	return e, nil
}
func (s *OperationsService) CreateOwnerExpense(ctx context.Context, userID int64, e models.InventoryOwnerExpense) (models.InventoryOwnerExpense, error) {
	if e.ProductID <= 0 || e.Quantity <= 0 {
		return e, fmt.Errorf("%w: owner expense", ErrValidation)
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return e, err
	}
	defer tx.Rollback()
	p, err := s.repo.SaleProductTx(ctx, tx, e.ProductID)
	if err != nil {
		return e, err
	}
	e.UserID = userID
	if p.UnitType == "weight" {
		e.UnitCost = p.PurchasePricePerKg
	} else {
		e.UnitCost = p.PurchasePrice
	}
	e.TotalValue = money(e.Quantity * e.UnitCost)
	id, err := s.repo.InsertOwnerExpenseTx(ctx, tx, e)
	if err != nil {
		return e, err
	}
	if _, err = s.repo.ChangeStockTx(ctx, tx, e.ProductID, -e.Quantity, "owner_expense", id, userID, e.Notes); err != nil {
		return e, err
	}
	if s.audit != nil {
		record := id
		if err = s.audit.InsertTx(ctx, tx, models.AuditEntry{UserID: userID, Action: "inventory_owner_expense", Module: "inventory_owner_expenses", RecordID: &record}); err != nil {
			return e, err
		}
	}
	if err = tx.Commit(); err != nil {
		return e, err
	}
	e.ID = id
	e.CreatedAt = time.Now().UTC()
	return e, nil
}

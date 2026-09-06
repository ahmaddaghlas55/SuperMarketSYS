package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"supermarket/internal/models"
)

type OperationsRepository struct{ db *sql.DB }

func NewOperationsRepository(db *sql.DB) *OperationsRepository { return &OperationsRepository{db: db} }
func (r *OperationsRepository) Begin(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

type SaleProduct struct {
	UnitType                                                                                string
	Quantity                                                                                float64
	SalePrice, PricePerKg, PricePerPiece, PricePerCarton, PurchasePrice, PurchasePricePerKg float64
	PiecesPerCarton                                                                         int64
}

func (r *OperationsRepository) ActivePromotionTx(ctx context.Context, tx *sql.Tx, productID int64, now time.Time, stock float64) (models.Promotion, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,product_id,promo_type,value,bundle_quantity,starts_at,ends_at,until_stock_zero,active,created_at
		FROM promotions WHERE product_id=? AND active=1 ORDER BY created_at DESC,id DESC`, productID)
	if err != nil {
		return models.Promotion{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var p models.Promotion
		var starts, ends, created sql.NullString
		var bundle sql.NullInt64
		var active, until int
		if err := rows.Scan(&p.ID, &p.ProductID, &p.PromoType, &p.Value, &bundle, &starts, &ends, &until, &active, &created); err != nil {
			return p, err
		}
		if bundle.Valid {
			p.BundleQuantity = &bundle.Int64
		}
		p.StartsAt = parsePromotionTime(starts)
		p.EndsAt = parsePromotionTime(ends)
		p.Active, p.UntilStockZero = active == 1, until == 1
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created.String)
		if p.StartsAt != nil && now.Before(*p.StartsAt) || p.EndsAt != nil && now.After(*p.EndsAt) || p.UntilStockZero && stock <= 0 {
			continue
		}
		return p, nil
	}
	if err := rows.Err(); err != nil {
		return models.Promotion{}, err
	}
	return models.Promotion{}, ErrNotFound
}

func parsePromotionTime(v sql.NullString) *time.Time {
	if !v.Valid || v.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, v.String)
	if err != nil {
		t, _ = time.Parse("2006-01-02 15:04:05", v.String)
	}
	if t.IsZero() {
		return nil
	}
	return &t
}

func (r *OperationsRepository) SaleProductTx(ctx context.Context, tx *sql.Tx, id int64) (SaleProduct, error) {
	var p SaleProduct
	err := tx.QueryRowContext(ctx, `SELECT unit_type,quantity,COALESCE(sale_price,0),COALESCE(price_per_kg,0),COALESCE(price_per_piece,0),COALESCE(price_per_carton,0),COALESCE(purchase_price,0),COALESCE(purchase_price_per_kg,0),COALESCE(pieces_per_carton,0) FROM products WHERE id=? AND active=1`, id).Scan(&p.UnitType, &p.Quantity, &p.SalePrice, &p.PricePerKg, &p.PricePerPiece, &p.PricePerCarton, &p.PurchasePrice, &p.PurchasePricePerKg, &p.PiecesPerCarton)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}
func (r *OperationsRepository) ExistingSale(ctx context.Context, requestID string) (models.Sale, error) {
	return r.getSale(r.db.QueryRowContext(ctx, `SELECT id,invoice_number,request_id,customer_id,user_id,subtotal,discount,total,amount_paid,remaining,amount_received,change_given,payment_method,created_at FROM sales WHERE request_id=?`, requestID))
}
func (r *OperationsRepository) getSale(row interface{ Scan(...any) error }) (models.Sale, error) {
	var s models.Sale
	var cid sql.NullInt64
	var recv, change sql.NullFloat64
	var created string
	err := row.Scan(&s.ID, &s.InvoiceNumber, &s.RequestID, &cid, &s.UserID, &s.Subtotal, &s.Discount, &s.Total, &s.AmountPaid, &s.Remaining, &recv, &change, &s.PaymentMethod, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return s, ErrNotFound
	}
	if err != nil {
		return s, err
	}
	if cid.Valid {
		s.CustomerID = &cid.Int64
	}
	if recv.Valid {
		s.AmountReceived = &recv.Float64
	}
	if change.Valid {
		s.ChangeGiven = &change.Float64
	}
	s.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
	s.Aging = s.Remaining > 0.000001 && !s.CreatedAt.IsZero() && !time.Now().UTC().Before(s.CreatedAt.AddDate(0, 1, 0))
	return s, nil
}
func (r *OperationsRepository) LoadSale(ctx context.Context, id int64) (models.Sale, error) {
	s, err := r.getSale(r.db.QueryRowContext(ctx, `SELECT id,invoice_number,request_id,customer_id,user_id,subtotal,discount,total,amount_paid,remaining,amount_received,change_given,payment_method,created_at FROM sales WHERE id=?`, id))
	if err != nil {
		return s, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,product_id,quantity,unit_price,cost_price,is_override,line_total FROM sale_items WHERE sale_id=?`, id)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var i models.SaleItem
		var over int
		if err := rows.Scan(&i.ID, &i.ProductID, &i.Quantity, &i.UnitPrice, &i.CostPrice, &over, &i.LineTotal); err != nil {
			return s, err
		}
		i.IsOverride = over == 1
		s.Items = append(s.Items, i)
	}
	return s, rows.Err()
}
func (r *OperationsRepository) ListSales(ctx context.Context, customerID int64, date, paymentStatus string) ([]models.Sale, error) {
	q := `SELECT id,invoice_number,request_id,customer_id,user_id,subtotal,discount,total,amount_paid,remaining,amount_received,change_given,payment_method,created_at FROM sales WHERE 1=1`
	args := []any{}
	if customerID > 0 {
		q += ` AND customer_id=?`
		args = append(args, customerID)
	}
	if date != "" {
		q += ` AND date(created_at)=?`
		args = append(args, date)
	}
	switch paymentStatus {
	case "paid":
		q += ` AND remaining<=0.000001`
	case "unpaid":
		q += ` AND amount_paid<=0.000001 AND total>0`
	case "partial":
		q += ` AND amount_paid>0.000001 AND remaining>0.000001`
	case "":
	default:
		return nil, fmt.Errorf("invalid payment_status")
	}
	q += ` ORDER BY created_at DESC,id DESC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Sale
	for rows.Next() {
		s, err := r.getSale(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (r *OperationsRepository) InsertSaleTx(ctx context.Context, tx *sql.Tx, s models.Sale) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO sales(invoice_number,request_id,customer_id,user_id,subtotal,discount,total,amount_paid,remaining,amount_received,change_given,payment_method) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, s.InvoiceNumber, s.RequestID, s.CustomerID, s.UserID, s.Subtotal, s.Discount, s.Total, s.AmountPaid, s.Remaining, s.AmountReceived, s.ChangeGiven, s.PaymentMethod)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func (r *OperationsRepository) InsertSaleItemTx(ctx context.Context, tx *sql.Tx, saleID int64, i models.SaleItem) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO sale_items(sale_id,product_id,quantity,unit_price,cost_price,is_override,line_total) VALUES(?,?,?,?,?,?,?)`, saleID, i.ProductID, i.Quantity, i.UnitPrice, i.CostPrice, boolInt(i.IsOverride), i.LineTotal)
	return err
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func (r *OperationsRepository) ChangeStockTx(ctx context.Context, tx *sql.Tx, productID int64, delta float64, typ string, refID, userID int64, notes string) (float64, error) {
	var current float64
	if err := tx.QueryRowContext(ctx, `SELECT quantity FROM products WHERE id=? AND active=1`, productID).Scan(&current); err != nil {
		return 0, err
	}
	next := current + delta
	if next < -0.000001 {
		return 0, fmt.Errorf("insufficient stock")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE products SET quantity=? WHERE id=?`, next, productID); err != nil {
		return 0, err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO stock_movements(product_id,type,quantity_change,balance_after,reference_type,reference_id,user_id,notes) VALUES(?,?,?,?,?,?,?,?)`, productID, typ, delta, next, typ, refID, userID, notes)
	return next, err
}
func (r *OperationsRepository) OpenShiftTx(ctx context.Context, tx *sql.Tx, userID int64) (models.CashShift, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM cash_shifts WHERE closed_at IS NULL LIMIT 1`).Scan(&id)
	if err == nil {
		return models.CashShift{}, fmt.Errorf("open shift already exists")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return models.CashShift{}, err
	}
	return models.CashShift{}, nil
}
func (r *OperationsRepository) ShiftIDTx(ctx context.Context, tx *sql.Tx, userID int64) (int64, error) {
	return r.AnyOpenShiftTx(ctx, tx)
}
func (r *OperationsRepository) AnyOpenShiftTx(ctx context.Context, tx *sql.Tx) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM cash_shifts WHERE closed_at IS NULL ORDER BY opened_at,id LIMIT 1`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("open shift required")
	}
	return id, err
}
func (r *OperationsRepository) AddCashMovementTx(ctx context.Context, tx *sql.Tx, m models.CashMovement) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO cash_movements(shift_id,type,direction,amount,reference_type,reference_id,notes) VALUES(?,?,?,?,?,?,?)`, m.ShiftID, m.Type, m.Direction, m.Amount, m.ReferenceType, m.ReferenceID, m.Notes)
	return err
}
func (r *OperationsRepository) InsertManualCashMovementTx(ctx context.Context, tx *sql.Tx, m models.CashMovement) (int64, error) {
	if m.Type != "manual_in" && m.Type != "manual_out" {
		return 0, fmt.Errorf("invalid manual movement type")
	}
	if m.Direction != "in" && m.Direction != "out" {
		return 0, fmt.Errorf("invalid cash movement direction")
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO cash_movements(shift_id,type,direction,amount,notes) VALUES(?,?,?,?,?)`, m.ShiftID, m.Type, m.Direction, m.Amount, m.Notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func (r *OperationsRepository) CreateShiftTx(ctx context.Context, tx *sql.Tx, s models.CashShift) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO cash_shifts(user_id,opening_balance,opening_notes) VALUES(?,?,?)`, s.UserID, s.OpeningBalance, s.OpeningNotes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func (r *OperationsRepository) GetShift(ctx context.Context, id int64) (models.CashShift, error) {
	var s models.CashShift
	var opened, closed sql.NullString
	var openingNotes, closingNotes sql.NullString
	var exp, actual, diff sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `SELECT id,user_id,opening_balance,opening_notes,opened_at,expected_closing,actual_closing,difference,closing_notes,closed_at FROM cash_shifts WHERE id=?`, id).Scan(&s.ID, &s.UserID, &s.OpeningBalance, &openingNotes, &opened, &exp, &actual, &diff, &closingNotes, &closed)
	if errors.Is(err, sql.ErrNoRows) {
		return s, ErrNotFound
	}
	if err != nil {
		return s, err
	}
	if err := r.populateCashTotals(ctx, &s); err != nil {
		return s, err
	}
	s.OpeningNotes = openingNotes.String
	s.ClosingNotes = closingNotes.String
	s.OpenedAt, _ = time.Parse("2006-01-02 15:04:05", opened.String)
	if exp.Valid {
		s.ExpectedClosing = &exp.Float64
	}
	if actual.Valid {
		s.ActualClosing = &actual.Float64
	}
	if diff.Valid {
		s.Difference = &diff.Float64
	}
	if closed.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", closed.String)
		s.ClosedAt = &t
	}
	return s, nil
}

func (r *OperationsRepository) populateCashTotals(ctx context.Context, s *models.CashShift) error {
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN direction='in' THEN amount ELSE 0 END),0), COALESCE(SUM(CASE WHEN direction='out' THEN amount ELSE 0 END),0) FROM cash_movements WHERE shift_id=?`, s.ID).Scan(&s.CashIn, &s.CashOut); err != nil {
		return err
	}
	s.RunningBalance = s.OpeningBalance + s.CashIn - s.CashOut
	return nil
}
func (r *OperationsRepository) GetCurrentShift(ctx context.Context) (models.CashShift, error) {
	var id int64
	if err := r.db.QueryRowContext(ctx, `SELECT id FROM cash_shifts WHERE closed_at IS NULL ORDER BY opened_at,id LIMIT 1`).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.CashShift{}, ErrNotFound
		}
		return models.CashShift{}, err
	}
	return r.GetShift(ctx, id)
}
func (r *OperationsRepository) CloseShiftTx(ctx context.Context, tx *sql.Tx, id int64, actual float64, notes string) (float64, error) {
	var opening float64
	if err := tx.QueryRowContext(ctx, `SELECT opening_balance FROM cash_shifts WHERE id=? AND closed_at IS NULL`, id).Scan(&opening); err != nil {
		return 0, err
	}
	var in, out float64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN direction='in' THEN amount ELSE 0 END),0),COALESCE(SUM(CASE WHEN direction='out' THEN amount ELSE 0 END),0) FROM cash_movements WHERE shift_id=?`, id).Scan(&in, &out); err != nil {
		return 0, err
	}
	expected := opening + in - out
	diff := actual - expected
	_, err := tx.ExecContext(ctx, `UPDATE cash_shifts SET expected_closing=?,actual_closing=?,difference=?,closing_notes=?,closed_at=datetime('now') WHERE id=? AND closed_at IS NULL`, expected, actual, diff, notes, id)
	return diff, err
}
func (r *OperationsRepository) ListShifts(ctx context.Context) ([]models.CashShift, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,user_id,opening_balance,opening_notes,opened_at,expected_closing,actual_closing,difference,closing_notes,closed_at FROM cash_shifts ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.CashShift
	for rows.Next() {
		var s models.CashShift
		var opened, closed sql.NullString
		var openingNotes, closingNotes sql.NullString
		var exp, actual, diff sql.NullFloat64
		if err := rows.Scan(&s.ID, &s.UserID, &s.OpeningBalance, &openingNotes, &opened, &exp, &actual, &diff, &closingNotes, &closed); err != nil {
			return nil, err
		}
		s.OpeningNotes = openingNotes.String
		s.ClosingNotes = closingNotes.String
		s.OpenedAt, _ = time.Parse("2006-01-02 15:04:05", opened.String)
		if exp.Valid {
			s.ExpectedClosing = &exp.Float64
		}
		if actual.Valid {
			s.ActualClosing = &actual.Float64
		}
		if diff.Valid {
			s.Difference = &diff.Float64
		}
		if closed.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", closed.String)
			s.ClosedAt = &t
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *OperationsRepository) ListCashMovements(ctx context.Context, shiftID int64) ([]models.CashMovement, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,shift_id,type,direction,amount,COALESCE(reference_type,''),reference_id,COALESCE(notes,''),created_at FROM cash_movements WHERE shift_id=? ORDER BY created_at,id`, shiftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.CashMovement
	for rows.Next() {
		var m models.CashMovement
		var refType, notes, created string
		var refID sql.NullInt64
		if err := rows.Scan(&m.ID, &m.ShiftID, &m.Type, &m.Direction, &m.Amount, &refType, &refID, &notes, &created); err != nil {
			return nil, err
		}
		m.ReferenceType, m.Notes = refType, notes
		if refID.Valid {
			m.ReferenceID = &refID.Int64
		}
		m.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *OperationsRepository) ListCustomers(ctx context.Context) ([]models.Customer, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT c.id,c.name,COALESCE(c.phone,''),c.active,COALESCE(SUM(s.remaining),0) FROM customers c LEFT JOIN sales s ON s.customer_id=c.id AND s.remaining>0 WHERE c.active=1 GROUP BY c.id ORDER BY c.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Customer
	for rows.Next() {
		var c models.Customer
		var a int
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &a, &c.Outstanding); err != nil {
			return nil, err
		}
		c.Active = a == 1
		out = append(out, c)
	}
	return out, rows.Err()
}
func (r *OperationsRepository) CreateCustomer(ctx context.Context, c models.Customer) (models.Customer, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO customers(name,phone) VALUES(?,?)`, c.Name, c.Phone)
	if err != nil {
		return c, err
	}
	c.ID, _ = res.LastInsertId()
	c.Active = true
	return c, nil
}
func (r *OperationsRepository) UpdateCustomer(ctx context.Context, id int64, c models.Customer) error {
	res, err := r.db.ExecContext(ctx, `UPDATE customers SET name=?,phone=? WHERE id=? AND active=1`, c.Name, c.Phone, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *OperationsRepository) DeactivateCustomer(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE customers SET active=0 WHERE id=? AND active=1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *OperationsRepository) ActivateCustomer(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE customers SET active=1 WHERE id=? AND active=0`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists int
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM customers WHERE id=?`, id).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}
func (r *OperationsRepository) GetCustomerDebt(ctx context.Context, id int64) (models.Customer, []models.Sale, error) {
	var c models.Customer
	var a int
	if err := r.db.QueryRowContext(ctx, `SELECT id,name,COALESCE(phone,''),active FROM customers WHERE id=?`, id).Scan(&c.ID, &c.Name, &c.Phone, &a); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return c, nil, err
	}
	c.Active = a == 1
	rows, err := r.db.QueryContext(ctx, `SELECT id,invoice_number,request_id,customer_id,user_id,subtotal,discount,total,amount_paid,remaining,amount_received,change_given,payment_method,created_at FROM sales WHERE customer_id=? ORDER BY created_at DESC`, id)
	if err != nil {
		return c, nil, err
	}
	defer rows.Close()
	var sales []models.Sale
	for rows.Next() {
		s, e := r.getSale(rows)
		if e != nil {
			return c, nil, e
		}
		sales = append(sales, s)
		c.Outstanding += s.Remaining
	}
	return c, sales, rows.Err()
}
func (r *OperationsRepository) CustomerOutstandingTx(ctx context.Context, tx *sql.Tx, id int64) (float64, error) {
	var outstanding float64
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(remaining),0) FROM sales WHERE customer_id=? AND remaining>0`, id).Scan(&outstanding)
	return outstanding, err
}
func (r *OperationsRepository) InsertCustomerPaymentTx(ctx context.Context, tx *sql.Tx, pay models.DebtPayment, shiftID *int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO customer_payments(sale_id,customer_id,amount,method,shift_id) VALUES(?,?,?,?,?)`, pay.SaleID, pay.CustomerID, pay.Amount, pay.Method, shiftID)
	return err
}
func (r *OperationsRepository) ApplyDebtPaymentTx(ctx context.Context, tx *sql.Tx, customerID int64, amount float64, method string, shiftID *int64) (float64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,remaining FROM sales WHERE customer_id=? AND remaining>0 ORDER BY created_at,id`, customerID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	left := amount
	var applied float64
	for rows.Next() && left > 0 {
		var sid int64
		var rem float64
		if err := rows.Scan(&sid, &rem); err != nil {
			return applied, err
		}
		use := rem
		if use > left {
			use = left
		}
		if _, err := tx.ExecContext(ctx, `UPDATE sales SET amount_paid=amount_paid+?,remaining=remaining-? WHERE id=?`, use, use, sid); err != nil {
			return applied, err
		}
		if err := r.InsertCustomerPaymentTx(ctx, tx, models.DebtPayment{SaleID: sid, CustomerID: customerID, Amount: use, Method: method}, shiftID); err != nil {
			return applied, err
		}
		left -= use
		applied += use
	}
	return applied, nil
}
func (r *OperationsRepository) InsertExpenseCategory(ctx context.Context, c models.ExpenseCategory) (models.ExpenseCategory, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO expense_categories(name) VALUES(?)`, c.Name)
	if err != nil {
		return c, err
	}
	c.ID, _ = res.LastInsertId()
	c.Active = true
	return c, nil
}
func (r *OperationsRepository) ListExpenseCategories(ctx context.Context) ([]models.ExpenseCategory, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,active FROM expense_categories WHERE active=1 ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ExpenseCategory
	for rows.Next() {
		var c models.ExpenseCategory
		var a int
		if err := rows.Scan(&c.ID, &c.Name, &a); err != nil {
			return nil, err
		}
		c.Active = a == 1
		out = append(out, c)
	}
	return out, rows.Err()
}
func (r *OperationsRepository) UpdateExpenseCategory(ctx context.Context, id int64, c models.ExpenseCategory) error {
	res, err := r.db.ExecContext(ctx, `UPDATE expense_categories SET name=? WHERE id=? AND active=1`, c.Name, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *OperationsRepository) DeactivateExpenseCategory(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE expense_categories SET active=0 WHERE id=? AND active=1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *OperationsRepository) ActivateExpenseCategory(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE expense_categories SET active=1 WHERE id=? AND active=0`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists int
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM expense_categories WHERE id=?`, id).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}
func (r *OperationsRepository) InsertExpenseTx(ctx context.Context, tx *sql.Tx, e models.Expense) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO expenses(expense_category_id,amount,payment_method,expense_date,notes,user_id,shift_id) VALUES(?,?,?,?,?,?,?)`, e.ExpenseCategoryID, e.Amount, e.PaymentMethod, e.ExpenseDate, e.Notes, e.UserID, e.ShiftID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func (r *OperationsRepository) InsertOwnerExpenseTx(ctx context.Context, tx *sql.Tx, e models.InventoryOwnerExpense) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO inventory_owner_expenses(product_id,user_id,quantity,unit_cost,total_value,notes) VALUES(?,?,?,?,?,?)`, e.ProductID, e.UserID, e.Quantity, e.UnitCost, e.TotalValue, e.Notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func (r *OperationsRepository) InsertSalesReturnTx(ctx context.Context, tx *sql.Tx, v models.SalesReturn) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO sales_returns(sale_id,customer_id,user_id,refund_type,linked_sale_id,total_value,notes) VALUES(?,?,?,?,?,?,?)`, v.SaleID, v.CustomerID, v.UserID, v.RefundType, v.LinkedSaleID, v.TotalValue, v.Notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func (r *OperationsRepository) InsertSalesReturnItemTx(ctx context.Context, tx *sql.Tx, id int64, i models.SalesReturnItem) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO sales_return_items(sales_return_id,product_id,quantity,unit_price,line_total) VALUES(?,?,?,?,?)`, id, i.ProductID, i.Quantity, i.UnitPrice, i.LineTotal)
	return err
}
func (r *OperationsRepository) SaleLineTx(ctx context.Context, tx *sql.Tx, saleID, productID int64) (float64, float64, error) {
	var q, p float64
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(quantity),0),COALESCE(SUM(line_total)/NULLIF(SUM(quantity),0),0) FROM sale_items WHERE sale_id=? AND product_id=?`, saleID, productID).Scan(&q, &p)
	return q, p, err
}
func (r *OperationsRepository) ReturnedSaleQtyTx(ctx context.Context, tx *sql.Tx, saleID, productID int64) (float64, error) {
	var q float64
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(ri.quantity),0) FROM sales_return_items ri JOIN sales_returns r ON r.id=ri.sales_return_id WHERE r.sale_id=? AND ri.product_id=?`, saleID, productID).Scan(&q)
	return q, err
}

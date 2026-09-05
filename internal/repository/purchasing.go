package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"supermarket/internal/models"
)

type PurchasingRepository struct{ db *sql.DB }

func NewPurchasingRepository(db *sql.DB) *PurchasingRepository { return &PurchasingRepository{db: db} }

func (r *PurchasingRepository) ListDealers(ctx context.Context) ([]models.Dealer, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT d.id,d.name,COALESCE(d.phone,''),COALESCE(d.address,''),COALESCE(d.notes,''),d.active,COALESCE((SELECT SUM(p.remaining) FROM purchases p WHERE p.dealer_id=d.id),0),COALESCE((SELECT SUM(c.amount-c.applied_amount) FROM dealer_credits c WHERE c.dealer_id=d.id),0) FROM dealers d WHERE d.active=1 ORDER BY d.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Dealer
	for rows.Next() {
		var d models.Dealer
		var a int
		if err := rows.Scan(&d.ID, &d.Name, &d.Phone, &d.Address, &d.Notes, &a, &d.Outstanding, &d.Credit); err != nil {
			return nil, err
		}
		d.Active = a == 1
		d.Balance = d.Outstanding - d.Credit
		if d.Balance < 0 {
			d.Balance = 0
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (r *PurchasingRepository) CreateDealer(ctx context.Context, d models.Dealer) (models.Dealer, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO dealers(name,phone,address,notes) VALUES (?,?,?,?)`, d.Name, d.Phone, d.Address, d.Notes)
	if err != nil {
		return d, err
	}
	d.ID, _ = res.LastInsertId()
	d.Active = true
	return d, nil
}
func (r *PurchasingRepository) UpdateDealer(ctx context.Context, id int64, d models.Dealer) error {
	res, err := r.db.ExecContext(ctx, `UPDATE dealers SET name=?,phone=?,address=?,notes=? WHERE id=? AND active=1`, d.Name, d.Phone, d.Address, d.Notes, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *PurchasingRepository) DeactivateDealer(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE dealers SET active=0 WHERE id=? AND active=1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PurchasingRepository) ActivateDealer(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE dealers SET active=1 WHERE id=? AND active=0`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists int
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dealers WHERE id=?`, id).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func scanPurchase(row *sql.Row) (models.Purchase, error) {
	var p models.Purchase
	var invoice, created sql.NullString
	err := row.Scan(&p.ID, &p.DealerID, &p.UserID, &invoice, &p.Subtotal, &p.Discount, &p.Total, &p.Paid, &p.Remaining, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	if invoice.Valid {
		p.DealerInvoiceNumber = invoice.String
	}
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created.String)
	return p, nil
}
func (r *PurchasingRepository) GetPurchase(ctx context.Context, id int64) (models.Purchase, error) {
	p, err := scanPurchase(r.db.QueryRowContext(ctx, `SELECT id,dealer_id,user_id,dealer_invoice_number,subtotal,discount,total,paid,remaining,created_at FROM purchases WHERE id=?`, id))
	if err != nil {
		return p, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,product_id,quantity,unit_cost,line_total FROM purchase_items WHERE purchase_id=?`, id)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	for rows.Next() {
		var i models.PurchaseItem
		if err := rows.Scan(&i.ID, &i.ProductID, &i.Quantity, &i.UnitCost, &i.LineTotal); err != nil {
			return p, err
		}
		p.Items = append(p.Items, i)
	}
	return p, rows.Err()
}
func (r *PurchasingRepository) ListPurchases(ctx context.Context, dealerID int64) ([]models.Purchase, error) {
	q := `SELECT id,dealer_id,user_id,dealer_invoice_number,subtotal,discount,total,paid,remaining,created_at FROM purchases`
	args := []any{}
	if dealerID > 0 {
		q += ` WHERE dealer_id=?`
		args = append(args, dealerID)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Purchase
	for rows.Next() {
		p, err := scanPurchaseScanner(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanPurchaseScanner(s rowScanner) (models.Purchase, error) {
	var p models.Purchase
	var invoice, created sql.NullString
	err := s.Scan(&p.ID, &p.DealerID, &p.UserID, &invoice, &p.Subtotal, &p.Discount, &p.Total, &p.Paid, &p.Remaining, &created)
	if err != nil {
		return p, err
	}
	if invoice.Valid {
		p.DealerInvoiceNumber = invoice.String
	}
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created.String)
	return p, nil
}

func (r *PurchasingRepository) InsertPurchaseTx(ctx context.Context, tx *sql.Tx, p models.Purchase) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO purchases(dealer_id,user_id,dealer_invoice_number,subtotal,discount,total,paid,remaining) VALUES (?,?,?,?,?,?,?,?)`, p.DealerID, p.UserID, p.DealerInvoiceNumber, p.Subtotal, p.Discount, p.Total, 0, p.Total)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func (r *PurchasingRepository) InsertPurchaseItemTx(ctx context.Context, tx *sql.Tx, purchaseID int64, i models.PurchaseItem) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO purchase_items(purchase_id,product_id,quantity,unit_cost,line_total) VALUES (?,?,?,?,?)`, purchaseID, i.ProductID, i.Quantity, i.UnitCost, i.LineTotal)
	return err
}
func (r *PurchasingRepository) ProductForUpdate(ctx context.Context, tx *sql.Tx, id int64) (string, float64, float64, error) {
	var unit string
	var qty, cost sql.NullFloat64
	err := tx.QueryRowContext(ctx, `SELECT unit_type,quantity,COALESCE(purchase_price,0) FROM products WHERE id=? AND active=1`, id).Scan(&unit, &qty, &cost)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, 0, ErrNotFound
	}
	return unit, qty.Float64, cost.Float64, err
}
func (r *PurchasingRepository) ChangeStockTx(ctx context.Context, tx *sql.Tx, productID int64, delta float64, typ string, refID, userID int64, notes string) (float64, error) {
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
	_, err := tx.ExecContext(ctx, `INSERT INTO stock_movements(product_id,type,quantity_change,balance_after,reference_type,reference_id,user_id,notes) VALUES (?,?,?,?,?,?,?,?)`, productID, typ, delta, next, typ, refID, userID, notes)
	return next, err
}
func (r *PurchasingRepository) SetProductCostTx(ctx context.Context, tx *sql.Tx, productID int64, cost float64) error {
	_, err := tx.ExecContext(ctx, `UPDATE products SET purchase_price=CASE WHEN unit_type <> 'weight' THEN ? ELSE purchase_price END,purchase_price_per_kg=CASE WHEN unit_type='weight' THEN ? ELSE purchase_price_per_kg END WHERE id=?`, cost, cost, productID)
	return err
}
func (r *PurchasingRepository) Begin(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}
func (r *PurchasingRepository) UpdatePurchasePaymentTx(ctx context.Context, tx *sql.Tx, purchaseID int64, amount float64) (models.Purchase, error) {
	var p models.Purchase
	var invoice, created sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT id,dealer_id,user_id,dealer_invoice_number,subtotal,discount,total,paid,remaining,created_at FROM purchases WHERE id=?`, purchaseID).Scan(&p.ID, &p.DealerID, &p.UserID, &invoice, &p.Subtotal, &p.Discount, &p.Total, &p.Paid, &p.Remaining, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return p, ErrNotFound
		}
		return p, err
	}
	p.DealerInvoiceNumber = invoice.String
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created.String)
	p.Paid += amount
	p.Remaining = p.Total - p.Paid
	if p.Remaining < 0 {
		p.Remaining = 0
	}
	if _, err := tx.ExecContext(ctx, `UPDATE purchases SET paid=?,remaining=? WHERE id=?`, p.Paid, p.Remaining, purchaseID); err != nil {
		return p, err
	}
	return p, nil
}

func (r *PurchasingRepository) PurchaseBalanceTx(ctx context.Context, tx *sql.Tx, purchaseID int64) (int64, float64, error) {
	var dealerID int64
	var remaining float64
	err := tx.QueryRowContext(ctx, `SELECT dealer_id,remaining FROM purchases WHERE id=?`, purchaseID).Scan(&dealerID, &remaining)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrNotFound
	}
	return dealerID, remaining, err
}
func (r *PurchasingRepository) InsertPaymentTx(ctx context.Context, tx *sql.Tx, purchaseID int64, amount float64, method string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO dealer_payments(purchase_id,amount,method) VALUES (?,?,?)`, purchaseID, amount, method)
	return err
}
func (r *PurchasingRepository) InsertReturnTx(ctx context.Context, tx *sql.Tx, ret models.PurchaseReturn) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO purchase_returns(purchase_id,user_id,reason,total_value,notes) VALUES (?,?,?,?,?)`, ret.PurchaseID, ret.UserID, ret.Reason, ret.TotalValue, ret.Notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func (r *PurchasingRepository) InsertReturnItemTx(ctx context.Context, tx *sql.Tx, id int64, i models.PurchaseReturnItem) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO purchase_return_items(purchase_return_id,product_id,quantity,unit_cost,line_total) VALUES (?,?,?,?,?)`, id, i.ProductID, i.Quantity, i.UnitCost, i.LineTotal)
	return err
}
func (r *PurchasingRepository) ReturnedQuantityTx(ctx context.Context, tx *sql.Tx, purchaseID, productID int64) (float64, error) {
	var q sql.NullFloat64
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(ri.quantity),0) FROM purchase_return_items ri JOIN purchase_returns r ON r.id=ri.purchase_return_id WHERE r.purchase_id=? AND ri.product_id=?`, purchaseID, productID).Scan(&q)
	return q.Float64, err
}
func (r *PurchasingRepository) PurchaseItemQuantityTx(ctx context.Context, tx *sql.Tx, purchaseID, productID int64) (float64, float64, error) {
	var q, c float64
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(quantity),0),COALESCE(SUM(line_total)/NULLIF(SUM(quantity),0),0) FROM purchase_items WHERE purchase_id=? AND product_id=?`, purchaseID, productID).Scan(&q, &c)
	return q, c, err
}
func (r *PurchasingRepository) OffsetPurchaseTx(ctx context.Context, tx *sql.Tx, purchaseID int64, value float64) error {
	_, err := tx.ExecContext(ctx, `UPDATE purchases SET remaining=MAX(0,remaining-?) WHERE id=?`, value, purchaseID)
	return err
}

func (r *PurchasingRepository) ApplyPurchaseReturnTx(ctx context.Context, tx *sql.Tx, purchaseID int64, value float64) (int64, float64, error) {
	dealerID, remaining, err := r.PurchaseBalanceTx(ctx, tx, purchaseID)
	if err != nil {
		return 0, 0, err
	}
	applied := value
	if applied > remaining {
		applied = remaining
	}
	credit := value - applied
	if err := r.OffsetPurchaseTx(ctx, tx, purchaseID, applied); err != nil {
		return 0, 0, err
	}
	return dealerID, credit, nil
}

func (r *PurchasingRepository) InsertDealerCreditTx(ctx context.Context, tx *sql.Tx, dealerID, returnID int64, amount float64) error {
	if amount <= 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO dealer_credits(dealer_id,purchase_return_id,amount) VALUES (?,?,?)`, dealerID, returnID, amount)
	return err
}

func (r *PurchasingRepository) ListReturns(ctx context.Context, purchaseID int64) ([]models.PurchaseReturn, error) {
	q := `SELECT r.id,r.purchase_id,r.user_id,r.reason,r.total_value,COALESCE(r.notes,''),r.created_at,COALESCE(c.amount-c.applied_amount,0) FROM purchase_returns r LEFT JOIN dealer_credits c ON c.purchase_return_id=r.id`
	args := []any{}
	if purchaseID > 0 {
		q += ` WHERE r.purchase_id=?`
		args = append(args, purchaseID)
	}
	q += ` ORDER BY r.created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var out []models.PurchaseReturn
	for rows.Next() {
		var v models.PurchaseReturn
		var created string
		if err := rows.Scan(&v.ID, &v.PurchaseID, &v.UserID, &v.Reason, &v.TotalValue, &v.Notes, &created, &v.CreditAmount); err != nil {
			return nil, err
		}
		v.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
		out = append(out, v)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		itemRows, err := r.db.QueryContext(ctx, `SELECT id,product_id,quantity,unit_cost,line_total FROM purchase_return_items WHERE purchase_return_id=?`, out[i].ID)
		if err != nil {
			return nil, err
		}
		for itemRows.Next() {
			var item models.PurchaseReturnItem
			if err := itemRows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.UnitCost, &item.LineTotal); err != nil {
				itemRows.Close()
				return nil, err
			}
			out[i].Items = append(out[i].Items, item)
		}
		if err := itemRows.Err(); err != nil {
			itemRows.Close()
			return nil, err
		}
		itemRows.Close()
	}
	return out, nil
}

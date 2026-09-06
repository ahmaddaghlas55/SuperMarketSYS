package repository

import (
	"context"
	"database/sql"
	"time"

	"supermarket/internal/models"
)

type ReportsRepository struct{ db *sql.DB }

func NewReportsRepository(db *sql.DB) *ReportsRepository { return &ReportsRepository{db: db} }

func reportDates(from, to string) (string, string) {
	if from == "" {
		from = "0000-01-01"
	}
	if to == "" {
		to = "9999-12-31"
	}
	return from, to
}

func (r *ReportsRepository) Sales(ctx context.Context, from, to string) (models.SalesReport, error) {
	from, to = reportDates(from, to)
	var out models.SalesReport
	out.From, out.To = from, to
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total + discount),0), COALESCE(SUM(discount),0),
		       COALESCE(SUM(total),0), COALESCE(SUM(amount_paid),0), COALESCE(SUM(remaining),0)
		FROM sales WHERE date(created_at)>=? AND date(created_at)<=?`, from, to).
		Scan(&out.InvoiceCount, &out.GrossSales, &out.DiscountsGiven, &out.NetSales, &out.PaidTotal, &out.RemainingTotal)
	if err != nil {
		return out, err
	}

	if out.InvoiceCount > 0 {
		out.AverageInvoice = out.NetSales / float64(out.InvoiceCount)
	}
	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_value),0) FROM sales_returns
		WHERE date(created_at)>=? AND date(created_at)<=?`, from, to).Scan(&out.ReturnsTotal); err != nil {
		return out, err
	}
	out.NetSales -= out.ReturnsTotal
	return out, nil
}

func (r *ReportsRepository) SalesRows(ctx context.Context, from, to string) ([]models.Sale, error) {
	from, to = reportDates(from, to)
	rows, err := r.db.QueryContext(ctx, `SELECT id,invoice_number,request_id,customer_id,user_id,subtotal,discount,total,amount_paid,remaining,amount_received,change_given,payment_method,created_at FROM sales WHERE date(created_at)>=? AND date(created_at)<=? ORDER BY created_at DESC,id DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Sale
	for rows.Next() {
		s, err := scanSale(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ReportsRepository) DebtSales(ctx context.Context) ([]models.Sale, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,invoice_number,request_id,customer_id,user_id,subtotal,discount,total,amount_paid,remaining,amount_received,change_given,payment_method,created_at FROM sales WHERE remaining>0 ORDER BY created_at DESC,id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Sale
	for rows.Next() {
		s, err := scanSale(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func scanSale(s interface{ Scan(...any) error }) (models.Sale, error) {
	var v models.Sale
	var customer sql.NullInt64
	var received, change sql.NullFloat64
	var created string
	if err := s.Scan(&v.ID, &v.InvoiceNumber, &v.RequestID, &customer, &v.UserID, &v.Subtotal, &v.Discount, &v.Total, &v.AmountPaid, &v.Remaining, &received, &change, &v.PaymentMethod, &created); err != nil {
		return v, err
	}
	if customer.Valid {
		v.CustomerID = &customer.Int64
	}
	if received.Valid {
		v.AmountReceived = &received.Float64
	}
	if change.Valid {
		v.ChangeGiven = &change.Float64
	}
	v.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
	return v, nil
}

func (r *ReportsRepository) Purchases(ctx context.Context, from, to string) (models.PurchasesReport, error) {
	from, to = reportDates(from, to)
	var out models.PurchasesReport
	out.From, out.To = from, to
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total),0), COALESCE(SUM(paid),0)
		FROM purchases WHERE date(created_at)>=? AND date(created_at)<=?`,
		from, to).Scan(&out.PurchaseCount, &out.GrossPurchases, &out.PaidTotal); err != nil {
		return out, err
	}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_value),0) FROM purchase_returns
		WHERE date(created_at)>=? AND date(created_at)<=?`, from, to).Scan(&out.PurchaseReturns); err != nil {
		return out, err
	}
	out.NetPurchases = out.GrossPurchases - out.PurchaseReturns
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.name,
		       MAX(0, COALESCE((SELECT SUM(p.remaining) FROM purchases p WHERE p.dealer_id=d.id),0)
		              - COALESCE((SELECT SUM(c.amount-c.applied_amount) FROM dealer_credits c WHERE c.dealer_id=d.id),0))
		FROM dealers d WHERE d.active=1 ORDER BY d.name`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var balance models.DealerBalance
		if err := rows.Scan(&balance.DealerID, &balance.DealerName, &balance.Balance); err != nil {
			return out, err
		}
		out.DealerBalances = append(out.DealerBalances, balance)
	}
	return out, rows.Err()
}

func (r *ReportsRepository) PurchaseRows(ctx context.Context, from, to string) ([]models.Purchase, error) {
	from, to = reportDates(from, to)
	rows, err := r.db.QueryContext(ctx, `SELECT id,dealer_id,user_id,dealer_invoice_number,subtotal,discount,total,paid,credit_applied,remaining,created_at FROM purchases WHERE date(created_at)>=? AND date(created_at)<=? ORDER BY created_at DESC,id DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Purchase
	for rows.Next() {
		var p models.Purchase
		var invoice, created sql.NullString
		if err := rows.Scan(&p.ID, &p.DealerID, &p.UserID, &invoice, &p.Subtotal, &p.Discount, &p.Total, &p.Paid, &p.CreditApplied, &p.Remaining, &created); err != nil {
			return nil, err
		}
		p.DealerInvoiceNumber = invoice.String
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created.String)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ReportsRepository) Stock(ctx context.Context) (models.StockReport, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT p.id,p.barcode,p.name,p.category_id,p.unit_type,p.sale_price,p.purchase_price,p.price_per_kg,p.purchase_price_per_kg,p.price_per_piece,p.price_per_carton,p.pieces_per_carton,p.quantity,p.active,p.created_at,COALESCE(c.low_stock_threshold,0) FROM products p LEFT JOIN categories c ON c.id=p.category_id WHERE p.active=1 ORDER BY p.name`)
	if err != nil {
		return models.StockReport{}, err
	}
	defer rows.Close()
	var out models.StockReport
	for rows.Next() {
		var p models.Product
		var barcode, created sql.NullString
		var category, pieces sql.NullInt64
		var sale, purchase, ppkg, cppkg, ppiece, pcarton sql.NullFloat64
		var active, threshold int
		if err := rows.Scan(&p.ID, &barcode, &p.Name, &category, &p.UnitType, &sale, &purchase, &ppkg, &cppkg, &ppiece, &pcarton, &pieces, &p.Quantity, &active, &created, &threshold); err != nil {
			return out, err
		}
		if barcode.Valid {
			p.Barcode = &barcode.String
		}
		if category.Valid {
			p.CategoryID = &category.Int64
		}
		if sale.Valid {
			p.SalePrice = &sale.Float64
		}
		if purchase.Valid {
			p.PurchasePrice = &purchase.Float64
		}
		if ppkg.Valid {
			p.PricePerKg = &ppkg.Float64
		}
		if cppkg.Valid {
			p.PurchasePricePerKg = &cppkg.Float64
		}
		if ppiece.Valid {
			p.PricePerPiece = &ppiece.Float64
		}
		if pcarton.Valid {
			p.PricePerCarton = &pcarton.Float64
		}
		if pieces.Valid {
			p.PiecesPerCarton = &pieces.Int64
		}
		p.Active, p.LowStock = active == 1, p.Quantity <= float64(threshold)
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created.String)
		cost := purchaseCost(p)
		out.ProductCount++
		out.TotalStockValue += p.Quantity * cost
		if p.LowStock {
			out.LowStockProducts = append(out.LowStockProducts, p)
		}
		if p.Quantity <= 0 {
			out.OutOfStockProducts = append(out.OutOfStockProducts, p)
		}
	}
	return out, rows.Err()
}

func purchaseCost(p models.Product) float64 {
	if p.UnitType == "weight" && p.PurchasePricePerKg != nil {
		return *p.PurchasePricePerKg
	}
	if p.PurchasePrice != nil {
		return *p.PurchasePrice
	}
	return 0
}

func (r *ReportsRepository) Profit(ctx context.Context, from, to string) (models.ProfitReport, error) {
	from, to = reportDates(from, to)
	var out models.ProfitReport
	out.From, out.To = from, to
	var returns float64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total),0) FROM sales
		WHERE date(created_at)>=? AND date(created_at)<=?`, from, to).Scan(&out.NetSales); err != nil {
		return out, err
	}
	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_value),0) FROM sales_returns
		WHERE date(created_at)>=? AND date(created_at)<=?`, from, to).Scan(&returns); err != nil {
		return out, err
	}
	out.NetSales -= returns
	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(si.quantity*si.cost_price),0)
		FROM sales s JOIN sale_items si ON si.sale_id=s.id
		WHERE date(s.created_at)>=? AND date(s.created_at)<=?`, from, to).Scan(&out.CostOfGoodsSold); err != nil {
		return out, err
	}
	var returnedCost float64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(ri.quantity * (
			SELECT COALESCE(SUM(si.quantity * si.cost_price) / NULLIF(SUM(si.quantity),0),0)
			FROM sale_items si
			WHERE si.sale_id=sr.sale_id AND si.product_id=ri.product_id
		)),0)
		FROM sales_returns sr
		JOIN sales_return_items ri ON ri.sales_return_id=sr.id
		WHERE date(sr.created_at)>=? AND date(sr.created_at)<=?`, from, to).Scan(&returnedCost); err != nil {
		return out, err
	}
	out.CostOfGoodsSold -= returnedCost
	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(e.amount),0) FROM expenses e
		WHERE e.expense_date>=? AND e.expense_date<=?`, from, to).Scan(&out.GeneralExpenses); err != nil {
		return out, err
	}
	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_value),0) FROM inventory_owner_expenses
		WHERE date(created_at)>=? AND date(created_at)<=?`, from, to).Scan(&out.LossOfProfit); err != nil {
		return out, err
	}
	out.GrossProfit = out.NetSales - out.CostOfGoodsSold
	out.NetProfit = out.GrossProfit - out.GeneralExpenses
	return out, nil
}

func (r *ReportsRepository) Dashboard(ctx context.Context, date string) (models.DashboardReport, error) {
	var d models.DashboardReport
	d.Date = date
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(total),0) FROM sales WHERE date(created_at)=?`, date).Scan(&d.TodaySalesTotal)
	if err != nil {
		return d, err
	}
	if err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM products p LEFT JOIN categories c ON c.id=p.category_id WHERE p.active=1 AND p.quantity<=COALESCE(c.low_stock_threshold,0)`).Scan(&d.LowStockCount); err != nil {
		return d, err
	}
	var open int
	if err = r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cash_shifts WHERE closed_at IS NULL)`).Scan(&open); err != nil {
		return d, err
	}
	d.OpenShift = open == 1
	if err = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM expenses WHERE expense_date=?`, date).Scan(&d.TodayExpenses); err != nil {
		return d, err
	}
	if err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT id FROM sales_returns WHERE date(created_at)=?
			UNION ALL
			SELECT id FROM purchase_returns WHERE date(created_at)=?
		)`, date, date).Scan(&d.TodayReturnsCount); err != nil {
		return d, err
	}
	profit, err := r.Profit(ctx, date, date)
	if err != nil {
		return d, err
	}
	d.TodayProfit = profit.NetProfit
	return d, nil
}

func (r *ReportsRepository) Expenses(ctx context.Context, from, to string) ([]models.ExpenseReportRow, error) {
	from, to = reportDates(from, to)
	rows, err := r.db.QueryContext(ctx, `SELECT id,expense_category_id,amount,payment_method,expense_date,COALESCE(notes,''),user_id FROM expenses WHERE expense_date>=? AND expense_date<=? ORDER BY expense_date DESC,id DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ExpenseReportRow
	for rows.Next() {
		var v models.ExpenseReportRow
		if err := rows.Scan(&v.ID, &v.ExpenseCategoryID, &v.Amount, &v.PaymentMethod, &v.ExpenseDate, &v.Notes, &v.UserID); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	ownerRows, err := r.db.QueryContext(ctx, `SELECT id,product_id,total_value,COALESCE(notes,''),user_id,date(created_at) FROM inventory_owner_expenses WHERE date(created_at)>=? AND date(created_at)<=? ORDER BY created_at DESC,id DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer ownerRows.Close()
	for ownerRows.Next() {
		var v models.ExpenseReportRow
		if err := ownerRows.Scan(&v.ID, &v.ExpenseCategoryID, &v.Amount, &v.Notes, &v.UserID, &v.ExpenseDate); err != nil {
			return nil, err
		}
		v.PaymentMethod = "inventory_owner"
		out = append(out, v)
	}
	return out, ownerRows.Err()
}

func (r *ReportsRepository) Returns(ctx context.Context, from, to string) ([]models.ReturnReportRow, error) {
	from, to = reportDates(from, to)
	rows, err := r.db.QueryContext(ctx, `SELECT r.id,'sales',r.sale_id,ri.product_id,ri.quantity,ri.line_total,'',r.created_at FROM sales_returns r JOIN sales_return_items ri ON ri.sales_return_id=r.id WHERE date(r.created_at)>=? AND date(r.created_at)<=?
		UNION ALL
		SELECT r.id,'purchase',r.purchase_id,ri.product_id,ri.quantity,ri.line_total,r.reason,r.created_at FROM purchase_returns r JOIN purchase_return_items ri ON ri.purchase_return_id=r.id WHERE date(r.created_at)>=? AND date(r.created_at)<=?
		ORDER BY 8 DESC,1 DESC`, from, to, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ReturnReportRow
	for rows.Next() {
		var v models.ReturnReportRow
		if err := rows.Scan(&v.ID, &v.ReturnType, &v.ReferenceID, &v.ProductID, &v.Quantity, &v.Value, &v.Reason, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

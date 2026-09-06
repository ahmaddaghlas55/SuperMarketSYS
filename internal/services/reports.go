package services

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
	"supermarket/internal/models"
	"supermarket/internal/repository"
)

type ReportsService struct{ repo *repository.ReportsRepository }

func NewReportsService(r *repository.ReportsRepository) *ReportsService {
	return &ReportsService{repo: r}
}

func (s *ReportsService) Sales(ctx context.Context, from, to string) (models.SalesReport, error) {
	if err := validateDateRange(from, to); err != nil {
		return models.SalesReport{}, err
	}
	return s.repo.Sales(ctx, from, to)
}
func (s *ReportsService) Purchases(ctx context.Context, from, to string) (models.PurchasesReport, error) {
	if err := validateDateRange(from, to); err != nil {
		return models.PurchasesReport{}, err
	}
	return s.repo.Purchases(ctx, from, to)
}
func (s *ReportsService) Stock(ctx context.Context) (models.StockReport, error) {
	return s.repo.Stock(ctx)
}
func (s *ReportsService) Profit(ctx context.Context, from, to string) (models.ProfitReport, error) {
	if err := validateDateRange(from, to); err != nil {
		return models.ProfitReport{}, err
	}
	return s.repo.Profit(ctx, from, to)
}
func (s *ReportsService) Dashboard(ctx context.Context, date string) (models.DashboardReport, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if err := validateDateRange(date, date); err != nil {
		return models.DashboardReport{}, err
	}
	return s.repo.Dashboard(ctx, date)
}

func workbook(headers []string, rows [][]any) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return nil, err
		}
	}
	for r, row := range rows {
		for c, value := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
	}
	var out bytes.Buffer
	if err := f.Write(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func exportID(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func (s *ReportsService) ExportSales(ctx context.Context, from, to string) ([]byte, error) {
	sales, err := s.repo.SalesRows(ctx, from, to)
	if err != nil {
		return nil, err
	}
	rows := make([][]any, 0, len(sales))
	for _, v := range sales {
		rows = append(rows, []any{v.ID, v.InvoiceNumber, exportID(v.CustomerID), v.Subtotal, v.Discount, v.Total, v.AmountPaid, v.Remaining, v.PaymentMethod, v.CreatedAt})
	}
	return workbook([]string{"id", "invoice_number", "customer_id", "subtotal", "discount", "total", "amount_paid", "remaining", "payment_method", "created_at"}, rows)
}

func (s *ReportsService) ExportDebt(ctx context.Context) ([]byte, error) {
	sales, err := s.repo.DebtSales(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([][]any, 0, len(sales))
	for _, v := range sales {
		rows = append(rows, []any{exportID(v.CustomerID), v.ID, v.InvoiceNumber, v.Total, v.AmountPaid, v.Remaining, v.CreatedAt})
	}
	return workbook([]string{"customer_id", "sale_id", "invoice_number", "total", "amount_paid", "remaining", "created_at"}, rows)
}

func (s *ReportsService) ExportPurchases(ctx context.Context, from, to string) ([]byte, error) {
	purchases, err := s.repo.PurchaseRows(ctx, from, to)
	if err != nil {
		return nil, err
	}
	rows := make([][]any, 0, len(purchases))
	for _, v := range purchases {
		rows = append(rows, []any{v.ID, v.DealerID, v.Total, v.Paid, v.CreditApplied, v.Remaining, v.CreatedAt})
	}
	return workbook([]string{"id", "dealer_id", "total", "paid", "credit_applied", "remaining", "created_at"}, rows)
}

func (s *ReportsService) ExportExpenses(ctx context.Context, from, to string) ([]byte, error) {
	expenses, err := s.repo.Expenses(ctx, from, to)
	if err != nil {
		return nil, err
	}
	rows := make([][]any, 0, len(expenses))
	for _, v := range expenses {
		rows = append(rows, []any{v.ID, v.ExpenseCategoryID, v.Amount, v.PaymentMethod, v.ExpenseDate, v.Notes, v.UserID})
	}
	return workbook([]string{"id", "expense_category_id", "amount", "payment_method", "expense_date", "notes", "user_id"}, rows)
}

func (s *ReportsService) ExportReturns(ctx context.Context, from, to string) ([]byte, error) {
	returns, err := s.repo.Returns(ctx, from, to)
	if err != nil {
		return nil, err
	}
	rows := make([][]any, 0, len(returns))
	for _, v := range returns {
		rows = append(rows, []any{v.ID, v.ReturnType, v.ReferenceID, v.ProductID, v.Quantity, v.Value, v.Reason, v.CreatedAt})
	}
	return workbook([]string{"id", "return_type", "reference_id", "product_id", "quantity", "value", "reason", "created_at"}, rows)
}

func validateDateRange(from, to string) error {
	if from != "" && len(from) != 10 || to != "" && len(to) != 10 {
		return fmt.Errorf("%w: invalid date", ErrValidation)
	}
	if from != "" {
		if _, err := time.Parse("2006-01-02", from); err != nil {
			return fmt.Errorf("%w: invalid date", ErrValidation)
		}
	}
	if to != "" {
		if _, err := time.Parse("2006-01-02", to); err != nil {
			return fmt.Errorf("%w: invalid date", ErrValidation)
		}
	}
	if from != "" && to != "" && from > to {
		return fmt.Errorf("%w: date range", ErrValidation)
	}
	return nil
}

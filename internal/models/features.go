package models

import "time"

type Promotion struct {
	ID             int64      `json:"id"`
	ProductID      int64      `json:"product_id"`
	PromoType      string     `json:"promo_type"`
	Value          float64    `json:"value"`
	BundleQuantity *int64     `json:"bundle_quantity,omitempty"`
	StartsAt       *time.Time `json:"starts_at,omitempty"`
	EndsAt         *time.Time `json:"ends_at,omitempty"`
	UntilStockZero bool       `json:"until_stock_zero"`
	Active         bool       `json:"active"`
	CreatedAt      time.Time  `json:"created_at"`
}

type AuditEntry struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Action      string    `json:"action"`
	Module      string    `json:"module"`
	RecordID    *int64    `json:"record_id,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProfitReport struct {
	From            string  `json:"from"`
	To              string  `json:"to"`
	NetSales        float64 `json:"net_sales"`
	CostOfGoodsSold float64 `json:"cost_of_goods_sold"`
	GrossProfit     float64 `json:"gross_profit"`
	GeneralExpenses float64 `json:"general_expenses"`
	NetProfit       float64 `json:"net_profit"`
	LossOfProfit    float64 `json:"loss_of_profit"`
}

type SalesReport struct {
	From           string  `json:"from"`
	To             string  `json:"to"`
	InvoiceCount   int64   `json:"invoice_count"`
	GrossSales     float64 `json:"gross_sales"`
	DiscountsGiven float64 `json:"discounts_given"`
	ReturnsTotal   float64 `json:"returns_total"`
	NetSales       float64 `json:"net_sales"`
	AverageInvoice float64 `json:"average_invoice"`
	PaidTotal      float64 `json:"paid_total"`
	RemainingTotal float64 `json:"remaining_total"`
}

type DealerBalance struct {
	DealerID   int64   `json:"dealer_id"`
	DealerName string  `json:"dealer_name"`
	Balance    float64 `json:"balance"`
}

type PurchasesReport struct {
	From            string          `json:"from"`
	To              string          `json:"to"`
	PurchaseCount   int64           `json:"purchase_count"`
	GrossPurchases  float64         `json:"gross_purchases"`
	PurchaseReturns float64         `json:"purchase_returns"`
	NetPurchases    float64         `json:"net_purchases"`
	PaidTotal       float64         `json:"paid_total"`
	DealerBalances  []DealerBalance `json:"dealer_balances"`
}

type StockReport struct {
	ProductCount       int64     `json:"product_count"`
	TotalStockValue    float64   `json:"total_stock_value"`
	LowStockProducts   []Product `json:"low_stock_products"`
	OutOfStockProducts []Product `json:"out_of_stock_products"`
}

type DashboardReport struct {
	Date              string  `json:"date"`
	TodaySalesTotal   float64 `json:"today_sales_total"`
	TodayProfit       float64 `json:"today_profit"`
	LowStockCount     int64   `json:"low_stock_count"`
	OpenShift         bool    `json:"open_shift"`
	TodayExpenses     float64 `json:"today_expenses"`
	TodayReturnsCount int64   `json:"today_returns_count"`
}

type ExpenseReportRow struct {
	ID                int64   `json:"id"`
	ExpenseCategoryID int64   `json:"expense_category_id"`
	Amount            float64 `json:"amount"`
	PaymentMethod     string  `json:"payment_method"`
	ExpenseDate       string  `json:"expense_date"`
	Notes             string  `json:"notes,omitempty"`
	UserID            int64   `json:"user_id"`
}

type ReturnReportRow struct {
	ID          int64   `json:"id"`
	ReturnType  string  `json:"return_type"`
	ReferenceID int64   `json:"reference_id"`
	ProductID   int64   `json:"product_id"`
	Quantity    float64 `json:"quantity"`
	Value       float64 `json:"value"`
	Reason      string  `json:"reason,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

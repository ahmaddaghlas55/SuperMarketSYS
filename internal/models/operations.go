package models

import "time"

type Customer struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Phone       string  `json:"phone,omitempty"`
	Active      bool    `json:"active"`
	Outstanding float64 `json:"outstanding"`
}

type SaleItem struct {
	ID            int64    `json:"id"`
	ProductID     int64    `json:"product_id"`
	Quantity      float64  `json:"quantity"`
	UnitPrice     float64  `json:"unit_price"`
	CostPrice     float64  `json:"cost_price"`
	PriceOverride *float64 `json:"price_override,omitempty"`
	IsOverride    bool     `json:"is_override"`
	LineTotal     float64  `json:"line_total"`
}

type Sale struct {
	ID             int64      `json:"id"`
	InvoiceNumber  string     `json:"invoice_number"`
	RequestID      string     `json:"request_id"`
	CustomerID     *int64     `json:"customer_id,omitempty"`
	UserID         int64      `json:"user_id"`
	Subtotal       float64    `json:"subtotal"`
	Discount       float64    `json:"discount"`
	Total          float64    `json:"total"`
	AmountPaid     float64    `json:"amount_paid"`
	Remaining      float64    `json:"remaining"`
	AmountReceived *float64   `json:"amount_received,omitempty"`
	ChangeGiven    *float64   `json:"change_given,omitempty"`
	PaymentMethod  string     `json:"payment_method"`
	Items          []SaleItem `json:"items,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	Aging          bool       `json:"aging"`
}

type DebtPayment struct {
	ID         int64     `json:"id"`
	SaleID     int64     `json:"sale_id"`
	CustomerID int64     `json:"customer_id"`
	Amount     float64   `json:"amount"`
	Method     string    `json:"method"`
	CreatedAt  time.Time `json:"created_at"`
}

type SalesReturnItem struct {
	ProductID int64   `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	LineTotal float64 `json:"line_total"`
}

type SalesReturn struct {
	ID            int64             `json:"id"`
	SaleID        int64             `json:"sale_id"`
	CustomerID    *int64            `json:"customer_id,omitempty"`
	UserID        int64             `json:"user_id"`
	RefundType    string            `json:"refund_type"`
	LinkedSaleID  *int64            `json:"linked_sale_id,omitempty"`
	TotalValue    float64           `json:"total_value"`
	NetDifference float64           `json:"net_difference"`
	Notes         string            `json:"notes,omitempty"`
	Items         []SalesReturnItem `json:"items"`
	CreatedAt     time.Time         `json:"created_at"`
}

type CashShift struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	OpeningBalance  float64    `json:"opening_balance"`
	CashIn          float64    `json:"cash_in"`
	CashOut         float64    `json:"cash_out"`
	RunningBalance  float64    `json:"running_balance"`
	OpeningNotes    string     `json:"opening_notes,omitempty"`
	OpenedAt        time.Time  `json:"opened_at"`
	ExpectedClosing *float64   `json:"expected_closing,omitempty"`
	ActualClosing   *float64   `json:"actual_closing,omitempty"`
	Difference      *float64   `json:"difference,omitempty"`
	ClosingNotes    string     `json:"closing_notes,omitempty"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
}

type CashMovement struct {
	ID            int64     `json:"id"`
	ShiftID       int64     `json:"shift_id"`
	Type          string    `json:"type"`
	Direction     string    `json:"direction"`
	Amount        float64   `json:"amount"`
	ReferenceType string    `json:"reference_type,omitempty"`
	ReferenceID   *int64    `json:"reference_id,omitempty"`
	Notes         string    `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type ExpenseCategory struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type Expense struct {
	ID                int64   `json:"id"`
	ExpenseCategoryID int64   `json:"expense_category_id"`
	Amount            float64 `json:"amount"`
	PaymentMethod     string  `json:"payment_method"`
	ExpenseDate       string  `json:"expense_date"`
	Notes             string  `json:"notes,omitempty"`
	UserID            int64   `json:"user_id"`
	ShiftID           *int64  `json:"shift_id,omitempty"`
}

type InventoryOwnerExpense struct {
	ID         int64     `json:"id"`
	ProductID  int64     `json:"product_id"`
	UserID     int64     `json:"user_id"`
	Quantity   float64   `json:"quantity"`
	UnitCost   float64   `json:"unit_cost"`
	TotalValue float64   `json:"total_value"`
	Notes      string    `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

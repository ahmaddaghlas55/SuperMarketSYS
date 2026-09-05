package models

import "time"

type Category struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	LowStockThreshold int64  `json:"low_stock_threshold"`
	Active            bool   `json:"active"`
}

type Product struct {
	ID                 int64     `json:"id"`
	Barcode            *string   `json:"barcode,omitempty"`
	Name               string    `json:"name"`
	CategoryID         *int64    `json:"category_id,omitempty"`
	UnitType           string    `json:"unit_type"`
	SalePrice          *float64  `json:"sale_price,omitempty"`
	PurchasePrice      *float64  `json:"purchase_price,omitempty"`
	PricePerKg         *float64  `json:"price_per_kg,omitempty"`
	PurchasePricePerKg *float64  `json:"purchase_price_per_kg,omitempty"`
	PricePerPiece      *float64  `json:"price_per_piece,omitempty"`
	PricePerCarton     *float64  `json:"price_per_carton,omitempty"`
	PiecesPerCarton    *int64    `json:"pieces_per_carton,omitempty"`
	Quantity           float64   `json:"quantity"`
	Active             bool      `json:"active"`
	CreatedAt          time.Time `json:"created_at"`
	LowStock           bool      `json:"low_stock"`
}

type Dealer struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Phone       string  `json:"phone,omitempty"`
	Address     string  `json:"address,omitempty"`
	Notes       string  `json:"notes,omitempty"`
	Active      bool    `json:"active"`
	Outstanding float64 `json:"outstanding"`
	Credit      float64 `json:"credit"`
	Balance     float64 `json:"balance"`
}

type PurchaseItem struct {
	ID        int64   `json:"id"`
	ProductID int64   `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	UnitCost  float64 `json:"unit_cost"`
	LineTotal float64 `json:"line_total"`
}

type Purchase struct {
	ID                  int64          `json:"id"`
	DealerID            int64          `json:"dealer_id"`
	UserID              int64          `json:"user_id"`
	DealerInvoiceNumber string         `json:"dealer_invoice_number,omitempty"`
	Subtotal            float64        `json:"subtotal"`
	Discount            float64        `json:"discount"`
	Total               float64        `json:"total"`
	Paid                float64        `json:"paid"`
	Remaining           float64        `json:"remaining"`
	CreatedAt           time.Time      `json:"created_at"`
	Items               []PurchaseItem `json:"items,omitempty"`
}

type PurchasePayment struct {
	ID         int64     `json:"id"`
	PurchaseID int64     `json:"purchase_id"`
	Amount     float64   `json:"amount"`
	Method     string    `json:"method"`
	CreatedAt  time.Time `json:"created_at"`
}

type PurchaseReturnItem struct {
	ID        int64   `json:"id"`
	ProductID int64   `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	UnitCost  float64 `json:"unit_cost"`
	LineTotal float64 `json:"line_total"`
}

type PurchaseReturn struct {
	ID           int64                `json:"id"`
	PurchaseID   int64                `json:"purchase_id"`
	UserID       int64                `json:"user_id"`
	Reason       string               `json:"reason"`
	TotalValue   float64              `json:"total_value"`
	Notes        string               `json:"notes,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	Items        []PurchaseReturnItem `json:"items,omitempty"`
	CreditAmount float64              `json:"credit_amount"`
}

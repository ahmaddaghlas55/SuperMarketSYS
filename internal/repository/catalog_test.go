package repository

import "testing"

func TestLowStockIncludesZeroQuantity(t *testing.T) {
	if !productLowStock(0, 5) {
		t.Fatal("zero stock must be marked low stock")
	}
	if productLowStock(6, 5) {
		t.Fatal("stock above threshold must not be marked low stock")
	}
}

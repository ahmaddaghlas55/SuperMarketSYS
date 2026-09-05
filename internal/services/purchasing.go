package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"supermarket/internal/models"
	"supermarket/internal/repository"
)

type PurchasingService struct {
	repo    *repository.PurchasingRepository
	catalog *repository.CatalogRepository
}

func NewPurchasingService(repo *repository.PurchasingRepository, catalog *repository.CatalogRepository) *PurchasingService {
	return &PurchasingService{repo: repo, catalog: catalog}
}
func (s *PurchasingService) ListDealers(ctx context.Context) ([]models.Dealer, error) {
	return s.repo.ListDealers(ctx)
}
func (s *PurchasingService) CreateDealer(ctx context.Context, d models.Dealer) (models.Dealer, error) {
	if strings.TrimSpace(d.Name) == "" {
		return d, fmt.Errorf("%w: dealer name", ErrValidation)
	}
	d.Name = strings.TrimSpace(d.Name)
	return s.repo.CreateDealer(ctx, d)
}
func (s *PurchasingService) UpdateDealer(ctx context.Context, id int64, d models.Dealer) error {
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("%w: dealer name", ErrValidation)
	}
	d.Name = strings.TrimSpace(d.Name)
	return s.repo.UpdateDealer(ctx, id, d)
}
func (s *PurchasingService) DeleteDealer(ctx context.Context, id int64) error {
	return s.repo.DeactivateDealer(ctx, id)
}
func (s *PurchasingService) ActivateDealer(ctx context.Context, id int64) error {
	return s.repo.ActivateDealer(ctx, id)
}
func (s *PurchasingService) GetPurchase(ctx context.Context, id int64) (models.Purchase, error) {
	return s.repo.GetPurchase(ctx, id)
}
func (s *PurchasingService) ListPurchases(ctx context.Context, dealerID int64) ([]models.Purchase, error) {
	return s.repo.ListPurchases(ctx, dealerID)
}
func (s *PurchasingService) ListReturns(ctx context.Context, purchaseID int64) ([]models.PurchaseReturn, error) {
	return s.repo.ListReturns(ctx, purchaseID)
}

func (s *PurchasingService) CreatePurchase(ctx context.Context, userID int64, p models.Purchase) (models.Purchase, error) {
	if p.DealerID <= 0 || len(p.Items) == 0 {
		return p, fmt.Errorf("%w: dealer and items required", ErrValidation)
	}
	if p.Discount < 0 {
		return p, fmt.Errorf("%w: discount", ErrValidation)
	}
	var subtotal float64
	for i := range p.Items {
		if p.Items[i].ProductID <= 0 || p.Items[i].Quantity <= 0 || p.Items[i].UnitCost < 0 {
			return p, fmt.Errorf("%w: invalid purchase item", ErrValidation)
		}
		p.Items[i].LineTotal = roundMoney(p.Items[i].Quantity * p.Items[i].UnitCost)
		subtotal += p.Items[i].LineTotal
	}
	p.Subtotal = roundMoney(subtotal)
	p.Total = roundMoney(p.Subtotal - p.Discount)
	if p.Total < 0 {
		return p, fmt.Errorf("%w: discount exceeds subtotal", ErrValidation)
	}
	p.UserID = userID
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return p, err
	}
	defer tx.Rollback()
	id, err := s.repo.InsertPurchaseTx(ctx, tx, p)
	if err != nil {
		return p, err
	}
	for _, item := range p.Items {
		if _, _, _, err = s.repo.ProductForUpdate(ctx, tx, item.ProductID); err != nil {
			return p, err
		}
		if err = s.repo.InsertPurchaseItemTx(ctx, tx, id, item); err != nil {
			return p, err
		}
		if _, err = s.repo.ChangeStockTx(ctx, tx, item.ProductID, item.Quantity, "purchase", id, userID, fmt.Sprintf("%.6f", item.UnitCost)); err != nil {
			return p, err
		}
		if err = s.repo.SetProductCostTx(ctx, tx, item.ProductID, item.UnitCost); err != nil {
			return p, err
		}
	}
	applied, err := s.repo.ApplyDealerCreditsTx(ctx, tx, p.DealerID, id, p.Total)
	if err != nil {
		return p, err
	}
	if err = tx.Commit(); err != nil {
		return p, err
	}
	result, err := s.repo.GetPurchase(ctx, id)
	if err == nil {
		result.AppliedCredit = applied
	}
	return result, err
}

func (s *PurchasingService) PayPurchase(ctx context.Context, userID, purchaseID int64, amount float64, method string) (models.Purchase, error) {
	if amount <= 0 || method != "cash" && method != "card" && method != "transfer" {
		return models.Purchase{}, fmt.Errorf("%w: invalid payment", ErrValidation)
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return models.Purchase{}, err
	}
	defer tx.Rollback()
	p, err := s.repo.UpdatePurchasePaymentTx(ctx, tx, purchaseID, amount)
	if errors.Is(err, repository.ErrNotFound) {
		return p, err
	}
	if err != nil {
		return p, err
	}
	if p.Paid > p.Total+0.000001 {
		return p, fmt.Errorf("%w: payment exceeds remaining", ErrValidation)
	}
	var shiftID *int64
	if method == "cash" {
		var id int64
		if err = tx.QueryRowContext(ctx, `SELECT id FROM cash_shifts WHERE closed_at IS NULL ORDER BY opened_at,id LIMIT 1`).Scan(&id); err != nil {
			return p, fmt.Errorf("%w: open shift required", ErrValidation)
		}
		shiftID = &id
	}
	if err = s.repo.InsertPaymentTx(ctx, tx, purchaseID, amount, method, shiftID); err != nil {
		return p, err
	}
	if method == "cash" {
		ref := purchaseID
		if _, err = tx.ExecContext(ctx, `INSERT INTO cash_movements(shift_id,type,direction,amount,reference_type,reference_id) VALUES (?,?,?,?,?,?)`, *shiftID, "dealer_payment", "out", amount, "purchase", ref); err != nil {
			return p, err
		}
	}
	if err = tx.Commit(); err != nil {
		return p, err
	}
	return s.repo.GetPurchase(ctx, purchaseID)
}

func (s *PurchasingService) ReturnPurchase(ctx context.Context, userID int64, r models.PurchaseReturn) (models.PurchaseReturn, error) {
	if r.PurchaseID <= 0 || len(r.Items) == 0 || r.Reason != "expired" && r.Reason != "damaged" && r.Reason != "unsold" {
		return r, fmt.Errorf("%w: invalid return", ErrValidation)
	}
	r.UserID = userID
	for i := range r.Items {
		if r.Items[i].Quantity <= 0 || r.Items[i].ProductID <= 0 {
			return r, fmt.Errorf("%w: invalid return item", ErrValidation)
		}
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return r, err
	}
	defer tx.Rollback()
	if _, _, err = s.repo.PurchaseBalanceTx(ctx, tx, r.PurchaseID); err != nil {
		return r, err
	}
	var total float64
	for i := range r.Items {
		purchased, cost, e := s.repo.PurchaseItemQuantityTx(ctx, tx, r.PurchaseID, r.Items[i].ProductID)
		if e != nil {
			return r, e
		}
		returned, e := s.repo.ReturnedQuantityTx(ctx, tx, r.PurchaseID, r.Items[i].ProductID)
		if e != nil {
			return r, e
		}
		if r.Items[i].Quantity > purchased-returned+0.000001 {
			return r, fmt.Errorf("%w: return exceeds purchased quantity", ErrValidation)
		}
		r.Items[i].UnitCost = cost
		r.Items[i].LineTotal = roundMoney(cost * r.Items[i].Quantity)
		total += r.Items[i].LineTotal
	}
	r.TotalValue = roundMoney(total)
	id, err := s.repo.InsertReturnTx(ctx, tx, r)
	if err != nil {
		return r, err
	}
	for _, item := range r.Items {
		if err = s.repo.InsertReturnItemTx(ctx, tx, id, item); err != nil {
			return r, err
		}
		if _, err = s.repo.ChangeStockTx(ctx, tx, item.ProductID, -item.Quantity, "return_purchase", id, userID, r.Reason); err != nil {
			return r, err
		}
	}
	dealerID, credit, err := s.repo.ApplyPurchaseReturnTx(ctx, tx, r.PurchaseID, r.TotalValue)
	if err != nil {
		return r, err
	}
	if err = s.repo.InsertDealerCreditTx(ctx, tx, dealerID, id, credit); err != nil {
		return r, err
	}
	if err = tx.Commit(); err != nil {
		return r, err
	}
	r.ID = id
	r.CreditAmount = credit
	r.CreatedAt = time.Now().UTC()
	return r, nil
}

func roundMoney(v float64) float64 { return float64(int64(v*100+0.5)) / 100 }

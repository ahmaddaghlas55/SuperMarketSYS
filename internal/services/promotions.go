package services

import (
	"context"
	"fmt"
	"time"

	"supermarket/internal/models"
	"supermarket/internal/repository"
)

type PromotionService struct {
	repo *repository.PromotionRepository
}

func NewPromotionService(r *repository.PromotionRepository) *PromotionService {
	return &PromotionService{repo: r}
}

func (s *PromotionService) Create(ctx context.Context, p models.Promotion) (models.Promotion, error) {
	if err := repository.ValidatePromotion(p); err != nil {
		return p, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	return s.repo.Create(ctx, p)
}

func (s *PromotionService) Get(ctx context.Context, id int64) (models.Promotion, error) {
	return s.repo.Get(ctx, id)
}

func (s *PromotionService) List(ctx context.Context, productID int64, activeOnly bool) ([]models.Promotion, error) {
	return s.repo.List(ctx, productID, activeOnly)
}

func (s *PromotionService) Update(ctx context.Context, id int64, p models.Promotion) error {
	if err := repository.ValidatePromotion(p); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	return s.repo.Update(ctx, id, p)
}

func (s *PromotionService) Delete(ctx context.Context, id int64) error {
	return s.repo.Deactivate(ctx, id)
}

func (s *PromotionService) Activate(ctx context.Context, id int64) error {
	return s.repo.Activate(ctx, id)
}

func PromotionPrice(p models.Promotion, base float64, quantity float64) float64 {
	switch p.PromoType {
	case "percent_off":
		return base * (1 - p.Value/100)
	case "fixed_price":
		return p.Value
	case "bundle":
		if p.BundleQuantity == nil || *p.BundleQuantity <= 1 || quantity < float64(*p.BundleQuantity) {
			return base
		}
		n := float64(*p.BundleQuantity)
		full := float64(int(quantity / n))
		remainder := quantity - full*n
		return (full*p.Value + remainder*base) / quantity
	default:
		return base
	}
}

func PromotionActiveAt(p models.Promotion, now time.Time, stock float64) bool {
	return p.Active && (p.StartsAt == nil || !now.Before(*p.StartsAt)) &&
		(p.EndsAt == nil || !now.After(*p.EndsAt)) && (!p.UntilStockZero || stock > 0)
}

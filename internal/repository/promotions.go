package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"supermarket/internal/models"
)

type PromotionRepository struct{ db *sql.DB }

func NewPromotionRepository(db *sql.DB) *PromotionRepository { return &PromotionRepository{db: db} }

func promotionTime(v sql.NullString) *time.Time {
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

func scanPromotion(s interface{ Scan(...any) error }) (models.Promotion, error) {
	var p models.Promotion
	var starts, ends, created sql.NullString
	var bundle sql.NullInt64
	var active, until int
	err := s.Scan(&p.ID, &p.ProductID, &p.PromoType, &p.Value, &bundle, &starts, &ends, &until, &active, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	if bundle.Valid {
		p.BundleQuantity = &bundle.Int64
	}
	p.StartsAt, p.EndsAt = promotionTime(starts), promotionTime(ends)
	p.UntilStockZero, p.Active = until == 1, active == 1
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created.String)
	return p, nil
}

func (r *PromotionRepository) Create(ctx context.Context, p models.Promotion) (models.Promotion, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO promotions(product_id,promo_type,value,bundle_quantity,starts_at,ends_at,until_stock_zero) VALUES(?,?,?,?,?,?,?)`,
		p.ProductID, p.PromoType, p.Value, p.BundleQuantity, formatPromotionTime(p.StartsAt), formatPromotionTime(p.EndsAt), boolInt(p.UntilStockZero))
	if err != nil {
		return p, err
	}
	p.ID, err = res.LastInsertId()
	p.Active = true
	p.CreatedAt = time.Now().UTC()
	return p, err
}

func (r *PromotionRepository) Get(ctx context.Context, id int64) (models.Promotion, error) {
	return scanPromotion(r.db.QueryRowContext(ctx, `SELECT id,product_id,promo_type,value,bundle_quantity,starts_at,ends_at,until_stock_zero,active,created_at FROM promotions WHERE id=?`, id))
}

func (r *PromotionRepository) List(ctx context.Context, productID int64, activeOnly bool) ([]models.Promotion, error) {
	q := `SELECT id,product_id,promo_type,value,bundle_quantity,starts_at,ends_at,until_stock_zero,active,created_at FROM promotions WHERE 1=1`
	args := []any{}
	if productID > 0 {
		q += ` AND product_id=?`
		args = append(args, productID)
	}
	if activeOnly {
		q += ` AND active=1 AND (starts_at IS NULL OR datetime(starts_at)<=datetime('now')) AND (ends_at IS NULL OR datetime(ends_at)>=datetime('now')) AND (until_stock_zero=0 OR EXISTS (SELECT 1 FROM products p WHERE p.id=promotions.product_id AND p.active=1 AND p.quantity>0))`
	}
	q += ` ORDER BY created_at DESC,id DESC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Promotion
	for rows.Next() {
		p, err := scanPromotion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func formatPromotionTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func (r *PromotionRepository) Update(ctx context.Context, id int64, p models.Promotion) error {
	res, err := r.db.ExecContext(ctx, `UPDATE promotions SET product_id=?,promo_type=?,value=?,bundle_quantity=?,starts_at=?,ends_at=?,until_stock_zero=? WHERE id=? AND active=1`,
		p.ProductID, p.PromoType, p.Value, p.BundleQuantity, formatPromotionTime(p.StartsAt), formatPromotionTime(p.EndsAt), boolInt(p.UntilStockZero), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PromotionRepository) Deactivate(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE promotions SET active=0 WHERE id=? AND active=1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PromotionRepository) Activate(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE promotions SET active=1 WHERE id=? AND active=0`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists int
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM promotions WHERE id=?`, id).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func (r *PromotionRepository) ActiveForProductTx(ctx context.Context, tx *sql.Tx, productID int64, now time.Time, stock float64) (models.Promotion, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,product_id,promo_type,value,bundle_quantity,starts_at,ends_at,until_stock_zero,active,created_at
		FROM promotions WHERE product_id=? AND active=1 ORDER BY created_at DESC,id DESC`, productID)
	if err != nil {
		return models.Promotion{}, err
	}
	defer rows.Close()
	for rows.Next() {
		p, err := scanPromotion(rows)
		if err != nil {
			return models.Promotion{}, err
		}
		if p.StartsAt != nil && now.Before(*p.StartsAt) {
			continue
		}
		if p.EndsAt != nil && now.After(*p.EndsAt) {
			continue
		}
		if p.UntilStockZero && stock <= 0 {
			continue
		}
		return p, nil
	}
	if err := rows.Err(); err != nil {
		return models.Promotion{}, err
	}
	return models.Promotion{}, ErrNotFound
}

func ValidatePromotion(p models.Promotion) error {
	if p.ProductID <= 0 || p.Value < 0 {
		return fmt.Errorf("invalid promotion")
	}
	switch p.PromoType {
	case "percent_off":
		if p.Value > 100 {
			return fmt.Errorf("percent discount exceeds 100")
		}
	case "fixed_price":
	case "bundle":
		if p.BundleQuantity == nil || *p.BundleQuantity < 2 || p.Value < 0 {
			return fmt.Errorf("invalid bundle promotion")
		}
	default:
		return fmt.Errorf("invalid promotion type")
	}
	if p.StartsAt != nil && p.EndsAt != nil && p.EndsAt.Before(*p.StartsAt) {
		return fmt.Errorf("promotion end precedes start")
	}
	return nil
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"supermarket/internal/models"
)

type CatalogRepository struct{ db *sql.DB }

func NewCatalogRepository(db *sql.DB) *CatalogRepository { return &CatalogRepository{db: db} }

func (r *CatalogRepository) CreateCategory(ctx context.Context, c models.Category) (models.Category, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO categories(name, low_stock_threshold) VALUES (?, ?)`, c.Name, c.LowStockThreshold)
	if err != nil {
		return c, fmt.Errorf("create category: %w", err)
	}
	c.ID, err = res.LastInsertId()
	c.Active = true
	return c, err
}

func (r *CatalogRepository) ListCategories(ctx context.Context) ([]models.Category, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,low_stock_threshold,active FROM categories WHERE active=1 ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Category
	for rows.Next() {
		var c models.Category
		var active int
		if err := rows.Scan(&c.ID, &c.Name, &c.LowStockThreshold, &active); err != nil {
			return nil, err
		}
		c.Active = active == 1
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CatalogRepository) UpdateCategory(ctx context.Context, id int64, c models.Category) error {
	res, err := r.db.ExecContext(ctx, `UPDATE categories SET name=?, low_stock_threshold=? WHERE id=? AND active=1`, c.Name, c.LowStockThreshold, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CatalogRepository) DeactivateCategory(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE categories SET active=0 WHERE id=? AND active=1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CatalogRepository) ActivateCategory(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE categories SET active=1 WHERE id=? AND active=0`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists int
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories WHERE id=?`, id).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func (r *CatalogRepository) getProduct(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id int64) (models.Product, error) {
	var p models.Product
	var barcode, created sql.NullString
	var category sql.NullInt64
	var sale, purchase, ppkg, cppkg, ppiece, pcarton sql.NullFloat64
	var pieces sql.NullInt64
	var active int
	err := q.QueryRowContext(ctx, `SELECT p.id,p.barcode,p.name,p.category_id,p.unit_type,p.sale_price,p.purchase_price,p.price_per_kg,p.purchase_price_per_kg,p.price_per_piece,p.price_per_carton,p.pieces_per_carton,p.quantity,p.active,p.created_at FROM products p WHERE p.id=? AND p.active=1`, id).
		Scan(&p.ID, &barcode, &p.Name, &category, &p.UnitType, &sale, &purchase, &ppkg, &cppkg, &ppiece, &pcarton, &pieces, &p.Quantity, &active, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
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
	p.Active = active == 1
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created.String)
	return p, nil
}

func (r *CatalogRepository) GetProduct(ctx context.Context, id int64) (models.Product, error) {
	return r.getProduct(ctx, r.db, id)
}

func (r *CatalogRepository) ListProducts(ctx context.Context, lowStock bool) ([]models.Product, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT p.id,p.barcode,p.name,p.category_id,p.unit_type,p.sale_price,p.purchase_price,p.price_per_kg,p.purchase_price_per_kg,p.price_per_piece,p.price_per_carton,p.pieces_per_carton,p.quantity,p.active,p.created_at,COALESCE(c.low_stock_threshold,0) FROM products p LEFT JOIN categories c ON c.id=p.category_id WHERE p.active=1 AND (?=0 OR p.quantity <= COALESCE(c.low_stock_threshold,0)) ORDER BY p.name`, boolToInt(lowStock))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Product
	for rows.Next() {
		var p models.Product
		var barcode, created sql.NullString
		var category sql.NullInt64
		var sale, purchase, ppkg, cppkg, ppiece, pcarton sql.NullFloat64
		var pieces sql.NullInt64
		var active int
		var threshold int64
		if err := rows.Scan(&p.ID, &barcode, &p.Name, &category, &p.UnitType, &sale, &purchase, &ppkg, &cppkg, &ppiece, &pcarton, &pieces, &p.Quantity, &active, &created, &threshold); err != nil {
			return nil, err
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
		p.Active = active == 1
		p.LowStock = p.Quantity <= float64(threshold)
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created.String)
		out = append(out, p)
	}
	return out, rows.Err()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (r *CatalogRepository) SaveProduct(ctx context.Context, p models.Product) (models.Product, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return p, err
	}
	defer tx.Rollback()
	p.ID, err = r.insertProductTx(ctx, tx, p)
	if err != nil {
		return p, err
	}
	if err = tx.Commit(); err != nil {
		return p, err
	}
	return r.GetProduct(ctx, p.ID)
}

func (r *CatalogRepository) SaveProductWithUser(ctx context.Context, p models.Product, userID int64) (models.Product, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return p, err
	}
	defer tx.Rollback()
	p.ID, err = r.insertProductTx(ctx, tx, p)
	if err != nil {
		return p, err
	}
	if p.Quantity != 0 {
		if _, err = tx.ExecContext(ctx, `INSERT INTO stock_movements(product_id,type,quantity_change,balance_after,reference_type,reference_id,user_id,notes) VALUES (?,?,?,?,?,?,?,?)`, p.ID, "adjustment", p.Quantity, p.Quantity, "product", p.ID, userID, "initial stock"); err != nil {
			return p, err
		}
	}
	if err = tx.Commit(); err != nil {
		return p, err
	}
	return r.GetProduct(ctx, p.ID)
}

func (r *CatalogRepository) insertProductTx(ctx context.Context, tx *sql.Tx, p models.Product) (int64, error) {
	barcode := nullableString(p.Barcode)
	if p.Barcode == nil || *p.Barcode == "" {
		allocated, err := r.allocateInternalBarcodeTx(ctx, tx)
		if err != nil {
			return 0, err
		}
		barcode = allocated
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO products(barcode,name,category_id,unit_type,sale_price,purchase_price,price_per_kg,purchase_price_per_kg,price_per_piece,price_per_carton,pieces_per_carton,quantity) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, barcode, p.Name, p.CategoryID, p.UnitType, p.SalePrice, p.PurchasePrice, p.PricePerKg, p.PurchasePricePerKg, p.PricePerPiece, p.PricePerCarton, p.PiecesPerCarton, p.Quantity)
	if err != nil {
		return 0, fmt.Errorf("create product: %w", err)
	}
	return res.LastInsertId()
}

func (r *CatalogRepository) InsertProductTx(ctx context.Context, tx *sql.Tx, p models.Product) (int64, error) {
	return r.insertProductTx(ctx, tx, p)
}

func (r *CatalogRepository) allocateInternalBarcodeTx(ctx context.Context, tx *sql.Tx) (string, error) {
	for attempt := 0; attempt < 100; attempt++ {
		if _, err := tx.ExecContext(ctx, `UPDATE product_barcode_sequences SET next_value=next_value+1 WHERE id=1`); err != nil {
			return "", fmt.Errorf("advance barcode sequence: %w", err)
		}
		var value int64
		if err := tx.QueryRowContext(ctx, `SELECT next_value-1 FROM product_barcode_sequences WHERE id=1`).Scan(&value); err != nil {
			return "", fmt.Errorf("read barcode sequence: %w", err)
		}
		barcode := "20" + fmt.Sprintf("%011d", value)
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM products WHERE barcode=?`, barcode).Scan(&exists); err != nil {
			return "", fmt.Errorf("check barcode: %w", err)
		}
		if exists == 0 {
			return barcode, nil
		}
	}
	return "", fmt.Errorf("allocate internal barcode: exhausted retries")
}

func (r *CatalogRepository) UpdateProduct(ctx context.Context, id int64, p models.Product) (models.Product, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE products SET barcode=COALESCE(?,barcode),name=?,category_id=?,unit_type=?,sale_price=?,purchase_price=?,price_per_kg=?,purchase_price_per_kg=?,price_per_piece=?,price_per_carton=?,pieces_per_carton=? WHERE id=? AND active=1`, nullableString(p.Barcode), p.Name, p.CategoryID, p.UnitType, p.SalePrice, p.PurchasePrice, p.PricePerKg, p.PurchasePricePerKg, p.PricePerPiece, p.PricePerCarton, p.PiecesPerCarton, id)
	if err != nil {
		return p, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return p, ErrNotFound
	}
	return r.GetProduct(ctx, id)
}

func (r *CatalogRepository) DeactivateProduct(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE products SET active=0 WHERE id=? AND active=1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CatalogRepository) ActivateProduct(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE products SET active=1 WHERE id=? AND active=0`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists int
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM products WHERE id=?`, id).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func nullableString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

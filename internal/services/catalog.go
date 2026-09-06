package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"supermarket/internal/models"
	"supermarket/internal/repository"
)

var ErrValidation = errors.New("validation failed")

type CatalogService struct {
	repo   *repository.CatalogRepository
	dbRepo *repository.PurchasingRepository
	promo  *repository.PromotionRepository
}

func NewCatalogService(repo *repository.CatalogRepository, dbRepo *repository.PurchasingRepository, promos ...*repository.PromotionRepository) *CatalogService {
	var promo *repository.PromotionRepository
	if len(promos) > 0 {
		promo = promos[0]
	}
	return &CatalogService{repo: repo, dbRepo: dbRepo, promo: promo}
}

func validateCategory(c models.Category) error {
	if strings.TrimSpace(c.Name) == "" || c.LowStockThreshold < 0 {
		return fmt.Errorf("%w: invalid category", ErrValidation)
	}
	return nil
}
func (s *CatalogService) CreateCategory(ctx context.Context, c models.Category) (models.Category, error) {
	if err := validateCategory(c); err != nil {
		return c, err
	}
	c.Name = strings.TrimSpace(c.Name)
	return s.repo.CreateCategory(ctx, c)
}
func (s *CatalogService) ListCategories(ctx context.Context) ([]models.Category, error) {
	return s.repo.ListCategories(ctx)
}
func (s *CatalogService) UpdateCategory(ctx context.Context, id int64, c models.Category) error {
	if err := validateCategory(c); err != nil {
		return err
	}
	c.Name = strings.TrimSpace(c.Name)
	return s.repo.UpdateCategory(ctx, id, c)
}
func (s *CatalogService) DeleteCategory(ctx context.Context, id int64) error {
	return s.repo.DeactivateCategory(ctx, id)
}
func (s *CatalogService) ActivateCategory(ctx context.Context, id int64) error {
	return s.repo.ActivateCategory(ctx, id)
}

func ValidateProduct(p models.Product) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	switch p.UnitType {
	case "piece":
		if p.SalePrice == nil || *p.SalePrice < 0 {
			return fmt.Errorf("%w: sale_price required", ErrValidation)
		}
	case "weight":
		if p.PricePerKg == nil || *p.PricePerKg < 0 {
			return fmt.Errorf("%w: price_per_kg required", ErrValidation)
		}
	case "carton":
		if p.PricePerPiece == nil || p.PricePerCarton == nil || p.PiecesPerCarton == nil || *p.PricePerPiece < 0 || *p.PricePerCarton < 0 || *p.PiecesPerCarton <= 0 {
			return fmt.Errorf("%w: invalid carton pricing", ErrValidation)
		}
	default:
		return fmt.Errorf("%w: invalid unit_type", ErrValidation)
	}
	if p.Quantity < 0 {
		return fmt.Errorf("%w: negative quantity", ErrValidation)
	}
	return nil
}
func (s *CatalogService) CreateProduct(ctx context.Context, p models.Product) (models.Product, error) {
	if err := ValidateProduct(p); err != nil {
		return p, err
	}
	if p.Quantity != 0 {
		return p, fmt.Errorf("%w: initial stock requires a user context", ErrValidation)
	}
	p.Name = strings.TrimSpace(p.Name)
	normalizeBarcode(&p)
	return s.repo.SaveProduct(ctx, p)
}
func (s *CatalogService) CreateProductAs(ctx context.Context, userID int64, p models.Product) (models.Product, error) {
	if err := ValidateProduct(p); err != nil {
		return p, err
	}
	p.Name = strings.TrimSpace(p.Name)
	normalizeBarcode(&p)
	return s.repo.SaveProductWithUser(ctx, p, userID)
}
func (s *CatalogService) GetProduct(ctx context.Context, id int64) (models.Product, error) {
	return s.repo.GetProduct(ctx, id)
}
func (s *CatalogService) ListProducts(ctx context.Context, low bool) ([]models.Product, error) {
	return s.repo.ListProducts(ctx, low)
}

func (s *CatalogService) CalculatePrice(ctx context.Context, id int64, quantity float64) (float64, error) {
	p, err := s.repo.GetProduct(ctx, id)
	if err != nil {
		return 0, err
	}
	unit, err := CalculateUnitPrice(p, quantity)
	if err != nil {
		return 0, err
	}
	if s.promo != nil {
		promotions, listErr := s.promo.List(ctx, id, true)
		if listErr != nil {
			return 0, listErr
		}
		if len(promotions) > 0 {
			unit = PromotionPrice(promotions[0], unit, quantity)
		}
	}
	return unit, nil
}
func (s *CatalogService) UpdateProduct(ctx context.Context, id int64, p models.Product) (models.Product, error) {
	if err := ValidateProduct(p); err != nil {
		return p, err
	}
	p.Name = strings.TrimSpace(p.Name)
	normalizeBarcode(&p)
	return s.repo.UpdateProduct(ctx, id, p)
}
func (s *CatalogService) DeleteProduct(ctx context.Context, id int64) error {
	return s.repo.DeactivateProduct(ctx, id)
}
func (s *CatalogService) ActivateProduct(ctx context.Context, id int64) error {
	return s.repo.ActivateProduct(ctx, id)
}

func (s *CatalogService) ImportProducts(ctx context.Context, reader io.Reader) (int, error) {
	return s.ImportProductsAs(ctx, reader, 0)
}

func (s *CatalogService) ImportProductsAs(ctx context.Context, reader io.Reader, userID int64) (int, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return 0, fmt.Errorf("open workbook: %w", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return 0, err
	}
	if len(rows) < 2 {
		return 0, fmt.Errorf("%w: spreadsheet is empty", ErrValidation)
	}
	headers := map[string]int{}
	for i, h := range rows[0] {
		headers[strings.ToLower(strings.TrimSpace(h))] = i
	}
	required := []string{"name", "unit_type"}
	for _, h := range required {
		if _, ok := headers[h]; !ok {
			return 0, fmt.Errorf("%w: missing %s", ErrValidation, h)
		}
	}
	tx, err := s.dbRepo.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	count := 0
	for n, row := range rows[1:] {
		val := func(key string) string {
			if i, ok := headers[key]; ok && i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}
		p := models.Product{Name: val("name"), UnitType: strings.ToLower(val("unit_type"))}
		if v := val("barcode"); v != "" {
			p.Barcode = &v
		}
		if v := val("category_id"); v != "" {
			id, e := strconv.ParseInt(v, 10, 64)
			if e != nil {
				return count, fmt.Errorf("row %d: category_id: %w", n+2, e)
			}
			p.CategoryID = &id
		} else if categoryName := val("category"); categoryName != "" {
			var id int64
			err := tx.QueryRowContext(ctx, `SELECT id FROM categories WHERE name=? AND active=1`, categoryName).Scan(&id)
			if err != nil {
				res, createErr := tx.ExecContext(ctx, `INSERT INTO categories(name,low_stock_threshold) VALUES (?,5)`, categoryName)
				if createErr != nil {
					return count, fmt.Errorf("row %d: category: %w", n+2, createErr)
				}
				id, err = res.LastInsertId()
			}
			if err != nil {
				return count, fmt.Errorf("row %d: category: %w", n+2, err)
			}
			p.CategoryID = &id
		}
		if v := val("sale_price"); v != "" {
			x, e := strconv.ParseFloat(v, 64)
			if e != nil {
				return count, fmt.Errorf("row %d: sale_price", n+2)
			}
			p.SalePrice = &x
		}
		if v := val("purchase_price"); v != "" {
			x, e := strconv.ParseFloat(v, 64)
			if e != nil {
				return count, fmt.Errorf("row %d: purchase_price", n+2)
			}
			p.PurchasePrice = &x
		}
		if v := val("purchase_price_per_kg"); v != "" {
			x, e := strconv.ParseFloat(v, 64)
			if e != nil {
				return count, fmt.Errorf("row %d: purchase_price_per_kg", n+2)
			}
			p.PurchasePricePerKg = &x
		}
		if v := val("price_per_kg"); v != "" {
			x, e := strconv.ParseFloat(v, 64)
			if e != nil {
				return count, fmt.Errorf("row %d: price_per_kg", n+2)
			}
			p.PricePerKg = &x
		}
		if v := val("price_per_piece"); v != "" {
			x, e := strconv.ParseFloat(v, 64)
			if e != nil {
				return count, fmt.Errorf("row %d: price_per_piece", n+2)
			}
			p.PricePerPiece = &x
		}
		if v := val("price_per_carton"); v != "" {
			x, e := strconv.ParseFloat(v, 64)
			if e != nil {
				return count, fmt.Errorf("row %d: price_per_carton", n+2)
			}
			p.PricePerCarton = &x
		}
		if v := val("pieces_per_carton"); v != "" {
			x, e := strconv.ParseInt(v, 10, 64)
			if e != nil {
				return count, fmt.Errorf("row %d: pieces_per_carton", n+2)
			}
			p.PiecesPerCarton = &x
		}
		if v := val("quantity"); v != "" {
			x, e := strconv.ParseFloat(v, 64)
			if e != nil {
				return count, fmt.Errorf("row %d: quantity", n+2)
			}
			p.Quantity = x
		}
		if err := ValidateProduct(p); err != nil {
			return count, fmt.Errorf("row %d: %w", n+2, err)
		}
		productID, err := s.repo.InsertProductTx(ctx, tx, p)
		if err != nil {
			return count, fmt.Errorf("row %d: %w", n+2, err)
		}
		if p.Quantity != 0 {
			if userID == 0 {
				return count, fmt.Errorf("row %d: initial quantity requires an authenticated user", n+2)
			}
			if _, movementErr := tx.ExecContext(ctx, `INSERT INTO stock_movements(product_id,type,quantity_change,balance_after,reference_type,reference_id,user_id,notes) VALUES (?,?,?,?,?,?,?,?)`, productID, "adjustment", p.Quantity, p.Quantity, "product", productID, userID, "initial stock"); movementErr != nil {
				return count, movementErr
			}
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		return count, err
	}
	return count, nil
}

func normalizeBarcode(p *models.Product) {
	if p.Barcode == nil {
		return
	}
	value := strings.TrimSpace(*p.Barcode)
	if value == "" {
		p.Barcode = nil
		return
	}
	p.Barcode = &value
}

func CalculateUnitPrice(p models.Product, quantity float64) (float64, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("%w: quantity must be positive", ErrValidation)
	}
	switch p.UnitType {
	case "piece":
		if p.SalePrice == nil {
			return 0, ErrValidation
		}
		return *p.SalePrice, nil
	case "weight":
		if p.PricePerKg == nil {
			return 0, ErrValidation
		}
		return *p.PricePerKg, nil
	case "carton":
		if p.PricePerPiece == nil || p.PricePerCarton == nil || p.PiecesPerCarton == nil {
			return 0, ErrValidation
		}
		if float64(*p.PiecesPerCarton) > 0 && float64(int(quantity)) >= float64(*p.PiecesPerCarton) && int(quantity)%int(*p.PiecesPerCarton) == 0 {
			return *p.PricePerCarton / float64(*p.PiecesPerCarton), nil
		}
		return *p.PricePerPiece, nil
	default:
		return 0, ErrValidation
	}
}

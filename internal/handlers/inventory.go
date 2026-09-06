package handlers

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"net/http"
	"strconv"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/jung-kurt/gofpdf"
	"supermarket/internal/middleware"
	"supermarket/internal/models"
	"supermarket/internal/repository"
	"supermarket/internal/services"
)

type CatalogHandler struct{ service *services.CatalogService }

func NewCatalogHandler(s *services.CatalogService) *CatalogHandler {
	return &CatalogHandler{service: s}
}

func idParam(r *http.Request) (int64, error) { return strconv.ParseInt(r.PathValue("id"), 10, 64) }
func serviceError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		middleware.WriteError(w, http.StatusNotFound, "not_found")
		return
	}
	if errors.Is(err, services.ErrValidation) {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "unique constraint") {
		middleware.WriteError(w, http.StatusConflict, "conflict")
		return
	}
	if strings.Contains(message, "constraint failed") || strings.Contains(message, "foreign key") {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	middleware.WriteError(w, http.StatusInternalServerError, "internal_error")
}

func (h *CatalogHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.ListCategories(r.Context())
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": v})
}
func (h *CatalogHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var v models.Category
	if !decodeJSON(w, r, &v) {
		return
	}
	v, e := h.service.CreateCategory(r.Context(), v)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}
func (h *CatalogHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	var v models.Category
	if !decodeJSON(w, r, &v) {
		return
	}
	if e = h.service.UpdateCategory(r.Context(), id, v); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
func (h *CatalogHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	if e = h.service.DeleteCategory(r.Context(), id); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
func (h *CatalogHandler) ActivateCategory(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	if e = h.service.ActivateCategory(r.Context(), id); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "active": true})
}
func (h *CatalogHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.ListProducts(r.Context(), r.URL.Query().Get("low_stock") == "true")
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"products": v})
}
func (h *CatalogHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	v, e := h.service.GetProduct(r.Context(), id)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, v)
}

func (h *CatalogHandler) CalculatePrice(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	qty, err := strconv.ParseFloat(r.URL.Query().Get("quantity"), 64)
	if err != nil {
		middleware.WriteError(w, 400, "invalid_quantity")
		return
	}
	unitPrice, err := h.service.CalculatePrice(r.Context(), id, qty)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"product_id": id, "quantity": qty, "unit_price": unitPrice, "line_total": unitPrice * qty})
}
func (h *CatalogHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var v models.Product
	if !decodeJSON(w, r, &v) {
		return
	}
	generated := v.Barcode == nil || *v.Barcode == ""
	v, e := h.service.CreateProductAs(r.Context(), currentUserID(r), v)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 201, map[string]any{"product": v, "barcode_generated": generated})
}

func (h *CatalogHandler) BarcodeLabel(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	p, e := h.service.GetProduct(r.Context(), id)
	if e != nil {
		serviceError(w, e)
		return
	}
	if p.Barcode == nil || *p.Barcode == "" {
		middleware.WriteError(w, 400, "barcode_missing")
		return
	}
	code, e := code128.Encode(*p.Barcode)
	if e != nil {
		serviceError(w, e)
		return
	}
	img, e := barcode.Scale(code, 360, 100)
	if e != nil {
		serviceError(w, e)
		return
	}
	var imageData bytes.Buffer
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	if e = png.Encode(&imageData, rgba); e != nil {
		serviceError(w, e)
		return
	}
	pdf := gofpdf.New("P", "mm", "A6", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(95, 10, p.Name, "", 1, "C", false, 0, "")
	pdf.RegisterImageOptionsReader("barcode", gofpdf.ImageOptions{ImageType: "PNG"}, &imageData)
	pdf.ImageOptions("barcode", 10, 25, 128, 36, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	pdf.SetXY(10, 64)
	pdf.CellFormat(128, 8, *p.Barcode, "", 0, "C", false, 0, "")
	var out bytes.Buffer
	if e = pdf.Output(&out); e != nil {
		serviceError(w, e)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="barcode-`+*p.Barcode+`.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out.Bytes())
}
func (h *CatalogHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	var v models.Product
	if !decodeJSON(w, r, &v) {
		return
	}
	v, e = h.service.UpdateProduct(r.Context(), id, v)
	if e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (h *CatalogHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	if e = h.service.DeleteProduct(r.Context(), id); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
func (h *CatalogHandler) ActivateProduct(w http.ResponseWriter, r *http.Request) {
	id, e := idParam(r)
	if e != nil {
		middleware.WriteError(w, 400, "invalid_id")
		return
	}
	if e = h.service.ActivateProduct(r.Context(), id); e != nil {
		serviceError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "active": true})
}
func (h *CatalogHandler) ImportProducts(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		middleware.WriteError(w, 400, "invalid_multipart")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		middleware.WriteError(w, 400, "file_required")
		return
	}
	defer file.Close()
	n, err := h.service.ImportProductsAs(r.Context(), file, currentUserID(r))
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"imported": n})
}

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
	"supermarket/internal/handlers"
	"supermarket/internal/middleware"
	"supermarket/internal/migrations"
	"supermarket/internal/repository"
	"supermarket/internal/services"
)

func main() {
	address := envOr("SERVER_ADDRESS", ":8080")
	databasePath := envOr("DATABASE_PATH", "supermarket.db")
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		log.Fatalf("enable foreign keys: %v", err)
	}
	if err := migrations.Run(ctx, db); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	authService := services.NewAuthService(
		repository.NewUserRepository(db),
		repository.NewSessionRepository(db),
		repository.NewAuditRepository(db),
	)
	authHandler := handlers.NewAuthHandler(authService)
	auditRepo := repository.NewAuditRepository(db)
	auditService := services.NewAuditService(auditRepo)
	catalogRepo := repository.NewCatalogRepository(db)
	purchasingRepo := repository.NewPurchasingRepository(db)
	promotionRepo := repository.NewPromotionRepository(db)
	catalogService := services.NewCatalogService(catalogRepo, purchasingRepo, promotionRepo)
	purchasingService := services.NewPurchasingService(purchasingRepo, catalogRepo, auditRepo)
	catalogHandler := handlers.NewCatalogHandler(catalogService)
	purchasingHandler := handlers.NewPurchasingHandler(purchasingService)
	operationsRepo := repository.NewOperationsRepository(db)
	operationsService := services.NewOperationsService(operationsRepo, auditRepo)
	operationsHandler := handlers.NewOperationsHandler(operationsService)
	promotionService := services.NewPromotionService(promotionRepo)
	promotionHandler := handlers.NewPromotionHandler(promotionService)
	reportsService := services.NewReportsService(repository.NewReportsRepository(db))
	reportsHandler := handlers.NewReportsHandler(reportsService, auditService)
	auditHandler := handlers.NewAuditHandler(auditService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("POST /api/auth/bootstrap", authHandler.Bootstrap)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	protected := middleware.RequireAuth(authService, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/me":
			authHandler.Me(w, r)
		case "/api/auth/logout":
			authHandler.Logout(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	mux.Handle("GET /api/auth/me", protected)
	mux.Handle("POST /api/auth/logout", protected)
	// Catalog and purchasing endpoints use the authenticated request user. Product
	// and category changes are admin-only; receipts are available to staff.
	adminCatalog := func(h http.Handler) http.Handler {
		return middleware.RequireAuth(authService, middleware.RequireRole("admin")(h))
	}
	staffOrAdmin := func(h http.Handler) http.Handler {
		return middleware.RequireAuth(authService, middleware.RequireRole("admin", "staff")(h))
	}
	mux.Handle("GET /api/users", adminCatalog(http.HandlerFunc(authHandler.ListUsers)))
	mux.Handle("POST /api/users", adminCatalog(http.HandlerFunc(authHandler.CreateUser)))
	mux.Handle("PUT /api/users/{id}", adminCatalog(http.HandlerFunc(authHandler.UpdateUser)))
	mux.Handle("DELETE /api/users/{id}", adminCatalog(http.HandlerFunc(authHandler.DeactivateUser)))
	mux.Handle("POST /api/users/{id}/activate", adminCatalog(http.HandlerFunc(authHandler.ActivateUser)))
	mux.Handle("GET /api/categories", staffOrAdmin(http.HandlerFunc(catalogHandler.ListCategories)))
	mux.Handle("POST /api/categories", adminCatalog(http.HandlerFunc(catalogHandler.CreateCategory)))
	mux.Handle("PUT /api/categories/{id}", adminCatalog(http.HandlerFunc(catalogHandler.UpdateCategory)))
	mux.Handle("DELETE /api/categories/{id}", adminCatalog(http.HandlerFunc(catalogHandler.DeleteCategory)))
	mux.Handle("POST /api/categories/{id}/activate", adminCatalog(http.HandlerFunc(catalogHandler.ActivateCategory)))
	mux.Handle("GET /api/products", staffOrAdmin(http.HandlerFunc(catalogHandler.ListProducts)))
	mux.Handle("GET /api/products/{id}", staffOrAdmin(http.HandlerFunc(catalogHandler.GetProduct)))
	mux.Handle("GET /api/products/{id}/price", staffOrAdmin(http.HandlerFunc(catalogHandler.CalculatePrice)))
	mux.Handle("POST /api/products", adminCatalog(http.HandlerFunc(catalogHandler.CreateProduct)))
	mux.Handle("PUT /api/products/{id}", adminCatalog(http.HandlerFunc(catalogHandler.UpdateProduct)))
	mux.Handle("DELETE /api/products/{id}", adminCatalog(http.HandlerFunc(catalogHandler.DeleteProduct)))
	mux.Handle("POST /api/products/{id}/activate", adminCatalog(http.HandlerFunc(catalogHandler.ActivateProduct)))
	mux.Handle("GET /api/products/{id}/barcode-label", staffOrAdmin(http.HandlerFunc(catalogHandler.BarcodeLabel)))
	mux.Handle("POST /api/products/import", adminCatalog(http.HandlerFunc(catalogHandler.ImportProducts)))
	mux.Handle("GET /api/dealers", staffOrAdmin(http.HandlerFunc(purchasingHandler.ListDealers)))
	mux.Handle("POST /api/dealers", adminCatalog(http.HandlerFunc(purchasingHandler.CreateDealer)))
	mux.Handle("PUT /api/dealers/{id}", adminCatalog(http.HandlerFunc(purchasingHandler.UpdateDealer)))
	mux.Handle("DELETE /api/dealers/{id}", adminCatalog(http.HandlerFunc(purchasingHandler.DeleteDealer)))
	mux.Handle("POST /api/dealers/{id}/activate", adminCatalog(http.HandlerFunc(purchasingHandler.ActivateDealer)))
	mux.Handle("GET /api/purchases", staffOrAdmin(http.HandlerFunc(purchasingHandler.ListPurchases)))
	mux.Handle("GET /api/purchases/{id}", staffOrAdmin(http.HandlerFunc(purchasingHandler.GetPurchase)))
	mux.Handle("POST /api/purchases", staffOrAdmin(http.HandlerFunc(purchasingHandler.CreatePurchase)))
	mux.Handle("POST /api/purchases/{id}/payments", adminCatalog(http.HandlerFunc(purchasingHandler.PayPurchase)))
	mux.Handle("POST /api/purchase-returns", adminCatalog(http.HandlerFunc(purchasingHandler.ReturnPurchase)))
	mux.Handle("GET /api/purchase-returns", adminCatalog(http.HandlerFunc(purchasingHandler.ListReturns)))
	mux.Handle("POST /api/sales", staffOrAdmin(http.HandlerFunc(operationsHandler.CreateSale)))
	mux.Handle("GET /api/sales", staffOrAdmin(http.HandlerFunc(operationsHandler.ListSales)))
	mux.Handle("GET /api/sales/{id}", staffOrAdmin(http.HandlerFunc(operationsHandler.GetSale)))
	mux.Handle("POST /api/sales/{id}/return", adminCatalog(http.HandlerFunc(operationsHandler.CreateReturnForSale)))
	mux.Handle("GET /api/customers", staffOrAdmin(http.HandlerFunc(operationsHandler.ListCustomers)))
	mux.Handle("POST /api/customers", staffOrAdmin(http.HandlerFunc(operationsHandler.CreateCustomer)))
	mux.Handle("GET /api/customers/{id}", staffOrAdmin(http.HandlerFunc(operationsHandler.GetCustomer)))
	mux.Handle("PUT /api/customers/{id}", adminCatalog(http.HandlerFunc(operationsHandler.UpdateCustomer)))
	mux.Handle("DELETE /api/customers/{id}", adminCatalog(http.HandlerFunc(operationsHandler.DeleteCustomer)))
	mux.Handle("POST /api/customers/{id}/activate", adminCatalog(http.HandlerFunc(operationsHandler.ActivateCustomer)))
	mux.Handle("GET /api/customers/{id}/debt", staffOrAdmin(http.HandlerFunc(operationsHandler.CustomerDebt)))
	mux.Handle("POST /api/customers/{id}/payments", staffOrAdmin(http.HandlerFunc(operationsHandler.PayDebt)))
	mux.Handle("POST /api/customers/{id}/pay", staffOrAdmin(http.HandlerFunc(operationsHandler.PayDebt)))
	mux.Handle("POST /api/sales-returns", adminCatalog(http.HandlerFunc(operationsHandler.CreateReturn)))
	mux.Handle("GET /api/cash-shifts", staffOrAdmin(http.HandlerFunc(operationsHandler.ListShifts)))
	mux.Handle("GET /api/cash/shifts/current", staffOrAdmin(http.HandlerFunc(operationsHandler.CurrentShift)))
	mux.Handle("GET /api/cash-shifts/{id}/movements", staffOrAdmin(http.HandlerFunc(operationsHandler.ListCashMovements)))
	mux.Handle("POST /api/cash-shifts", staffOrAdmin(http.HandlerFunc(operationsHandler.OpenShift)))
	mux.Handle("POST /api/cash/shifts/open", staffOrAdmin(http.HandlerFunc(operationsHandler.OpenShift)))
	mux.Handle("POST /api/cash-shifts/{id}/close", staffOrAdmin(http.HandlerFunc(operationsHandler.CloseShift)))
	mux.Handle("POST /api/cash/shifts/{id}/close", staffOrAdmin(http.HandlerFunc(operationsHandler.CloseShift)))
	mux.Handle("POST /api/cash/movements", staffOrAdmin(http.HandlerFunc(operationsHandler.CreateManualCashMovement)))
	mux.Handle("GET /api/expense-categories", staffOrAdmin(http.HandlerFunc(operationsHandler.ListExpenseCategories)))
	mux.Handle("POST /api/expense-categories", adminCatalog(http.HandlerFunc(operationsHandler.CreateExpenseCategory)))
	mux.Handle("PUT /api/expense-categories/{id}", adminCatalog(http.HandlerFunc(operationsHandler.UpdateExpenseCategory)))
	mux.Handle("DELETE /api/expense-categories/{id}", adminCatalog(http.HandlerFunc(operationsHandler.DeleteExpenseCategory)))
	mux.Handle("POST /api/expense-categories/{id}/activate", adminCatalog(http.HandlerFunc(operationsHandler.ActivateExpenseCategory)))
	mux.Handle("POST /api/expenses", adminCatalog(http.HandlerFunc(operationsHandler.CreateExpense)))
	mux.Handle("POST /api/inventory-owner-expenses", adminCatalog(http.HandlerFunc(operationsHandler.CreateOwnerExpense)))
	mux.Handle("GET /api/promotions", staffOrAdmin(http.HandlerFunc(promotionHandler.List)))
	mux.Handle("GET /api/promotions/active", staffOrAdmin(http.HandlerFunc(promotionHandler.ListActive)))
	mux.Handle("GET /api/promotions/{id}", staffOrAdmin(http.HandlerFunc(promotionHandler.Get)))
	mux.Handle("POST /api/promotions", adminCatalog(http.HandlerFunc(promotionHandler.Create)))
	mux.Handle("PUT /api/promotions/{id}", adminCatalog(http.HandlerFunc(promotionHandler.Update)))
	mux.Handle("DELETE /api/promotions/{id}", adminCatalog(http.HandlerFunc(promotionHandler.Delete)))
	mux.Handle("POST /api/promotions/{id}/activate", adminCatalog(http.HandlerFunc(promotionHandler.Activate)))
	mux.Handle("GET /api/reports/sales", adminCatalog(http.HandlerFunc(reportsHandler.Sales)))
	mux.Handle("GET /api/reports/purchases", adminCatalog(http.HandlerFunc(reportsHandler.Purchases)))
	mux.Handle("GET /api/reports/stock", adminCatalog(http.HandlerFunc(reportsHandler.Stock)))
	mux.Handle("GET /api/reports/profit", adminCatalog(http.HandlerFunc(reportsHandler.Profit)))
	mux.Handle("GET /api/reports/dashboard", adminCatalog(http.HandlerFunc(reportsHandler.Dashboard)))
	mux.Handle("GET /api/exports/sales.xlsx", adminCatalog(http.HandlerFunc(reportsHandler.ExportSales)))
	mux.Handle("GET /api/exports/debt.xlsx", adminCatalog(http.HandlerFunc(reportsHandler.ExportDebt)))
	mux.Handle("GET /api/exports/dealers-purchases.xlsx", adminCatalog(http.HandlerFunc(reportsHandler.ExportPurchases)))
	mux.Handle("GET /api/exports/expenses.xlsx", adminCatalog(http.HandlerFunc(reportsHandler.ExportExpenses)))
	mux.Handle("GET /api/exports/returns.xlsx", adminCatalog(http.HandlerFunc(reportsHandler.ExportReturns)))
	mux.Handle("GET /api/audit-log", adminCatalog(http.HandlerFunc(auditHandler.List)))
	adminOnly := middleware.RequireAuth(authService, middleware.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "role": "admin"})
	})))
	mux.Handle("/api/admin/ping", adminOnly)
	staffOrAdminPing := middleware.RequireAuth(authService, middleware.RequireRole("admin", "staff")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})))
	mux.Handle("/api/staff/ping", staffOrAdminPing)

	server := &http.Server{Addr: address, Handler: loggingMiddleware(mux), ReadHeaderTimeout: 5 * time.Second}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()
	log.Printf("server listening on %s", address)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusResponseWriter{ResponseWriter: w}
		start := time.Now()
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		fmt.Fprintf(os.Stdout, "%s %s %d %s\n", r.Method, r.URL.Path, status, time.Since(start))
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

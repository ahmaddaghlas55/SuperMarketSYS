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
	)
	authHandler := handlers.NewAuthHandler(authService)

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
	adminOnly := middleware.RequireAuth(authService, middleware.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "role": "admin"})
	})))
	mux.Handle("/api/admin/ping", adminOnly)
	staffOrAdmin := middleware.RequireAuth(authService, middleware.RequireRole("admin", "staff")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})))
	mux.Handle("/api/staff/ping", staffOrAdmin)

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

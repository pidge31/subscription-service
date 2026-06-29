// @title           Subscription Service API
// @version         1.0
// @description     REST API for aggregating online subscription data.
// @host            localhost:8080
// @BasePath        /

package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratePostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/pidge31/subscription-service/docs"
	app "github.com/pidge31/subscription-service/internal"
	"github.com/pidge31/subscription-service/migrations"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := app.LoadConfig()
	db := app.NewDB(cfg)
	defer db.Close()

	runMigrations(db.DB)

	repo := app.NewRepository(db)
	svc := app.NewService(repo)
	h := app.NewHandler(svc)

	mux := http.NewServeMux()
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
	mux.Handle("/", h.Routes())

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}

func runMigrations(db *sql.DB) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		slog.Error("failed to create migration source", "error", err)
		os.Exit(1)
	}

	driver, err := migratePostgres.WithInstance(db, &migratePostgres.Config{})
	if err != nil {
		slog.Error("failed to create migration driver", "error", err)
		os.Exit(1)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		slog.Error("failed to create migrator", "error", err)
		os.Exit(1)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied successfully")
}

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"homebudget/internal/config"
	"homebudget/internal/database"
	"homebudget/internal/handler"
	"homebudget/internal/repository"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	repos := &repository.Repos{
		Member:      repository.NewMemberRepo(db),
		Account:     repository.NewAccountRepo(db),
		Category:    repository.NewCategoryRepo(db),
		SharedGroup: repository.NewSharedGroupRepo(db),
		Transaction: repository.NewTransactionRepo(db),
		Planned:     repository.NewPlannedRepo(db),
		Analytics:   repository.NewAnalyticsRepo(db),
		Lookup:      repository.NewLookupRepo(db),
		Loan:        repository.NewLoanRepo(db),
	}

	router := handler.NewRouter(repos, cfg.CORSOrigin)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⏳ Shutting down…")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("👋 Bye")
}
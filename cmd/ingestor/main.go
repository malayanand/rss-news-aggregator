package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/malayanand/newsx/internal/api"
	"github.com/malayanand/newsx/internal/classifier"
	"github.com/malayanand/newsx/internal/scheduler"
	"github.com/malayanand/newsx/internal/store"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	clfURL := os.Getenv("CLASSIFIER_URL")
	log.Printf("dsn: %s", dsn)

	db, err := store.NewDbConnection(dsn)
	if err != nil {
		log.Fatalf("Error connecting to db: %v", err)
	}

	clf := classifier.NewClient(clfURL)
	srv := api.NewServer(db, clf)
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: srv,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go scheduler.Start(ctx, db, clf)
	go func() {
		log.Printf("listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("error starting server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received, shutting down…")

	// call the shutdown on the http server to close the server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP server Shutdown: %v", err)
	}
	log.Println("Server gracefully stopped")
}

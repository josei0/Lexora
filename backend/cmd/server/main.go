package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lexora/backend/config"
	httpdelivery "github.com/lexora/backend/internal/delivery/http"
	"github.com/lexora/backend/internal/delivery/http/handler"
	"github.com/lexora/backend/internal/repository/postgres"
)

func main() {
	// load env
	config.LoadDotenv(".env")
	config.LoadDotenv("../.env")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	// db pool
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()
	log.Println("postgres connected")

	// wiring
	api := handler.New()
	router := httpdelivery.NewRouter(api)
	server := httpdelivery.NewServer(cfg.Port, router)

	// run
	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := server.Start(); err != nil {
			log.Fatalf("server: %v", err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

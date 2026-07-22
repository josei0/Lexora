package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/lexora/backend/config"
	"github.com/lexora/backend/internal/repository/postgres"
	"github.com/lexora/backend/pkg/hash"
)

// seed super admin
func main() {
	config.LoadDotenv(".env")
	config.LoadDotenv("../.env")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.SuperAdminPassword == "" {
		log.Fatal("SUPERADMIN_PASSWORD required in .env")
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	pw, err := hash.Password(cfg.SuperAdminPassword)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}

	// idempotent upsert
	_, err = pool.Exec(ctx, `
		insert into users (email, password_hash, full_name, system_role, is_active)
		values ($1, $2, 'Super Admin', 'super_admin', true)
		on conflict (email) do update set password_hash = excluded.password_hash
	`, cfg.SuperAdminEmail, pw)
	if err != nil && err != pgx.ErrNoRows {
		log.Fatalf("seed: %v", err)
	}

	log.Printf("super admin seeded: %s", cfg.SuperAdminEmail)
}

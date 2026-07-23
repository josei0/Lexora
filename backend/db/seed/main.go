package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexora/backend/config"
	"github.com/lexora/backend/internal/domain"
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

	seedPlans(ctx, pool)
}

// seed plans (idempotent). Semua tier -> Sonnet: Maia baru serve Sonnet, Opus/Haiku 404.
func seedPlans(ctx context.Context, pool *pgxpool.Pool) {
	repo := postgres.NewPlanRepo(pool)
	plans := []domain.Plan{
		{Code: domain.PlanDemo, Name: "Demo", Model: "maia/claude-sonnet-4-5", PriceIDR: 0, MonthlyTokenLimit: 200_000, IsActive: true},
		{Code: domain.PlanPro, Name: "Pro", Model: "maia/claude-sonnet-4-5", PriceIDR: 275_000, MonthlyTokenLimit: 2_000_000, IsActive: true},
	}
	for i := range plans {
		if err := repo.Upsert(ctx, &plans[i]); err != nil {
			log.Fatalf("seed plan %s: %v", plans[i].Code, err)
		}
	}
	log.Printf("plans seeded: demo (200k tok, gratis), pro (2jt tok/seat, Rp275rb)")
}

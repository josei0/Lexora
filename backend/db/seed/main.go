package main

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexora/backend/config"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/repository/postgres"
	"github.com/lexora/backend/pkg/hash"
)

// ponytail: password dev seragam untuk 3 akun tes lokal, bukan buat produksi
const devPassword = "password123"

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

	// 1. super admin (tanpa org)
	upsertUser(ctx, pool, cfg.SuperAdminEmail, cfg.SuperAdminPassword, "Super Admin", domain.SystemRoleSuperAdmin)
	log.Printf("super admin: %s", cfg.SuperAdminEmail)

	seedPlans(ctx, pool)

	// 2. org Pro: admin (org_admin) + user pro (member)
	proOrg := upsertOrg(ctx, pool, "Firma Hukum Lexora", "firma-lexora")
	subscribe(ctx, pool, proOrg, domain.PlanPro, 5)
	admin := upsertUser(ctx, pool, "admin@lexora.id", devPassword, "Admin Firma", domain.SystemRoleNone)
	proUser := upsertUser(ctx, pool, "pro@lexora.id", devPassword, "User Pro", domain.SystemRoleNone)
	addMember(ctx, pool, admin, proOrg, domain.OrgRoleAdmin)
	addMember(ctx, pool, proUser, proOrg, domain.OrgRoleMember)

	// 3. org Demo: user free (org_admin biar bisa lihat dashboard free tier)
	freeOrg := upsertOrg(ctx, pool, "Kantor Hukum Merdeka", "kantor-merdeka")
	subscribe(ctx, pool, freeOrg, domain.PlanDemo, 1)
	freeUser := upsertUser(ctx, pool, "free@lexora.id", devPassword, "User Free", domain.SystemRoleNone)
	addMember(ctx, pool, freeUser, freeOrg, domain.OrgRoleAdmin)

	log.Printf("seed selesai: 4 akun (superadmin, admin, pro, free)")
}

// idempotent user upsert; return id
func upsertUser(ctx context.Context, pool *pgxpool.Pool, email, password, name, role string) uuid.UUID {
	pw, err := hash.Password(password)
	if err != nil {
		log.Fatalf("hash %s: %v", email, err)
	}
	var id uuid.UUID
	err = pool.QueryRow(ctx, `
		insert into users (email, password_hash, full_name, system_role, is_active, must_change_password)
		values ($1, $2, $3, $4, true, false)
		on conflict (email) do update set password_hash = excluded.password_hash, full_name = excluded.full_name
		returning id`, email, pw, name, role).Scan(&id)
	if err != nil {
		log.Fatalf("user %s: %v", email, err)
	}
	return id
}

// idempotent org upsert; return id
func upsertOrg(ctx context.Context, pool *pgxpool.Pool, name, slug string) uuid.UUID {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		insert into organizations (name, slug) values ($1, $2)
		on conflict (slug) do update set name = excluded.name
		returning id`, name, slug).Scan(&id)
	if err != nil {
		log.Fatalf("org %s: %v", slug, err)
	}
	return id
}

func addMember(ctx context.Context, pool *pgxpool.Pool, userID, orgID uuid.UUID, role string) {
	_, err := pool.Exec(ctx, `
		insert into memberships (user_id, organization_id, role) values ($1, $2, $3)
		on conflict (user_id, organization_id) do update set role = excluded.role`,
		userID, orgID, role)
	if err != nil {
		log.Fatalf("membership: %v", err)
	}
}

func subscribe(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, planCode string, seats int) {
	plan, err := postgres.NewPlanRepo(pool).ByCode(ctx, planCode)
	if err != nil {
		log.Fatalf("plan %s: %v", planCode, err)
	}
	err = postgres.NewSubscriptionRepo(pool).Upsert(ctx, &domain.Subscription{
		OrganizationID: orgID, PlanID: plan.ID, Seats: seats,
	})
	if err != nil {
		log.Fatalf("subscribe %s: %v", planCode, err)
	}
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
	log.Printf("plans: demo (200k tok, gratis), pro (2jt tok/seat, Rp275rb)")
}

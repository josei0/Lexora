// usertool: CRUD akun dari CLI (dipakai lewat `docker compose run --rm backend ./usertool`)
package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexora/backend/config"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/hash"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cfg, err := config.Load()
	if err != nil {
		die("config: %v", err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		die("db: %v", err)
	}
	defer pool.Close()

	switch os.Args[1] {
	case "list":
		list(ctx, pool)
	case "create":
		args := os.Args[2:]
		if len(args) < 3 {
			die("usage: create <email> <password> <name> [role=none|super_admin]")
		}
		role := domain.SystemRoleNone
		if len(args) >= 4 {
			role = args[3]
		}
		create(ctx, pool, args[0], args[1], args[2], role)
	case "passwd":
		mustArgs(4, "passwd <email> <new-password>")
		passwd(ctx, pool, os.Args[2], os.Args[3])
	case "activate":
		mustArgs(3, "activate <email>")
		setActive(ctx, pool, os.Args[2], true)
	case "deactivate":
		mustArgs(3, "deactivate <email>")
		setActive(ctx, pool, os.Args[2], false)
	case "delete":
		mustArgs(3, "delete <email>")
		del(ctx, pool, os.Args[2])
	default:
		usage()
	}
}

func list(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `
		select email, full_name, system_role, is_active, created_at
		from users order by created_at`)
	if err != nil {
		die("query: %v", err)
	}
	defer rows.Close()
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "EMAIL\tNAME\tROLE\tACTIVE\tCREATED")
	for rows.Next() {
		var email, name, role string
		var active bool
		var created interface{}
		if err := rows.Scan(&email, &name, &role, &active, &created); err != nil {
			die("scan: %v", err)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%v\n", email, name, role, active, created)
	}
	w.Flush()
}

func create(ctx context.Context, pool *pgxpool.Pool, email, password, name, role string) {
	if role != domain.SystemRoleNone && role != domain.SystemRoleSuperAdmin {
		die("role harus 'none' atau 'super_admin', bukan %q", role)
	}
	pw, err := hash.Password(password)
	if err != nil {
		die("hash: %v", err)
	}
	_, err = pool.Exec(ctx, `
		insert into users (email, password_hash, full_name, system_role, is_active, must_change_password)
		values ($1, $2, $3, $4, true, false)`, email, pw, name, role)
	if err != nil {
		die("create: %v", err)
	}
	fmt.Printf("dibuat: %s (%s)\n", email, role)
}

func passwd(ctx context.Context, pool *pgxpool.Pool, email, password string) {
	pw, err := hash.Password(password)
	if err != nil {
		die("hash: %v", err)
	}
	affect(ctx, pool, `update users set password_hash=$1, must_change_password=false where email=$2`, pw, email)
	fmt.Printf("password direset: %s\n", email)
}

func setActive(ctx context.Context, pool *pgxpool.Pool, email string, active bool) {
	affect(ctx, pool, `update users set is_active=$1 where email=$2`, active, email)
	fmt.Printf("is_active=%v: %s\n", active, email)
}

func del(ctx context.Context, pool *pgxpool.Pool, email string) {
	affect(ctx, pool, `delete from users where email=$1`, email)
	fmt.Printf("dihapus: %s\n", email)
}

// affect jalanin exec dan pastikan minimal 1 baris kena (email valid)
func affect(ctx context.Context, pool *pgxpool.Pool, sql string, args ...interface{}) {
	tag, err := pool.Exec(ctx, sql, args...)
	if err != nil {
		die("exec: %v", err)
	}
	if tag.RowsAffected() == 0 {
		die("tidak ada user yang cocok")
	}
}

func mustArgs(n int, u string) {
	if len(os.Args) < n {
		die("usage: %s", u)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usertool <command>
  list
  create <email> <password> <name> [role=none|super_admin]
  passwd <email> <new-password>
  activate <email>
  deactivate <email>
  delete <email>`)
	os.Exit(2)
}

func die(f string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, f+"\n", a...)
	os.Exit(1)
}

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lexora/backend/config"
	httpdelivery "github.com/lexora/backend/internal/delivery/http"
	"github.com/lexora/backend/internal/delivery/http/handler"
	"github.com/lexora/backend/internal/repository/postgres"
	"github.com/lexora/backend/internal/repository/qdrant"
	"github.com/lexora/backend/internal/usecase"
	"github.com/lexora/backend/pkg/embedding"
	"github.com/lexora/backend/pkg/extract"
	"github.com/lexora/backend/pkg/jwt"
	"github.com/lexora/backend/pkg/llm"
	"github.com/lexora/backend/pkg/mailer"
	"github.com/lexora/backend/pkg/oauth"
	"github.com/lexora/backend/pkg/payment"
	"github.com/lexora/backend/pkg/storage"
	"github.com/lexora/backend/pkg/websearch"
)

// adapter: pkg/oauth.Google -> usecase.GoogleVerifier (translate tipe Claims).
type googleVerifier struct{ g *oauth.Google }

func (v googleVerifier) Enabled() bool { return v.g.Enabled() }
func (v googleVerifier) Verify(ctx context.Context, idToken string) (*usecase.GoogleClaims, error) {
	c, err := v.g.Verify(ctx, idToken)
	if err != nil {
		return nil, err
	}
	return &usecase.GoogleClaims{Sub: c.Sub, Email: c.Email, Name: c.Name}, nil
}

// maiaBalanceWatch: goroutine TERPISAH (interval beda dari dailyJobs 24 jam).
// Cek estimasi saldo Maia tiap 6 jam, email alert kalau di bawah threshold (update6 §4.3).
// ponytail: kirim tiap tick selama di bawah threshold (bisa 4 email/hari). Tambah
// throttle "1x per turun-lewat-threshold" kalau inbox jadi berisik.
func maiaBalanceWatch(ctx context.Context, est *usecase.BalanceEstimator, m *mailer.Mailer, threshold float64) {
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			bal, err := est.Estimate(ctx)
			if err != nil {
				log.Printf("estimasi saldo maia gagal: %v", err)
				continue
			}
			if bal < threshold {
				_ = m.Send("mindlaw.env@gmail.com", "⚠️ Saldo Maia menipis",
					fmt.Sprintf("Estimasi saldo Maia $%.2f di bawah ambang $%.2f. Top-up manual di dashboard Maia.", bal, threshold))
			}
		}
	}
}

// ticker harian: terbitkan invoice renewal H-7 + retensi web_searches 90 hari.
// Satu ticker untuk semua job harian (update5 §8). Recovery: jalan sekali saat start.
// ponytail: pindah ke job terjadwal kalau nanti ada banyak job
func dailyJobs(ctx context.Context, invUC *usecase.Invoice, searches *postgres.WebSearchRepo) {
	const retention = 90 * 24 * time.Hour
	t := time.NewTicker(24 * time.Hour)
	defer t.Stop()
	for {
		if n, err := invUC.CreateRenewals(ctx, time.Now()); err != nil {
			log.Printf("invoice renewal: %v", err)
		} else if n > 0 {
			log.Printf("invoice renewal: %d invoice terbit", n)
		}
		if n, err := searches.DeleteOlderThan(ctx, time.Now().Add(-retention)); err != nil {
			log.Printf("retensi web_searches: %v", err)
		} else if n > 0 {
			log.Printf("retensi web_searches: %d baris dihapus", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func main() {
	config.LoadDotenv(".env")
	config.LoadDotenv("../.env")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancelRoot := context.WithCancel(context.Background())
	defer cancelRoot()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()
	log.Println("postgres connected")

	signer := jwt.New(cfg.JWTSecret, cfg.JWTAccessTTL, jwt.AudienceApp)
	adminSigner := jwt.New(cfg.JWTAdminSecret, cfg.JWTAccessTTL, jwt.AudienceAdmin)
	userRepo := postgres.NewUserRepo(pool)
	orgRepo := postgres.NewOrgRepo(pool)
	memberRepo := postgres.NewMembershipRepo(pool)
	refreshRepo := postgres.NewRefreshRepo(pool)
	docRepo := postgres.NewDocumentRepo(pool)

	recoveryRepo := postgres.NewRecoveryRepo(pool)
	authUC := usecase.NewAuth(userRepo, memberRepo, refreshRepo, recoveryRepo, signer, adminSigner, cfg.JWTRefreshTTL)
	orgUC := usecase.NewOrganization(orgRepo, userRepo, memberRepo)

	store := storage.NewLocal(cfg.StorageDir)
	extractor := extract.New(true) // ingestion: OCR halaman scan
	embedder := embedding.NewMaia(cfg.EmbeddingURL, cfg.MaiaAPIKey, cfg.EmbeddingModel, cfg.EmbeddingDim)
	vectors := qdrant.New(cfg.QdrantURL)
	ingestUC := usecase.NewIngestion(docRepo, store, extractor, embedder, vectors)
	ingestUC.Start(ctx, 3)
	ingestUC.Recover(ctx)

	docUC := usecase.NewDocument(docRepo, store, ingestUC)

	guard := websearch.NewGuard(cfg.WebSearchDomains)
	searchProvider := websearch.NewMaiaSearch(cfg.EmbeddingURL, cfg.MaiaAPIKey, cfg.WebSearchModel, cfg.WebSearchDomains)
	webUC := usecase.NewWebIngest(docRepo, store, websearch.NewFetcher(guard), searchProvider, ingestUC)

	chatRepo := postgres.NewChatRepo(pool)
	llmHigh := llm.NewMaia(cfg.EmbeddingURL, cfg.MaiaAPIKey, cfg.ChatModelHigh)
	llmNormal := llm.NewMaia(cfg.EmbeddingURL, cfg.MaiaAPIKey, cfg.ChatModelNormal)
	ragUC := usecase.NewRAG(chatRepo, embedder, vectors, llmHigh, llmNormal, cfg.RAGTopK, cfg.RAGMinScore)

	planRepo := postgres.NewPlanRepo(pool)
	subRepo := postgres.NewSubscriptionRepo(pool)
	promptRepo := postgres.NewPromptRepo(pool)
	usageRepo := postgres.NewUsageRepo(pool)

	billingUC := usecase.NewBilling(subRepo, usageRepo)
	subUC := usecase.NewSubscription(subRepo, planRepo, usageRepo)
	dashUC := usecase.NewDashboard(usageRepo, subRepo)
	promptUC := usecase.NewPrompt(promptRepo)
	exportUC := usecase.NewExport(ragUC)

	searchRepo := postgres.NewWebSearchRepo(pool)
	invoiceRepo := postgres.NewInvoiceRepo(pool)
	invoiceUC := usecase.NewInvoice(invoiceRepo)
	topupRepo := postgres.NewTopupRepo(pool)
	billingUC.SetTopup(topupRepo)
	invoiceUC.SetTopup(subRepo, topupRepo)
	// gateway aktif: Mayar (update7). Kosong = mode manual (invoice pending tanpa checkout URL).
	// mayarGW disimpan untuk juga jadi verifier webhook (re-fetch) di invoiceAPI.
	var mayarGW *payment.Mayar
	if cfg.MayarAPIKey != "" {
		mayarGW = payment.NewMayar(cfg.MayarAPIKey, cfg.MayarBaseURL, cfg.MayarSuccessURL)
		invoiceUC.SetGateway(mayarGW)
	}
	ragUC.SetBilling(billingUC)
	ragUC.SetWebSearch(searchProvider, searchRepo)
	docUC.SetBilling(billingUC)
	webUC.SetBilling(billingUC)
	webUC.SetWebSearchRepo(searchRepo)
	go dailyJobs(ctx, invoiceUC, searchRepo)
	ragUC.SetPrompts(promptRepo)
	ragUC.SetExtractor(extract.New(false)) // lampiran chat sinkron: tanpa OCR
	mail := mailer.New(cfg.SMTPUser, cfg.SMTPAppPassword)
	orgUC.SetSeatGuard(subUC)
	orgUC.SetMailer(mail, cfg.AppBaseURL)
	authUC.SetGoogle(googleVerifier{oauth.NewGoogle(cfg.GoogleClientID)}, orgUC)

	// alert saldo Maia: aktif kalau threshold di-set (update6 §4.3)
	if cfg.MaiaBalanceThreshold > 0 {
		est := usecase.NewBalanceEstimator(usageRepo, cfg.MaiaTopupTotalUSD)
		go maiaBalanceWatch(ctx, est, mail, cfg.MaiaBalanceThreshold)
	}

	auditRepo := postgres.NewAuditRepo(pool)
	auditUC := usecase.NewAudit(auditRepo)

	api := handler.New(authUC, orgUC, docUC, auditUC, cfg.JWTRefreshTTL, cfg.CookieSecure, cfg.CORSOriginsAdmin)
	chatAPI := handler.NewChatAPI(ragUC)
	billingAPI := handler.NewBillingAPI(subUC, dashUC, promptUC, exportUC, billingUC, auditUC, cfg.ChatModelNormal)
	webAPI := handler.NewWebAPI(webUC, auditUC)
	invoiceAPI := handler.NewInvoiceAPI(invoiceUC, auditUC)
	if mayarGW != nil {
		invoiceAPI.SetMayarVerifier(mayarGW) // webhook Mayar re-fetch (nil = nonaktif)
	}
	router := httpdelivery.NewRouter(api, chatAPI, billingAPI, webAPI, invoiceAPI, signer, adminSigner, cfg.CORSOriginsApp, cfg.CORSOriginsAdmin)
	server := httpdelivery.NewServer(cfg.Port, router)

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := server.Start(); err != nil {
			log.Fatalf("server: %v", err)
		}
	}()

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

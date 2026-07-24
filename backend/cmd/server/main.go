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
	"github.com/lexora/backend/internal/repository/qdrant"
	"github.com/lexora/backend/internal/usecase"
	"github.com/lexora/backend/pkg/embedding"
	"github.com/lexora/backend/pkg/extract"
	"github.com/lexora/backend/pkg/jwt"
	"github.com/lexora/backend/pkg/llm"
	"github.com/lexora/backend/pkg/storage"
)

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

	authUC := usecase.NewAuth(userRepo, memberRepo, refreshRepo, signer, adminSigner, cfg.JWTRefreshTTL)
	orgUC := usecase.NewOrganization(orgRepo, userRepo, memberRepo)

	store := storage.NewLocal(cfg.StorageDir)
	extractor := extract.New()
	embedder := embedding.NewMaia(cfg.EmbeddingURL, cfg.MaiaAPIKey, cfg.EmbeddingModel, cfg.EmbeddingDim)
	vectors := qdrant.New(cfg.QdrantURL)
	ingestUC := usecase.NewIngestion(docRepo, store, extractor, embedder, vectors)
	ingestUC.Start(ctx, 3)
	ingestUC.Recover(ctx)

	docUC := usecase.NewDocument(docRepo, store, ingestUC)

	chatRepo := postgres.NewChatRepo(pool)
	chatLLM := llm.NewMaia(cfg.EmbeddingURL, cfg.MaiaAPIKey, cfg.ChatModel)
	ragUC := usecase.NewRAG(chatRepo, embedder, vectors, chatLLM, cfg.RAGTopK, cfg.RAGMinScore)

	planRepo := postgres.NewPlanRepo(pool)
	subRepo := postgres.NewSubscriptionRepo(pool)
	promptRepo := postgres.NewPromptRepo(pool)
	usageRepo := postgres.NewUsageRepo(pool)

	billingUC := usecase.NewBilling(subRepo, usageRepo)
	subUC := usecase.NewSubscription(subRepo, planRepo, usageRepo)
	dashUC := usecase.NewDashboard(usageRepo, subRepo)
	promptUC := usecase.NewPrompt(promptRepo)
	exportUC := usecase.NewExport(ragUC)

	ragUC.SetBilling(billingUC)
	ragUC.SetPrompts(promptRepo)
	ragUC.SetExtractor(extractor)
	orgUC.SetSeatGuard(subUC)

	auditRepo := postgres.NewAuditRepo(pool)
	auditUC := usecase.NewAudit(auditRepo)

	api := handler.New(authUC, orgUC, docUC, auditUC, cfg.JWTRefreshTTL, cfg.CookieSecure, cfg.CORSOriginsAdmin)
	chatAPI := handler.NewChatAPI(ragUC)
	billingAPI := handler.NewBillingAPI(subUC, dashUC, promptUC, exportUC, billingUC, auditUC)
	router := httpdelivery.NewRouter(api, chatAPI, billingAPI, signer, adminSigner, cfg.CORSOriginsApp, cfg.CORSOriginsAdmin)
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

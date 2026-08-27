package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"saas-project/backend/internal/analytics"
	"saas-project/backend/internal/auth"
	"saas-project/backend/internal/billing"
	"saas-project/backend/internal/config"
	"saas-project/backend/internal/db"
	"saas-project/backend/internal/link"
	"saas-project/backend/internal/middleware"
	"saas-project/backend/internal/org"
	"saas-project/backend/internal/user"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer database.Close()

	if err := database.RunMigrations(ctx, "migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	issuer := auth.NewIssuer(cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTLMinutes, cfg.JWTRefreshTTLDays)

	userRepo := user.NewRepository(database.Pool)
	orgRepo := org.NewRepository(database.Pool)
	linkRepo := link.NewRepository(database.Pool)
	analyticsRepo := analytics.NewRepository(database.Pool)

	analyticsService := analytics.NewService(analyticsRepo)
	billingService := billing.NewService(billing.Config{
		SecretKey:             cfg.StripeSecretKey,
		WebhookSecret:         cfg.StripeWebhookSecret,
		ProPriceID:            cfg.StripePriceIDPro,
		FreePlanLinkLimit:     cfg.FreePlanLinkLimit,
		FreePlanMonthlyClicks: cfg.FreePlanMonthlyClickLimit,
		FrontendURL:           cfg.FrontendURL,
	}, orgRepo, analyticsService)

	userHandler := user.NewHandler(userRepo, orgRepo, issuer)
	orgHandler := org.NewHandler(orgRepo)
	linkHandler := link.NewHandler(linkRepo, analyticsService, billingService, cfg.BaseURL)
	analyticsHandler := analytics.NewHandler(analyticsService)
	billingHandler := billing.NewHandler(billingService, orgRepo, cfg.StripeWebhookSecret)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(15 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Public short-link redirect.
	r.Get("/{slug}", linkHandler.Redirect)

	// Stripe webhook needs the raw body, so it's mounted outside the JSON API group.
	r.Post("/webhooks/stripe", billingHandler.Webhook)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/signup", userHandler.Signup)
		r.Post("/auth/login", userHandler.Login)
		r.Post("/auth/refresh", userHandler.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(issuer))

			r.Get("/me", userHandler.Me)
			r.Get("/org/members", orgHandler.ListMembers)

			r.Post("/links", linkHandler.Create)
			r.Get("/links", linkHandler.List)
			r.Delete("/links/{id}", linkHandler.Delete)

			r.Get("/analytics/summary", analyticsHandler.Summary)

			r.Post("/billing/checkout", billingHandler.CreateCheckoutSession)
			r.Get("/billing/usage", billingHandler.Usage)
		})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

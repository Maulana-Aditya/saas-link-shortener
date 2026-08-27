// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port                       string
	DatabaseURL                string
	JWTAccessSecret            string
	JWTRefreshSecret           string
	JWTAccessTTLMinutes        int
	JWTRefreshTTLDays          int
	BaseURL                    string
	FrontendURL                string
	StripeSecretKey            string
	StripeWebhookSecret        string
	StripePriceIDPro           string
	FreePlanLinkLimit          int
	FreePlanMonthlyClickLimit  int
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		JWTAccessSecret:      os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:     os.Getenv("JWT_REFRESH_SECRET"),
		JWTAccessTTLMinutes:  getEnvInt("JWT_ACCESS_TTL_MINUTES", 15),
		JWTRefreshTTLDays:    getEnvInt("JWT_REFRESH_TTL_DAYS", 30),
		BaseURL:              getEnv("BASE_URL", "http://localhost:8080"),
		FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:5173"),
		StripeSecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripePriceIDPro:     os.Getenv("STRIPE_PRICE_ID_PRO"),
		FreePlanLinkLimit:        getEnvInt("FREE_PLAN_LINK_LIMIT", 50),
		FreePlanMonthlyClickLimit: getEnvInt("FREE_PLAN_MONTHLY_CLICK_LIMIT", 1000),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTAccessSecret == "" || cfg.JWTRefreshSecret == "" {
		return nil, fmt.Errorf("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET are required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

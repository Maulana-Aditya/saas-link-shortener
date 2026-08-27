package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Service records clicks asynchronously so the public redirect handler is not
// slowed down by analytics writes, and serves aggregate summaries.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// RecordAsync implements link.ClickRecorder.
func (s *Service) RecordAsync(orgID, linkID string, r *http.Request) {
	ipHash := hashIP(clientIP(r))
	referrer := r.Referer()
	userAgent := r.UserAgent()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.repo.InsertClick(ctx, orgID, linkID, strPtr(ipHash), nil, strPtr(referrer), strPtr(userAgent)); err != nil {
			log.Printf("analytics: failed to record click for link %s: %v", linkID, err)
		}
	}()
}

func (s *Service) Summary(ctx context.Context, orgID string, days int) (map[string]any, error) {
	since := time.Now().AddDate(0, 0, -days)

	byDay, err := s.repo.ClicksByDay(ctx, orgID, since)
	if err != nil {
		return nil, err
	}
	referrers, err := s.repo.TopReferrers(ctx, orgID, since, 10)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"clicks_by_day":  byDay,
		"top_referrers":  referrers,
		"since":          since,
	}, nil
}

// MonthlyClickCount implements billing.UsageSource for plan-limit checks.
func (s *Service) MonthlyClickCount(ctx context.Context, orgID string) (int64, error) {
	startOfMonth := time.Now().AddDate(0, 0, -30)
	return s.repo.MonthlyClickCount(ctx, orgID, startOfMonth)
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func hashIP(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:])
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

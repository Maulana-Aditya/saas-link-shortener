# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A multi-tenant SaaS: link shortening + click analytics with Stripe subscription billing (free/pro plans). Backend in Go, frontend in React. Built as a portfolio-quality reference implementation of a real SaaS architecture (auth, multi-tenancy, billing) — extend the domain model (`links`/`click_events`) to pivot to a different product if needed; the auth/org/billing skeleton is meant to be reused as-is.

## Commands

### Backend (`backend/`)
```
docker compose up -d              # start Postgres (from repo root)
go run ./cmd/api                  # run the API; applies pending migrations on startup
go build ./...                    # build
go vet ./...                      # lint
go test ./...                     # test (no test suite yet — add as you go)
```
Config is loaded from `.env` (see `.env.example`) via `internal/config`. `DATABASE_URL`, `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET` are required; everything else has defaults.

### Frontend (`frontend/`)
```
npm install
npm run dev        # dev server on :5173
npm run build      # tsc -b && vite build
npx tsc --noEmit   # type-check only
```
`VITE_API_URL` (see `.env.example`) points at the backend, default `http://localhost:8080`.

## Architecture

### Multi-tenancy
Every domain table (`links`, `click_events`, `api_keys`) is scoped by `org_id`. A user can belong to multiple orgs via `org_members` (role column). The JWT access token embeds both `user_id` and the *active* `org_id` — there is no in-token org switching yet; `org.FirstOrgForUser` picks the first org on login. If you add multi-org switching, it's a new `/api/v1/org/switch`-style endpoint that reissues tokens with a different `org_id` claim.

### Backend package layout (`backend/internal/`)
- `config` — env loading
- `db` — pgx pool + hand-rolled migration runner (`RunMigrations` reads `.up.sql`/`.down.sql` files from `migrations/`, tracks applied versions in `schema_migrations`). No sqlc/ORM — queries are written directly in each package's `repository.go` against `pgxpool.Pool`.
- `auth` — JWT issuing/parsing (`Issuer`) and bcrypt password hashing. Access + refresh tokens are separate HS256 secrets/TTLs.
- `middleware` — `RequireAuth` parses the bearer access token and injects `user_id`/`org_id` into request context; read them back with `middleware.UserID(ctx)` / `middleware.OrgID(ctx)`.
- `user` — signup/login/refresh/me handlers. Signup creates a user **and** an org **and** the owner membership in one sequence (see `user.Handler.Signup`) — org creation is not optional, every user always has at least one org.
- `org` — org CRUD/membership queries, member listing.
- `link` — short link CRUD + the public redirect handler. Slugs are random base62 (7 chars), collisions retried against the DB unique constraint rather than pre-checked. Plan limits are enforced here via the `PlanLimiter` interface (implemented by `billing.Service`) — `link` does not import `billing` directly, avoiding a cycle.
- `analytics` — click recording is fire-and-forget (`Service.RecordAsync` spawns a goroutine with its own background context) so the public redirect stays fast; implements `link.ClickRecorder`. Also implements `billing.UsageSource` (`MonthlyClickCount`) for usage-based plan checks.
- `billing` — Stripe Checkout session creation + webhook handling (`checkout.session.completed` promotes an org to `pro`; `customer.subscription.deleted` downgrades by Stripe customer ID). Depends on `org` and `analytics` (via the `UsageSource` interface) but nothing depends on it — wire new plan-gated features through the `PlanLimiter`/`UsageSource` interfaces, not a direct import.
- `httputil` — shared `WriteJSON`/`WriteError` response helpers used by every handler package.

Dependency direction to preserve: `link` and `analytics` stay billing-agnostic (interfaces only); `billing` is the only package allowed to import both `org` and `analytics` concretely. `cmd/api/main.go` is the sole place that wires concrete implementations into interfaces.

### Public vs authenticated routes
`cmd/api/main.go` mounts three route groups: the public redirect (`GET /{slug}`) and Stripe webhook (`POST /webhooks/stripe`) at root — the webhook needs the raw body, so it's deliberately outside the JSON API group — and everything else under `/api/v1`, split into unauthenticated (`/auth/*`) and `middleware.RequireAuth`-gated routes.

### Frontend (`frontend/src/`)
- `lib/api.ts` — fetch wrapper; stores access/refresh tokens in `localStorage`, auto-retries once on a 401 by calling `/api/v1/auth/refresh`.
- `lib/auth.tsx` — `AuthProvider`/`useAuth()`; hydrates `user`/`org` from `/api/v1/me` on load if a token exists.
- `components/RequireAuth.tsx` — route guard, redirects to `/login`.
- `pages/` — one file per route (`Login`, `Signup`, `Links`, `Analytics`, `Billing`), each owns its own data fetching (no global state library).

### Adding a new domain resource
Follow the existing package shape: `repository.go` (pgx queries against `db.*` structs), `handlers.go` (chi handlers using `middleware.OrgID`/`UserID` + `httputil.WriteJSON`/`WriteError`), wire into `cmd/api/main.go`. Add the table via a new numbered migration pair in `backend/migrations/` (`NNNN_description.up.sql` / `.down.sql`) — always scope new tables by `org_id` unless there's a specific reason not to.

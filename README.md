# Shrtn — Link Shortener & Analytics SaaS

A multi-tenant link-shortening SaaS: short links, click analytics, and Stripe subscription billing. Backend in Go, frontend in React.

## Stack

- **Backend**: Go, `chi` router, `pgx` (no ORM), PostgreSQL, JWT auth, Stripe
- **Frontend**: React + Vite + TypeScript + Tailwind CSS v4, React Router

## Local setup

1. Start Postgres:
   ```
   docker compose up -d
   ```
2. Backend:
   ```
   cd backend
   cp .env.example .env   # then fill in JWT secrets and (optionally) Stripe test keys
   go run ./cmd/api
   ```
   Migrations run automatically on startup. API listens on `:8080` by default.
3. Frontend:
   ```
   cd frontend
   cp .env.example .env
   npm install
   npm run dev
   ```
   Dev server on `:5173`.

## Testing the flow

1. Sign up at `http://localhost:5173/signup` — this creates a user and an org (tenant) in one step.
2. Create a short link from the dashboard.
3. Visit the printed short URL (`http://localhost:8080/<slug>`) — it 302-redirects and records a click.
4. Check the Analytics tab for the recorded click.
5. Billing tab lets you start a Stripe Checkout session (requires real Stripe test-mode keys and a webhook forwarded to `/webhooks/stripe`, e.g. via `stripe listen --forward-to localhost:8080/webhooks/stripe`).

## Architecture

See `CLAUDE.md` for the full architecture breakdown, commands, and conventions used in this repo.

## Known gaps (not yet implemented)

- Email delivery (verification, password reset)
- Team invitations
- Production deployment config / CI
- Automated test suite

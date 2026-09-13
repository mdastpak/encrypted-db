# Plan: Consolidate branches into `main` + Refactor/Security/Optimize

## Context
- Repo: `D:\Projects\dpm/encrypted-db`, Go 1.22.5 (toolchain go1.22.8). Remote `origin` mirrors `main`, `develop`, `feat/database-pooling`, `feat/refactore-v2`.
- Branch tree audit:
  - `main` — scaffold only (`.gitattributes`, `.gitignore`, `LICENSE`, `README.md`). Empty of source.
  - `develop` — most complete: `cmd/server/main.go`, `internal/auth/*` (JWT+RSA), `internal/ssl/*` (admin/user PEMs), `internal/db/migrations/{1,2}_create_*` (currencies + users), `internal/rabbitmq/rabbitmq.go`, `internal/models/{currncies,infra,response,users}`, `pkg/utils/*`, full `config.yaml` (server/postgres/redis/rabbitmq/jwt/otp).
  - `feat/database-pooling` (current worktree) — top-level `main.go` + `services/` + `routers/`, uses `pgxpool`, but has **no auth, no SSL keys, no users migration, no `internal/auth`**. Diverges 1 ahead / 5 behind `develop`.
  - `feat/refactore-v2` — partial auth refactor (`internal/auth/{jwt,middleware}.go`), `system/system.go`; subset of `develop`. Diverges 0 ahead / 4 behind `develop`.
- No `_test.go` files exist; no CI workflow present. No `go.sum` drift suspected.

## Decision (recommended, assumed unless objected)
- **Base = `develop`. `main` will be fast-forwarded to `develop`, then feature branches merged into `main`.** Rationale: `develop` has complete auth/SSL/users; pooling/refactor branches are regressions in surface area.
- **Merge style:** rebase each feature branch onto `main(develop)` then `--no-ff` merge, so history is readable and each feature is a single merge commit.
- **Conflict resolution:** prefer `develop`'s auth/SSL/users/config structure; carry over `pgxpool` pooling from `feat/database-pooling` and any auth-jwt simplification from `feat/refactore-v2`.
- **Scope:** destructive git ops (reset/push to `origin/main`, branch deletion) require explicit approval; plan assumes approval for local `main`.

## Step 1 — Consolidation (git, no source edits until conflicts resolved)
1. Back up: `git branch backup/pre-merge main` (preserve clean state).
2. `git checkout main` → `git reset --hard develop` (main becomes full develop tree).
3. Rebase `feat/database-pooling` onto `main`; merge with `git merge --no-ff --no-commit feat/database-pooling`.
   - Expected conflicts: `config/config.yaml` (develop has jwt/otp; pooling lacks), `internal/db/postgres.go` (sql vs pgxpool), `main.go`/`services/` (pooling has top-level `main.go`; develop has `cmd/server/main.go`). Resolve by:
     - Keep single entrypoint. Two options: (a) move pooling's `services/`+`routers/` into `internal/` and switch entrypoint to `cmd/server/main.go`; (b) adopt top-level `main.go` and `services/` from pooling as the canonical entrypoint, deleting `cmd/server`. **Recommendation (a)** to keep `cmd/` convention.
     - Unify config.yaml: merge jwt/otp sections into config, add env-overridable fields.
     - Adopt `pgxpool` in `internal/db/postgres.go`; drop `database/sql`+lib/pq if unused elsewhere.
4. Rebase `feat/refactore-v2` onto `main`; merge `--no-ff --no-commit`.
   - Expected conflicts: `internal/auth/*`, `internal/handlers/*`. Prefer `develop`'s multi-file auth layout; cherry-pick refactore's simplifications only if equivalent.
5. Run `go build ./...` after each merge before committing.
6. Optionally `git branch -d feat/database-pooling feat/refactore-v2` (after confirming merged).
7. Push `main` to `origin` (requires approval).

## Step 2 — Code Refactoring
1. Replace deprecated `io/ioutil.ReadFile` with `os.ReadFile` in `config/config.go`.
2. Fix `GetNewContext`: `defer cancel()` before `return` cancels context immediately — remove defer or return a factory `NewContext() (ctx, cancel)`.
3. Remove unused imports (`pgx` in pooling postgres.go; verify with `go vet`).
4. Normalize config load path: use `embed` or `-config` flag instead of relative `config/config.yaml`.
5. Make migration path configurable (`-migrations` flag) instead of hardcoded `file://internal/db/migrations`.
6. Introduce service interfaces (e.g., `PostgresService`, `RedisService`, `RabbitMQService`) in `services/` to decouple handlers from concrete implementations.
7. Split fat `handler.go` files into one file per handler (admin currencies already split; replicate pattern).
8. Replace `log` with `log/slog` (structured logging); inject logger.
9. Remove trailing whitespace (found in `config/config.yaml`, `internal/db/migrations/2_create_users_table.up.sql`, `internal/handlers/socket/router.go`).
10. Add `golangci-lint` config (`.golangci.yml`) and enforce `gofmt`/`goimports`.

## Step 3 — Security Enhancements
1. **Secrets:** Remove hardcoded DB/RabbitMQ credentials and JWT material from `config.yaml`. Load via `os.Getenv` with validation in `config.go`; add `config.example.yaml` and `.env.example`. Add `config.yaml` to `.gitignore`.
2. **PostgreSQL:** Change `sslmode=disable` → `sslmode=require` (or `verify-full` with CA); make configurable.
3. **TLS:** Serve HTTP via `ListenAndServeTLS` when `server.tls.enabled`; add cert config.
4. **JWT/RSA:** Ensure `internal/ssl/*` PEMs are not committed (move to external mount / Vault); validate key permissions (0600). Fix `jwt.user.private_key` config typo in develop `config.yaml` (currently points to admin directory — bug).
5. **Transport security:** Add security headers middleware (HSTS, X-Content-Type-Options, X-Frame-Options, CSP); disable `gin.Default()` debug mode in prod.
6. **Input validation:** Use `binding` tags; validate OTP length/charset; reject oversized payloads.
7. **Rate limiting:** Add `golang.org/x/time/rate` or `ulule/limiter` on auth/OTP endpoints.
8. **Errors:** Do not leak internal errors to clients; return generic messages; log details server-side.
9. **Redis/RabbitMQ:** Require auth by default (password not empty); TLS for AMQPS.
10. **Audit log:** Add request logging for admin/user mutations.

## Step 4 — Feature Optimization
1. **DB connection pool:** Set `MaxConns` from config; add health check + `ResetSession`/connect timeouts.
2. **Redis cache:** Add TTL + structured invalidation hooks on currency write paths; check existing `pkg/utils/cache.go` for staleness.
3. **RabbitMQ:** Use publisher confirms + `durable` exchanges/queues; reconnect logic in `rabbitmq.go`.
4. **WebSocket:** Add ping/pong with deadline + compression per `internal/handlers/socket/ping.go`; validate origin.
5. **Graceful shutdown:** Extend `cmd/server/main.go` shutdown to close postgres, redis, rabbitmq, and websocket connections with timeout.
6. **Migrations:** Wrap `runMigrations` with timeout context; treat `ErrNoChange` correctly.

## Validation (run before declaring done)
- `go build ./...`
- `go vet ./...`
- `go test ./...` (new unit tests target: `config`, `helpers/response`, `pkg/utils`; `go test -race`)
- `golangci-lint run` (with config from Step 2.10)
- `gofmt -l .`
- Smoke: containerize Postgres + Redis + RabbitMQ via compose; run binary and health-check endpoints.
- `git diff --check` clean on merged `main`.

## Risks / Open
1. **Canonical base** — assumed `develop`. If `main` is intended to be the pooling layout, re-order steps.
2. **Destructive git** (reset to develop, push) — needs user approval; plan stops at local `main` ready.
3. **Feature-branch semantics** — pooling branch's top-level `main.go`/`services/` vs develop's `cmd/server/` is a structural clash; Step 1.3 recommends adopting `cmd/` convention.
4. **Test coverage** — none exists; Step 4 validation requires writing tests. Scope may be reduced if out of budget.
5. **Secret management system** — recommended env vars; if organization uses Vault/SSM, adapt Step 3.1 accordingly.

## Assumptions
- Author identity for merge commits may use existing git config (no identity change).
- Remote `origin` can be force-updated only with explicit user command (not in this plan).

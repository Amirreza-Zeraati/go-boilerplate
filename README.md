# go-boilerplate

A reusable Go backend template: **Gin + GORM + PostgreSQL + Redis**, with server-side session auth, role-based authorization, and production-minded HTTP middleware. Clone it, rename the module, build the next project on top.

## Stack & features

Foundation + data layer (steps 1–9) and the HTTP/auth layer (steps 10–19):

- Config loader (env + `.env`), structured logging (`log/slog`)
- PostgreSQL via GORM with connection pooling
- **Embedded migrations** (golang-migrate) — runs on startup, no CLI needed
- Repository pattern + service layer (clean separation: handler → service → repository)
- Redis client
- **Session-based auth** (server-side sessions in Redis, opaque cookie — not JWT)
- **Role-based authorization** middleware
- Gin router with an explicit middleware chain: request ID, recovery (JSON), structured request logging, CORS, Redis-backed rate limiting
- Request validation (Gin binding + friendly field errors)
- Health (`/healthz`) and readiness (`/readyz`) endpoints
- Graceful shutdown

## Layout

```
cmd/api/main.go              # composition root: wires everything, handles shutdown
internal/
  config/                    # typed config from env; DSN/URL/Addr builders
  database/                  # gorm connection, pool, Ping, Close, WaitForConnection
  redis/                     # go-redis client wrapper
  session/                   # server-side sessions in Redis (Store interface + impl)
  models/                    # Base (uuid, timestamps, soft delete) + User
  repository/                # interfaces + GORM implementations
  service/                   # business logic (auth: register/authenticate/get)
  dto/                       # request/response shapes + validation tags
  response/                  # standard JSON envelope + validation error mapping
  middleware/                # request id, logger, recovery, cors, ratelimit, auth, rbac, cookies
  handler/                   # HTTP handlers (auth, health)
  server/                    # gin router wiring + http.Server with graceful shutdown
  migrate/                   # embedded-migration runner
migrations/                  # *.sql + embed.go
pkg/logger/                  # slog setup
```

## Getting started

```bash
# 1. Rename the module to your repo (updates go.mod + all imports)
grep -rl 'github.com/Amirreza-Zeraati/go-boilerplate' . | xargs sed -i 's|github.com/Amirreza-Zeraati/go-boilerplate|github.com/YOURNAME/YOURREPO|g'

# 2. Dependencies
go mod tidy

# 3. Config
cp .env.example .env    # edit DB/Redis creds

# 4. Start Postgres + Redis (locally or via containers), then:
make run
```

On startup the app loads config, connects to Postgres (retrying), **applies migrations up to the newest version** (when `DB_AUTO_MIGRATE=true`), connects to Redis, wires all layers, and serves HTTP. Ctrl-C triggers graceful shutdown.

## Migrations

Two ways to run them, both using the same files in `migrations/`:

- **Automatic (default):** on startup the app runs all pending migrations via the embedded runner. Controlled by `DB_AUTO_MIGRATE`. In production you may prefer to set this `false` and run migrations as a separate deploy step.
- **Manual CLI:** `make migrate-up` / `make migrate-down` (needs [golang-migrate](https://github.com/golang-migrate/migrate) installed). Create a new one with `make migrate-create name=add_orders`.

Either way, "up" advances the schema to the latest version and no-ops if already current.

## API

```
GET  /healthz                     liveness  (process up)
GET  /readyz                      readiness (pings Postgres + Redis)

POST /api/v1/auth/register        { email, password }         -> 201 user
POST /api/v1/auth/login           { email, password }         -> 200 user + sets session cookie
POST /api/v1/auth/logout          (session required)          -> 200
GET  /api/v1/auth/me              (session required)          -> 200 user
GET  /api/v1/admin/ping           (session + role=admin)      -> 200   [example RBAC route]
```

Responses use a consistent envelope: `{"data": ...}` on success, `{"error": {"message", "fields"}}` on failure.

### Auth model (sessions, not JWT)

Login verifies the password (bcrypt), creates a session in Redis, and returns an **opaque, random session ID** in an HttpOnly cookie. Every authenticated request looks the session up in Redis and refreshes its TTL (sliding expiration). The user's role is cached in the session, so authorization needs no DB hit per request. Logout deletes the session server-side — instant revocation, which JWTs can't do without extra machinery.

To make an admin: register a user, then set their role in the DB (`UPDATE users SET role='admin' WHERE email=...`), and log in again.

## Notes

- `SESSION_COOKIE_SECURE=true` in production (HTTPS). With `SameSite=None` for cross-site clients, Secure is mandatory.
- Rate limiting is a per-IP fixed window backed by Redis and **fails open** — if Redis blips, requests aren't blocked.
- The example `User`, admin route, and repository are meant to be replaced/extended with your real domain.

## Next phases (from the full plan)

Prometheus metrics → OpenAPI/Swagger → Air hot reload → unit tests → Docker & Compose → GitHub Actions CI → Kubernetes → RabbitMQ.

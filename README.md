# go-boilerplate

A reusable Go backend template: **Gin + GORM + PostgreSQL + Redis**, with server-side session auth, role-based authorization, centralized error handling, Prometheus metrics, and a test suite that runs without any external services.

## Features

- Config loader (env + `.env`), structured logging (`log/slog`)
- PostgreSQL via GORM with connection pooling
- **Embedded migrations** (golang-migrate) — applied on startup, no CLI needed
- Repository pattern + service layer (handler → service → repository)
- **Session-based auth** (server-side sessions in Redis, opaque cookie — not JWT)
- **Role-based authorization** middleware
- **Centralized error handling** (`internal/apperr`) — one error type carries status, code, and safe message
- **Prometheus metrics** at `/metrics`, including dependency health gauges
- Middleware chain: request ID, recovery, structured logging, metrics, CORS, Redis rate limiting
- Health (`/healthz`) and readiness (`/readyz`) endpoints
- Graceful shutdown
- **Hot reload** via Air, **test suite** with no Docker/DB required

## Layout

```
cmd/api/main.go              # composition root: wires everything, handles shutdown
internal/
  apperr/                    # the application error type (status + code + message + cause)
  config/                    # typed config from env; DSN/URL/Addr builders
  database/  redis/          # connection lifecycle wrappers
  session/                   # server-side sessions in Redis
  models/                    # Base (uuid, timestamps, soft delete) + User
  repository/                # interfaces + GORM implementations
  service/                   # business logic
  dto/                       # request/response shapes + validation tags
  response/                  # JSON envelope; maps apperr -> HTTP
  middleware/                # request id, logger, recovery, metrics, cors, ratelimit, auth, rbac
  metrics/                   # Prometheus collectors (private registry)
  handler/                   # HTTP handlers
  routes/                    # route registration, one file per domain
  server/                    # gin engine + http.Server with graceful shutdown
  migrate/                   # embedded-migration runner
migrations/                  # *.sql + embed.go
pkg/logger/                  # slog setup
```

## Getting started

```bash
grep -rl 'github.com/Amirreza-Zeraati/go-boilerplate' . \
  | xargs sed -i 's|github.com/Amirreza-Zeraati/go-boilerplate|github.com/YOU/YOURREPO|g'

go mod tidy
cp .env.example .env      # edit DB/Redis creds
make run                  # or: make tools && make dev   (hot reload)
```

## Error handling

Every layer returns `*apperr.Error`, which carries its own HTTP status, a stable machine-readable code, a client-safe message, optional per-field messages, and a wrapped cause.

```go
// service
return nil, apperr.Internal("could not load user").Wrap(dbErr)

// handler — no status mapping, no error switch
if err != nil {
    response.Fail(c, err)
    return
}
```

The client sees the safe message; the wrapped `dbErr` goes to the logs and never crosses the wire. Unknown (non-`apperr`) errors become a generic 500 with the original preserved as the cause, so a raw driver error can't leak.

Error responses are uniform:

```json
{ "error": { "code": "validation_failed", "message": "validation failed",
             "fields": { "Email": "must be a valid email" } } }
```

Available codes: `validation_failed`, `invalid_input`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `rate_limited`, `internal_error`, `unavailable`.

## Metrics

`GET /metrics` exposes Go runtime and process metrics plus:

| Metric | Type | Labels |
|---|---|---|
| `http_requests_total` | counter | method, route, status |
| `http_request_duration_seconds` | histogram | method, route |
| `http_requests_in_flight` | gauge | — |
| `dependency_up` | gauge | dependency |

The `route` label is the registered pattern (`/api/v1/users/:id`), never the concrete URL — using raw paths would create a time series per ID and blow up cardinality. `dependency_up` is refreshed by `/readyz`, so the check that gates traffic is the one Prometheus alerts on.

Rate limiting applies to `/api/v1` only, so probes and scrapes are never throttled.

## Testing

```bash
make test          # all tests
make test-race     # with the race detector
make test-cover    # coverage summary + coverage.html
```

No database, Redis, or Docker required: services are tested against fake repositories (the payoff of the repository interface), middleware and handlers against `httptest`, and the session store against in-process miniredis.

| Package | Covers |
|---|---|
| `apperr` | codes/statuses, wrapping, copy-on-write, safe conversion of unknown errors |
| `service` | register/authenticate rules, password hashing, no user enumeration |
| `middleware` | session auth, cookie clearing, RBAC, metric label cardinality |
| `handler` | validation, error→status mapping, HttpOnly cookie, no hash leakage |
| `session` | create/get/delete, TTL expiry and sliding refresh |
| `config` | DSN/URL builders, defaults, env overrides |

## API

```
GET  /healthz                     liveness
GET  /readyz                      readiness (pings Postgres + Redis)
GET  /metrics                     Prometheus scrape

POST /api/v1/auth/register        { email, password }    -> 201 user
POST /api/v1/auth/login           { email, password }    -> 200 user + session cookie
POST /api/v1/auth/logout          (session required)     -> 200
GET  /api/v1/auth/me              (session required)     -> 200 user
GET  /api/v1/admin/ping           (session + role=admin) -> 200
```

## Adding a resource

Copy the `User` vertical slice: migration → model → repository interface + impl → service → dto → handler → a `registerX` function in `internal/routes/`, then wire it in `main.go`. Return `apperr` values from the service and the HTTP layer needs no changes.

## Next phases

OpenAPI/Swagger → Docker & Compose → GitHub Actions CI → Kubernetes → RabbitMQ.

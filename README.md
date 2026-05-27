# CodeArena

A full-stack competitive coding platform in the style of LeetCode. Candidates browse problems, write solutions in a Monaco-powered editor, submit, and receive real-time verdicts over WebSockets. An Angular admin panel lets operators author problems, manage test cases, and review submission logs — all backed by the same Go API.

**Live:**
- Candidate UI → https://codearena.alexmcgarity.dev
- Admin panel → https://codearena-admin.alexmcgarity.dev
- API → https://codearena-api.fly.dev/healthz

---

## Tech Stack

| Layer | Technology | Why |
|---|---|---|
| Candidate UI | React 18 (Monaco editor, Zustand, React Query) | Component-driven; Monaco is the VS Code editor engine — native syntax highlighting with no configuration |
| Admin panel | Angular 21 (standalone components, signals, reactive forms) | Form-heavy CRUD dashboard; Angular's reactive forms enforce validation structure; signals replace RxJS for local state |
| API server | Go (stdlib `net/http`, pgx, gorilla/websocket) | Goroutines handle concurrent judge jobs and WebSocket connections with near-zero overhead; no framework tax |
| Code judge | C++17 (fork/exec, setrlimit, nlohmann/json) | Low-level process control is the right tool for sandboxed execution; `setrlimit` enforces CPU and memory limits at the OS level |
| Database | PostgreSQL | Relational model fits `problems → test_cases → submissions`; pgx gives direct SQL control with no ORM overhead |
| Auth | JWT HS256 (golang-jwt/jwt) | Stateless; both frontends consume the same token; role claim gates admin routes |
| Hosting | Fly.io (API + judge) · Vercel (React + Angular) | Fly.io runs the Docker container with the C++ binary; Vercel handles static deployments with zero config |

---

## Architecture

```
┌─────────────────────────────────┐
│      Candidate UI (React)       │  Monaco editor · Zustand auth
│   codearena.alexmcgarity.dev    │  React Query · WebSocket client
└──────────────┬──────────────────┘
               │ HTTPS / WSS
┌──────────────▼──────────────────┐
│       API Server (Go)           │  net/http · JWT middleware
│   codearena-api.fly.dev         │  WebSocket hub · async judge dispatch
└──────┬──────────────┬───────────┘
       │ JSON stdin   │ SQL (pgx)
┌──────▼──────┐  ┌────▼──────────┐
│  C++ Judge  │  │  PostgreSQL   │
│  fork/exec  │  │  problems     │
│  setrlimit  │  │  test_cases   │
│  SIGKILL    │  │  submissions  │
└─────────────┘  │  users        │
                 └───────────────┘
┌─────────────────────────────────┐
│      Admin Panel (Angular)      │  Standalone components · signals
│   codearena-admin.alexmcgarity.dev │  Reactive forms · HTTP interceptor
└─────────────────────────────────┘
```

---

## Features

### Candidate UI
- Problem list with difficulty badges
- Monaco editor — syntax highlighting for C++ and Python 3
- Submit → real-time result via WebSocket (no polling)
- JWT auth: register, login, per-user submission history
- React Query for server state; Zustand for auth state

### C++ Judge
- Accepts a JSON payload: `{ language, code, test_cases[], time_limit_ms, memory_limit_mb }`
- **C++**: `g++ -O2 -std=c++17` compile → binary executed per test case
- **Python 3**: source written to temp file → `python3` executed per test case
- **Linux sandbox**: `fork`/`exec` with `setrlimit(RLIMIT_CPU)` + `setrlimit(RLIMIT_AS)` in the child; parent polls with `waitpid` and sends `SIGKILL` on wall-clock timeout
- Verdicts: `accepted` · `wrong_answer` · `compile_error` · `time_limit_exceeded` · `unsupported_language`

### API Server
- `POST /submissions` — accepts code, enqueues async goroutine, returns submission ID
- `GET /submissions/:id/live` — upgrades to WebSocket; subscribe-then-check pattern avoids the race between judge completion and client connection
- `POST /auth/register` · `POST /auth/login` — bcrypt hash, JWT issuance
- `/admin/*` routes gated by `requireAdmin` middleware (role claim in JWT)
- Dual store: PostgreSQL primary with automatic in-memory fallback for dev
- Migration runner applies `db/migrations/*.sql` on startup, tracking applied files in `schema_migrations`

### Admin Panel
- Problem authoring form with live Markdown preview (via `marked`)
- Inline test case manager — add, edit, delete input/expected pairs
- Submission log — all submissions across all users with verdict colour-coding
- Angular signals for local component state; functional HTTP interceptor attaches Bearer token

---

## Project Structure

```
CodeArena/
├── judge/               # C++ judge binary
│   ├── src/main.cpp     # fork/exec sandbox, JSON protocol, C++/Python dispatch
│   └── CMakeLists.txt   # FetchContent → nlohmann/json (header-only)
├── api/                 # Go API server
│   ├── cmd/server/      # main.go entry point
│   └── internal/
│       ├── auth/        # JWT sign/verify (HS256)
│       └── server/      # handlers, hub, store, migrations, mock judge
├── web/                 # React candidate UI (Vite)
│   └── src/
│       ├── pages/       # Problems, Login, Register, Profile
│       └── store/       # Zustand auth store
├── admin/               # Angular admin panel
│   └── src/app/
│       ├── core/        # AuthService (signals), interceptor, guard
│       └── pages/       # Login, Problems, ProblemEdit, Users, Submissions
├── db/
│   └── migrations/      # 0001_init → 0005_test_cases
├── Dockerfile           # Three-stage: judge build · Go build · runtime
└── fly.toml             # Fly.io deployment config
```

---

## Local Development

### Prerequisites
- Go 1.25+
- Node.js 20+
- Docker (for Postgres) or a local PostgreSQL instance
- CMake 3.20+ and g++ (to build the judge locally)

### 1. Start the database

```bash
docker compose up -d db
```

### 2. Run the API

```bash
cd api
USE_MEMORY_STORE=true go run ./cmd/server   # in-memory, no DB needed
# OR with Postgres:
DATABASE_URL=postgres://postgres:postgres@localhost:5432/codearena?sslmode=disable \
  JUDGE_MOCK=true go run ./cmd/server
```

`JUDGE_MOCK=true` skips the real judge binary — useful when the C++ build isn't set up.

### 3. Run the React UI

```bash
cd web
npm install
npm run dev     # http://localhost:5173
```

### 4. Run the Angular admin panel

```bash
cd admin
npm install
npm run start   # http://localhost:4200
```

### 5. Build the judge (optional)

```bash
cd judge
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j$(nproc)
# binary at judge/build/codearena-judge
JUDGE_BINARY_PATH=./judge/build/codearena-judge go run ./cmd/server
```

### Environment variables (API)

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/codearena?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET` | `dev-secret-change-in-production` | HS256 signing key |
| `JUDGE_BINARY_PATH` | `./codearena-judge` | Path to compiled judge binary |
| `JUDGE_MOCK` | — | Set to `true` to use heuristic mock instead of real judge |
| `USE_MEMORY_STORE` | — | Set to `true` to skip PostgreSQL entirely |
| `MIGRATIONS_DIR` | `./migrations` | Directory containing `*.sql` migration files |
| `SEED_ADMIN_EMAIL` | — | If set, upserts this user as admin on startup |
| `SEED_ADMIN_PASSWORD` | — | Password for the seeded admin user |

---

## Tests

```bash
cd api
go test ./...         # 112 tests
```

Coverage spans:
- JWT sign/verify, expired tokens, tampered signatures
- WebSocket hub: subscribe, notify, buffered early-notify, concurrent stress
- HTTP handlers: all routes, content-type, CORS, auth middleware
- Store: full CRUD for problems, test cases, users, submissions; concurrent insert/update
- End-to-end submission flow: goroutine dispatches judge, verdict written to store and pushed to hub

---

## Deployment

The API and judge run in a single Docker container on Fly.io. The multi-stage `Dockerfile` compiles the C++ judge in an Ubuntu build stage and the Go binary in a golang stage, then copies both into a minimal runtime image with `g++` and `python3` available for executing submissions.

```bash
# API + judge
fly deploy --app codearena-api

# React
cd web && vercel --prod

# Angular
cd admin && vercel --prod
```

# CodeArena

Competitive coding platform (mini LeetCode). Full-stack resume project targeting Microsoft SWE II.
See SPEC.md for full architecture, feature breakdown, and phased plan.

## Tech Stack

| Layer | Tech |
|---|---|
| Candidate UI | React (Monaco editor, Zustand, React Query) |
| Admin panel | Angular (reactive forms, NgRx if needed) |
| API server | Go (net/http or Chi router, pgx, gorilla/websocket) |
| Code judge | C++ (fork/exec, seccomp sandbox, rlimit) |
| Database | PostgreSQL |
| Auth | JWT (Go issues, both frontends consume) |

## Repository Layout

```
CodeArena/
├── judge/      # C++ judge — CMake build
├── api/        # Go API server
├── web/        # React candidate UI
├── admin/      # Angular admin panel
├── db/
│   └── migrations/
└── docker-compose.yml
```

## Development Phases

1. **MVP** ✅ — Go API + C++ judge + React UI, end-to-end submission working
2. **Real-time + Auth** ✅ — WebSockets, JWT, submission history
3. **Admin panel** ✅ — Angular CRUD for problems and test cases
4. **Polish** ✅ — multi-language judge, deploy to Fly.io + Vercel, add to Vibe-Website

## Current Progress (as of 2026-05-27) — ALL PHASES COMPLETE

### Phase 1 — Complete

**Go API** (`api/`):
- Routes: `GET /problems`, `GET /problems/:id`, `POST /submissions`, `GET /submissions/:id`, `GET /healthz`
- Dual store: PostgreSQL via pgx (primary) with automatic in-memory fallback (`USE_MEMORY_STORE=true` to force)
- Async goroutine dispatch to judge with 20 s timeout; verdict written back to store
- CORS + logging middleware; no router dependency (stdlib `net/http`)
- Mock judge mode via `JUDGE_MOCK=true` env var (heuristic: detects `Hello World` + `cout`)

**C++ judge** (`judge/`):
- Reads source from stdin, compiles with `g++ -O2 -std=c++17`, runs against hardcoded test cases
- Returns JSON verdict: `accepted` / `wrong_answer` / `compile_error`
- CMake build; invoked as a subprocess by the Go API via `JUDGE_BINARY_PATH`
- **No sandboxing yet** — seccomp/rlimit not implemented

**React UI** (`web/`):
- Problem list sidebar + problem detail view
- Plain `<textarea>` code editor (Monaco not yet integrated)
- Only C++ selectable in language dropdown

**Database** (`db/migrations/0001_init.sql`):
- `problems` table (id, title, description, difficulty, tags[], created_at)
- `submissions` table (id, problem_id, language, code, verdict, created_at)
- Seed data loaded at API startup via `ON CONFLICT DO NOTHING`
- **No test_cases table yet** — judge uses hardcoded cases

**Docker Compose**: API + Postgres service, `db-data` volume

**Tests**: 46 passing at end of Phase 1. Grew to 112 across all phases (see Phase 4 for final breakdown).

### Phase 2 — Complete

**Go API additions**:
- `POST /auth/register` — bcrypt hash, issues JWT (24 h, HS256); `JWT_SECRET` env var
- `POST /auth/login` — verifies bcrypt, issues JWT
- `GET /users/me/submissions` — returns submission history for the authenticated user
- `GET /submissions/:id/live` — upgrades to WebSocket; subscribe-then-check pattern avoids race between judge finishing and client connecting; 60 s timeout
- `requireAuth` middleware — extracts and verifies Bearer JWT, injects claims into context
- WebSocket hub (`Hub`) — maps submission ID → buffered channels; `Notify` is non-blocking (select/default) so a missing client never blocks the judge goroutine
- `Authorization` header added to CORS allowed headers
- `POST /submissions` now requires auth; stores `user_id` with each submission
- New deps: `gorilla/websocket v1.5.3`, `golang-jwt/jwt/v5 v5.3.1`, `golang.org/x/crypto`

**Database**:
- `0002_users.sql` — `users` table (id, email, password_hash, created_at)
- `0003_submissions_user.sql` — `user_id INTEGER REFERENCES users(id)` added to submissions

**React UI additions** (`web/`):
- Zustand auth store (`src/store/auth.js`) — register/login/logout, JWT persisted in `localStorage`
- React Router v6 — `/`, `/login`, `/register`, `/profile`
- Top nav bar — shows email + logout when authenticated, login/register links otherwise
- `Login.jsx` / `Register.jsx` — auth forms with inline error display
- `Problems.jsx` — polling removed; WebSocket opens on submit result, redirects to `/login` if 401
- `Profile.jsx` — submission history table via React Query (`GET /users/me/submissions`)
- New deps: `react-router-dom`, `zustand`, `@tanstack/react-query`

### Phase 3 — Complete

**Go API additions**:
- `0004_admin_role.sql` — `role TEXT NOT NULL DEFAULT 'user'` on users; `time_limit_ms` and `memory_limit_mb` columns on problems
- `0005_test_cases.sql` — `test_cases` table (id, problem_id, input, expected, created_at); `ON DELETE CASCADE` from problems
- JWT `Claims` now carries `Role`; login response includes `role` field
- `requireAdmin` middleware — wraps `requireAuth`, checks `role = 'admin'`, returns 403 otherwise
- `SeedAdmin` on both stores — upserts an admin user at startup when `SEED_ADMIN_EMAIL` + `SEED_ADMIN_PASSWORD` env vars are set
- Admin routes (all behind `requireAdmin`):
  - `GET|POST /admin/problems` — list all / create
  - `GET|PUT|DELETE /admin/problems/:id` — read / update / delete
  - `GET|POST /admin/problems/:id/testcases` — list / create test cases for a problem
  - `PUT|DELETE /admin/testcases/:id` — update / delete individual test case
  - `GET /admin/users` — full user list with roles
  - `GET /admin/submissions` — all submissions joined to problem + user
- `Problem` type extended with `TimeLimitMs` and `MemoryLimitMb`; seeded problems updated to include defaults
- 15 new tests (admin CRUD, middleware role enforcement); total Go test count: 61

**Angular admin panel** (`admin/`) — Angular 21, standalone components, signals, lazy-loaded routes:
- `core/auth.service.ts` — signals for `token`, `user`, `isLoggedIn`, `isAdmin`; rejects non-admin logins
- `core/auth.interceptor.ts` — functional `HttpInterceptorFn` that attaches `Authorization: Bearer <token>`
- `core/auth.guard.ts` — `CanActivateFn`, redirects to `/login` if not admin
- `pages/login/` — reactive form, validates admin role before accepting the session
- `pages/problems/` — sortable table with difficulty badges; create / delete actions
- `pages/problem-edit/` — reactive form with live markdown preview (`marked`); inline test case manager (add, save edits, delete per row); navigates to edit page after create
- `pages/users/` — user table with admin/user role badges
- `pages/submissions/` — full submission log with verdict color-coding and formatted timestamps
- Sidebar shell with `routerLinkActive` highlighting and logout
- New dep: `marked` (markdown → HTML for description preview)

### Phase 4 — Complete

**C++ judge rewrite** (`judge/src/main.cpp`):
- Reads a JSON payload from stdin: `{ language, code, test_cases[], time_limit_ms, memory_limit_mb }`
- Dynamic test cases — judge no longer has hardcoded cases; Go API fetches them from the DB and sends them
- C++ path: `g++ -O2 -std=c++17` compile → run binary per test case with temp files in `/tmp`
- Python 3 path: write `.py` → `python3 submission.py` per test case
- Linux sandbox: `fork`/`exec` with `setrlimit(RLIMIT_CPU)` + `setrlimit(RLIMIT_AS)` in child; `SIGKILL` + poll-wait for wall-clock timeout in parent
- Non-Linux fallback (Windows dev) uses `popen` without rlimit
- `judge/CMakeLists.txt` — adds nlohmann/json v3.11.3 via `FetchContent` (header-only, no runtime deps)
- Verdicts: `accepted` / `wrong_answer` / `compile_error` / `time_limit_exceeded` / `unsupported_language` / `judge_error`

**Go API — judge integration**:
- `runJudge` now accepts `testCases []TestCase`, `timeLimitMs`, `memLimitMb`; builds JSON payload and pipes it to the judge binary
- Submission goroutine fetches problem limits + test cases from store before calling judge; returns `no_test_cases` if none configured
- Mock judge updated to handle both `cpp` and `python` heuristics

**React candidate UI**:
- `@monaco-editor/react` replaces `<textarea>` — VS-dark theme, no minimap, auto-layout, font size 14
- Python 3 added to language dropdown with its own default starter code
- Verdict result card has coloured left-border per verdict type
- API URLs use `import.meta.env.VITE_API_URL` / `VITE_WS_URL` with `localhost:8080` fallback for dev

**Migration runner** (`api/internal/server/migrations.go`):
- Reads `*.sql` files from `./migrations/` in alphabetical order at startup
- Tracks applied files in `schema_migrations` table; idempotent across restarts
- `MIGRATIONS_DIR` env var overrides the default path

**Deployment**:
- `Dockerfile` — three-stage build: Ubuntu+CMake (judge) → golang:1.25 (API) → Ubuntu runtime with g++ + python3
- `fly.toml` — Fly.io config for `iad` region, 512 MB shared VM, auto-stop on idle
- `web/vercel.json` / `admin/vercel.json` — SPA rewrites; Angular points to `dist/admin/browser`
- Angular `src/environments/` — `environment.ts` (localhost) / `environment.prod.ts` (Fly.io URL), wired via `fileReplacements` in `angular.json`
- Git repo initialized; all code committed

**Live URLs**:
- API: `https://codearena-api.fly.dev` (`/healthz` → `{"status":"ok"}`)
- React: `https://web-g128hh1ip-alexmcgaritys-projects.vercel.app`
- Admin: `https://admin-m10ft84rc-alexmcgaritys-projects.vercel.app`
- Admin credentials: `amcgarity123@gmail.com` / `CodeArena2026!`

**Tests**: 112 total, all passing
- `internal/auth/jwt_test.go` — 8 tests: round-trip, expired, tampered signature, wrong secret, env var
- `internal/server/hub_test.go` — 9 tests: subscribe/notify, buffered early-notify, multiple subscribers, unsubscribe isolation, cleanup, concurrent stress, independent IDs
- `internal/server/inmemory_test.go` — 42 tests: all store methods, SeedAdmin (create + promote + normalise), problem CRUD, test case CRUD, list views, concurrent verdict updates, concurrent user creation
- `internal/server/server_test.go` — 47 tests: all handler paths, content-type, CORS auth header, full submission flow via hub (3 integration tests), admin edge cases, auth missing fields, method-not-allowed coverage
- `internal/server/judge_test.go` — 6 tests: mock verdicts for C++ and Python, language aliases, unsupported language

**Vibe-Website**: CodeArena added as full project card in `projects.html`, replacing "Project 4" placeholder

**Pending** (one-time action):
- Add credit card at `fly.io/dashboard` to remove the 5-minute trial machine limit (hobby tier is free)

## Conventions

- Go: standard library preferred, no heavy frameworks; use `internal/` for all non-exported packages
- React: functional components only, no class components
- Angular: standalone components (no NgModules unless necessary), use signals over RxJS where possible
- C++ judge: C++17, CMake, keep it a single binary with no runtime dependencies
- SQL: raw migrations in `db/migrations/`, no ORM — use `pgx` directly in Go
- No monorepo tooling — each subdirectory is its own project with its own build system

## Key Design Decisions

- React and Angular share the same Go API — no separate BFF layers
- Judge communicates with API via subprocess call (`exec.Command`); Go API passes source on stdin, reads JSON verdict from stdout
- WebSocket connection is per-submission, not a persistent global channel; hub uses subscribe-then-check to handle the race between judge completion and client connection
- Admin routes are the same Go server, separated by `/admin/` prefix and middleware role check
- JWT is HS256, 24 h expiry, secret from `JWT_SECRET` env var (falls back to a dev default)

## Hosting

All services are live.

| Service | URL |
|---|---|
| Go API + judge | `https://codearena-api.fly.dev` (Fly.io, `iad`) |
| React candidate UI | `https://web-g128hh1ip-alexmcgaritys-projects.vercel.app` |
| Angular admin panel | `https://admin-m10ft84rc-alexmcgaritys-projects.vercel.app` |
| Database | Fly.io managed Postgres (`codearena-db`, `iad`) |

Linked from Vibe-Website (`../Vibe-Website/projects.html`).

### Re-deploying

```bash
# API + judge (from repo root)
fly deploy --app codearena-api

# React (from web/)
cd web && vercel --prod

# Angular (from admin/)
cd admin && vercel --prod
```

### Fly.io secrets reference

| Secret | Purpose |
|---|---|
| `DATABASE_URL` | Auto-set by `fly postgres attach` |
| `JWT_SECRET` | HS256 signing key (48-char random) |
| `SEED_ADMIN_EMAIL` | Admin user email bootstrapped at startup |
| `SEED_ADMIN_PASSWORD` | Admin user password bootstrapped at startup |

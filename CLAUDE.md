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
4. **Polish** — multi-language judge, deploy to Fly.io + Vercel, add to Vibe-Website

## Current Progress (as of 2026-05-27)

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

**Tests** (`api/internal/server/`): 46 passing — handler tests (httptest), store unit tests, mock judge tests. Updated to 61 tests after Phase 3.

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

### Phase 4 — Not started
- Wire judge to `test_cases` table per problem (currently uses hardcoded cases in judge binary)
- Seccomp/rlimit sandboxing in judge
- Multi-language support (Python)
- Monaco editor in React candidate UI (currently plain `<textarea>`)
- Deploy Go API + judge to Fly.io or Railway; React + Angular to Vercel
- Link from Vibe-Website projects page

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

## Hosting Target

Vibe-Website (existing project at `../Vibe-Website`). CodeArena will be linked from the projects page once deployed.
- Go API + judge: Fly.io or Railway
- React: Vercel
- Angular: Vercel or same host as React

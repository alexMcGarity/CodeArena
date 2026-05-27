# CodeArena — Project Specification

**Status:** Pre-development  
**Target hosting:** Vibe-Website  
**Goal:** Full-stack resume project demonstrating React, Angular, Go, and C++ across a defensible, production-realistic architecture.

---

## Overview

CodeArena is a competitive coding platform in the style of LeetCode or HackerRank. Users browse problems, write code in a browser editor, submit their solution, and receive real-time feedback on whether it passed. An admin panel lets an operator author problems and manage users.

The platform is intentionally scoped to be completable by one developer while showcasing four distinct technologies — each chosen for a reason that can be defended in a technical interview.

---

## Architecture

```
┌──────────────────────────────────────┐
│          Candidate UI (React)        │  ← problem browser, code editor, live results
└────────────────┬─────────────────────┘
                 │ HTTP / WebSocket
┌────────────────▼─────────────────────┐
│          API Server (Go)             │  ← REST endpoints, submission queue, WS hub
└──────┬──────────────────────┬────────┘
       │ gRPC / subprocess    │ SQL
┌──────▼──────┐        ┌──────▼──────┐
│  C++ Judge  │        │  Postgres   │
│  (sandbox)  │        │  (problems, │
│             │        │   users,    │
└─────────────┘        │   results)  │
                       └─────────────┘
┌──────────────────────────────────────┐
│          Admin Panel (Angular)       │  ← problem authoring, user management
└──────────────────────────────────────┘
```

---

## Tech Stack & Justifications

| Layer | Technology | Why |
|---|---|---|
| Candidate UI | **React** | Component-driven, fast iteration, integrates cleanly with Monaco editor (same engine as VS Code) |
| Admin panel | **Angular** | Form-heavy CRUD dashboard — Angular's reactive forms and strict structure are a natural fit; also separates concerns from the candidate-facing app |
| API server | **Go** | Goroutines handle concurrent submission queues and WebSocket connections with low overhead; fast cold starts for cloud deployment |
| Code judge | **C++** | Sandboxed code execution requires low-level process control (`fork`/`exec`, `seccomp`, resource limits via `rlimit`); no managed runtime overhead |
| Database | **PostgreSQL** | Relational model fits problems → test cases → submissions; Go's `pgx` driver is excellent |
| Realtime | **WebSockets** (Go → React) | Push submission results to the candidate UI without polling |

---

## Features

### Candidate UI (React)
- Problem list with difficulty filter (Easy / Medium / Hard) and tag filter
- Problem detail page: description, examples, constraints
- Monaco code editor with language selector (C++, Python, Go — v1 can start with just C++)
- Submit button → spinner → real-time result card (Accepted / Wrong Answer / Time Limit / Runtime Error)
- Basic auth (JWT): register, login, profile page with submission history

### Admin Panel (Angular)
- Login-gated (separate admin role)
- Problem CRUD: title, description (markdown), difficulty, tags, time/memory limits
- Test case manager: add/edit/delete input-output pairs per problem
- User list with basic stats (submission count, pass rate)
- Submission log viewer

### API Server (Go)
- `POST /submissions` — accepts code + language, enqueues job, returns submission ID
- `GET /submissions/:id` — poll status (used as fallback)
- `WS /submissions/:id/live` — push result when judge finishes
- `GET /problems`, `GET /problems/:id` — problem listing and detail
- `POST /auth/register`, `POST /auth/login` — JWT issuance
- Admin routes behind middleware: `POST/PUT/DELETE /admin/problems`, etc.

### C++ Judge
- Receives submission payload (source code, language, problem ID)
- Writes source to temp file, compiles with timeout
- Runs compiled binary against each test case in an isolated subprocess
- Enforces wall time limit and memory limit via OS primitives
- Returns per-test-case result and aggregate verdict

---

## Phased Implementation Plan

### Phase 1 — Core loop (MVP)
1. Go API: problem CRUD endpoints + Postgres schema
2. C++ judge: compile and run a single C++ submission against hardcoded test cases
3. React UI: problem list + detail + editor + submit → poll result

**Milestone:** Submit "Hello World" and get Accepted/Wrong Answer back end-to-end.

### Phase 2 — Real-time + Auth
1. WebSocket hub in Go, push judge results to React
2. JWT auth in Go, login/register in React
3. Submission history stored in Postgres, visible on profile page

### Phase 3 — Admin panel
1. Angular project scaffold, connect to Go admin routes
2. Problem authoring form (markdown preview)
3. Test case manager
4. User and submission log views

### Phase 4 — Polish & hosting
1. Multi-language support in judge (add Python via subprocess)
2. Deploy Go API + judge to Fly.io or Railway
3. Deploy React to Vercel, Angular to Vercel or same host
4. Add CodeArena to Vibe-Website projects page with live link

---

## Repository Structure (planned)

```
CodeArena/
├── judge/          # C++ judge binary
│   ├── src/
│   └── CMakeLists.txt
├── api/            # Go API server
│   ├── cmd/
│   ├── internal/
│   └── go.mod
├── web/            # React candidate UI
│   ├── src/
│   └── package.json
├── admin/          # Angular admin panel
│   ├── src/
│   └── package.json
├── db/
│   └── migrations/ # SQL migration files
├── docker-compose.yml
└── SPEC.md
```

---

## Resume Talking Points

- "Designed a multi-frontend architecture where React serves the latency-sensitive candidate experience and Angular handles the form-heavy admin workflow — both backed by the same Go API."
- "Built a sandboxed C++ judge using `fork`/`exec` and `seccomp` filters to safely execute untrusted user code with enforced time and memory limits."
- "Used Go's goroutine model to fan out concurrent judge jobs and push results to clients over WebSockets without blocking the HTTP server."
- "Chose PostgreSQL with normalized test-case storage so adding a new problem requires no schema changes."

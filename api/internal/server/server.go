package server

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/codearena/api/internal/auth"
)

type contextKey string

const claimsKey contextKey = "claims"

type Problem struct {
    ID            int      `json:"id"`
    Title         string   `json:"title"`
    Difficulty    string   `json:"difficulty"`
    Description   string   `json:"description"`
    Tags          []string `json:"tags"`
    TimeLimitMs   int      `json:"time_limit_ms"`
    MemoryLimitMb int      `json:"memory_limit_mb"`
}

type User struct {
    ID           int    `json:"id"`
    Email        string `json:"email"`
    Role         string `json:"role"`
    PasswordHash string `json:"-"`
}

type TestCase struct {
    ID        int    `json:"id"`
    ProblemID int    `json:"problem_id"`
    Input     string `json:"input"`
    Expected  string `json:"expected"`
}

type UserSummary struct {
    ID    int    `json:"id"`
    Email string `json:"email"`
    Role  string `json:"role"`
}

type SubmissionLogItem struct {
    ID           int    `json:"submission_id"`
    ProblemTitle string `json:"problem_title"`
    UserEmail    string `json:"user_email"`
    Language     string `json:"language"`
    Verdict      string `json:"verdict"`
    CreatedAt    string `json:"created_at"`
}

type SubmissionRequest struct {
    ProblemID int    `json:"problem_id"`
    Code      string `json:"code"`
    Language  string `json:"language"`
}

type SubmissionResponse struct {
    SubmissionID int    `json:"submission_id"`
    Status       string `json:"status"`
}

type SubmissionStatus struct {
    SubmissionID int    `json:"submission_id"`
    Status       string `json:"status"`
    Verdict      string `json:"verdict"`
}

type SubmissionHistoryItem struct {
    ID           int    `json:"submission_id"`
    ProblemID    int    `json:"problem_id"`
    ProblemTitle string `json:"problem_title"`
    Language     string `json:"language"`
    Verdict      string `json:"verdict"`
    CreatedAt    string `json:"created_at"`
}

type Server struct {
    store Store
    hub   *Hub
}

var problems = []Problem{
    {
        ID:            1,
        Title:         "Hello World",
        Difficulty:    "Easy",
        Description:   "Write a program that prints `Hello World` to stdout.",
        Tags:          []string{"intro", "output"},
        TimeLimitMs:   2000,
        MemoryLimitMb: 256,
    },
    {
        ID:            2,
        Title:         "Sum of Two Numbers",
        Difficulty:    "Easy",
        Description:   "Read two integers from stdin and print their sum.",
        Tags:          []string{"math", "input-output"},
        TimeLimitMs:   2000,
        MemoryLimitMb: 256,
    },
}

func New() *http.Server {
    ctx := context.Background()
    var store Store

    useInMemory := os.Getenv("USE_MEMORY_STORE") == "true"
    if !useInMemory {
        db, err := connectDB(ctx)
        if err != nil {
            log.Printf("warning: failed to connect to database: %v, falling back to in-memory store", err)
            store = NewInMemoryStore()
        } else {
            migrationsDir := os.Getenv("MIGRATIONS_DIR")
            if migrationsDir == "" {
                migrationsDir = "./migrations"
            }
            if err := runMigrations(ctx, db, migrationsDir); err != nil {
                log.Printf("warning: migrations: %v", err)
            }
            if err := seedProblems(ctx, db); err != nil {
                log.Printf("warning: failed to seed problems: %v", err)
            }
            store = &PostgresStore{db: db}
        }
    } else {
        store = NewInMemoryStore()
    }

    s := &Server{store: store, hub: NewHub()}

    if adminEmail := os.Getenv("SEED_ADMIN_EMAIL"); adminEmail != "" {
        if err := s.store.SeedAdmin(ctx, adminEmail, os.Getenv("SEED_ADMIN_PASSWORD")); err != nil {
            log.Printf("warning: seed admin: %v", err)
        }
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", healthzHandler)
    mux.HandleFunc("/problems", s.problemsHandler)
    mux.HandleFunc("/problems/", s.problemDetailHandler)
    mux.HandleFunc("/submissions", requireAuth(s.submissionsHandler))
    mux.HandleFunc("/submissions/", s.submissionsMuxHandler)
    mux.HandleFunc("/auth/register", s.registerHandler)
    mux.HandleFunc("/auth/login", s.loginHandler)
    mux.HandleFunc("/users/me/submissions", requireAuth(s.userSubmissionsHandler))

    // admin routes
    mux.HandleFunc("/admin/problems", requireAdmin(s.adminProblemsMuxHandler))
    mux.HandleFunc("/admin/problems/", requireAdmin(s.adminProblemsDetailMuxHandler))
    mux.HandleFunc("/admin/testcases/", requireAdmin(s.adminTestCasesMuxHandler))
    mux.HandleFunc("/admin/users", requireAdmin(s.adminListUsersHandler))
    mux.HandleFunc("/admin/submissions", requireAdmin(s.adminListSubmissionsHandler))

    return &http.Server{
        Addr:    ":8080",
        Handler: loggingMiddleware(corsMiddleware(mux)),
    }
}

// --- middleware ---

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        header := r.Header.Get("Authorization")
        if !strings.HasPrefix(header, "Bearer ") {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        claims, err := auth.Verify(strings.TrimPrefix(header, "Bearer "))
        if err != nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        ctx := context.WithValue(r.Context(), claimsKey, claims)
        next(w, r.WithContext(ctx))
    }
}

func claimsFromContext(ctx context.Context) (*auth.Claims, bool) {
    c, ok := ctx.Value(claimsKey).(*auth.Claims)
    return c, ok
}

func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
    return requireAuth(func(w http.ResponseWriter, r *http.Request) {
        claims, _ := claimsFromContext(r.Context())
        if claims.Role != "admin" {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }
        next(w, r)
    })
}

// --- handlers ---

func healthzHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) problemsHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        methodNotAllowed(w)
        return
    }
    probs, err := s.store.FetchProblems(r.Context())
    if err != nil {
        http.Error(w, "failed to load problems", http.StatusInternalServerError)
        return
    }
    respondJSON(w, probs)
}

func (s *Server) problemDetailHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        methodNotAllowed(w)
        return
    }
    id, err := parseID(r.URL.Path, "/problems/")
    if err != nil {
        http.NotFound(w, r)
        return
    }
    problem, err := s.store.FetchProblemByID(r.Context(), id)
    if err != nil {
        if errors.Is(err, errNotFound) || err.Error() == "problem not found" {
            http.NotFound(w, r)
            return
        }
        http.Error(w, "failed to load problem", http.StatusInternalServerError)
        return
    }
    respondJSON(w, problem)
}

func (s *Server) submissionsHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        methodNotAllowed(w)
        return
    }
    claims, _ := claimsFromContext(r.Context())

    var req SubmissionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    if req.ProblemID == 0 || req.Code == "" || req.Language == "" {
        http.Error(w, "problem_id, code, and language are required", http.StatusBadRequest)
        return
    }

    submissionID, err := s.store.InsertSubmission(r.Context(), req, claims.UserID)
    if err != nil {
        http.Error(w, "failed to create submission", http.StatusInternalServerError)
        return
    }

    go func(id int, req SubmissionRequest) {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        verdict := "judge_error"

        problem, err := s.store.FetchProblemByID(ctx, req.ProblemID)
        if err != nil {
            log.Printf("fetch problem for submission %d: %v", id, err)
            goto done
        }

        {
            testCases, err := s.store.GetTestCasesByProblemID(ctx, req.ProblemID)
            if err != nil {
                log.Printf("fetch test cases for submission %d: %v", id, err)
                goto done
            }
            if len(testCases) == 0 {
                verdict = "no_test_cases"
                goto done
            }

            verdict, err = runJudge(ctx, req.Language, req.Code, testCases,
                problem.TimeLimitMs, problem.MemoryLimitMb)
            if err != nil {
                log.Printf("judge error for submission %d: %v", id, err)
                verdict = "judge_error"
            }
        }

    done:
        if err := s.store.UpdateSubmissionVerdict(context.Background(), id, verdict); err != nil {
            log.Printf("failed to update verdict for submission %d: %v", id, err)
        }
        s.hub.Notify(id, verdict)
    }(submissionID, req)

    w.WriteHeader(http.StatusAccepted)
    respondJSON(w, SubmissionResponse{SubmissionID: submissionID, Status: "queued"})
}

// submissionsMuxHandler dispatches /submissions/:id to the status handler
// and /submissions/:id/live to the WebSocket handler.
func (s *Server) submissionsMuxHandler(w http.ResponseWriter, r *http.Request) {
    if strings.HasSuffix(r.URL.Path, "/live") {
        s.submissionLiveHandler(w, r)
        return
    }
    s.submissionStatusHandler(w, r)
}

func (s *Server) submissionStatusHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        methodNotAllowed(w)
        return
    }
    id, err := parseID(r.URL.Path, "/submissions/")
    if err != nil {
        http.NotFound(w, r)
        return
    }
    status, err := s.store.GetSubmissionStatus(r.Context(), id)
    if err != nil {
        if err.Error() == "submission not found" {
            http.NotFound(w, r)
            return
        }
        http.Error(w, "failed to load submission status", http.StatusInternalServerError)
        return
    }
    respondJSON(w, status)
}

func (s *Server) userSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        methodNotAllowed(w)
        return
    }
    claims, _ := claimsFromContext(r.Context())
    items, err := s.store.GetSubmissionsByUserID(r.Context(), claims.UserID)
    if err != nil {
        http.Error(w, "failed to load submissions", http.StatusInternalServerError)
        return
    }
    respondJSON(w, items)
}

// --- helpers ---

var errNotFound = errors.New("not found")

func parseID(path string, prefix string) (int, error) {
    if !strings.HasPrefix(path, prefix) {
        return 0, fmt.Errorf("invalid prefix")
    }
    idText := strings.TrimPrefix(path, prefix)
    idText = strings.SplitN(idText, "/", 2)[0]
    idText = strings.Trim(idText, "/")
    if idText == "" {
        return 0, fmt.Errorf("missing id")
    }
    return strconv.Atoi(idText)
}

func respondJSON(w http.ResponseWriter, value interface{}) {
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(value); err != nil {
        http.Error(w, "failed to encode response", http.StatusInternalServerError)
    }
}

func methodNotAllowed(w http.ResponseWriter) {
    w.Header().Set("Allow", "GET, POST")
    http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        next.ServeHTTP(w, r)
    })
}

package server

import (
    "encoding/json"
    "net/http"
    "strings"
)

// problemRequest is the body for create/update problem.
type problemRequest struct {
    Title         string   `json:"title"`
    Description   string   `json:"description"`
    Difficulty    string   `json:"difficulty"`
    Tags          []string `json:"tags"`
    TimeLimitMs   int      `json:"time_limit_ms"`
    MemoryLimitMb int      `json:"memory_limit_mb"`
}

func (r *problemRequest) toProb(id int) Problem {
    p := Problem{
        ID:            id,
        Title:         r.Title,
        Description:   r.Description,
        Difficulty:    r.Difficulty,
        Tags:          r.Tags,
        TimeLimitMs:   r.TimeLimitMs,
        MemoryLimitMb: r.MemoryLimitMb,
    }
    if p.Tags == nil {
        p.Tags = []string{}
    }
    if p.TimeLimitMs == 0 {
        p.TimeLimitMs = 2000
    }
    if p.MemoryLimitMb == 0 {
        p.MemoryLimitMb = 256
    }
    return p
}

type testCaseRequest struct {
    Input    string `json:"input"`
    Expected string `json:"expected"`
}

// --- /admin/problems (no trailing slash) ---

func (s *Server) adminProblemsMuxHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        probs, err := s.store.FetchProblems(r.Context())
        if err != nil {
            http.Error(w, "failed to load problems", http.StatusInternalServerError)
            return
        }
        respondJSON(w, probs)
    case http.MethodPost:
        var req problemRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }
        if req.Title == "" || req.Description == "" || req.Difficulty == "" {
            http.Error(w, "title, description, and difficulty are required", http.StatusBadRequest)
            return
        }
        id, err := s.store.CreateProblem(r.Context(), req.toProb(0))
        if err != nil {
            http.Error(w, "failed to create problem", http.StatusInternalServerError)
            return
        }
        p, _ := s.store.FetchProblemByID(r.Context(), id)
        w.WriteHeader(http.StatusCreated)
        respondJSON(w, p)
    default:
        methodNotAllowed(w)
    }
}

// --- /admin/problems/:id  and  /admin/problems/:id/testcases ---

func (s *Server) adminProblemsDetailMuxHandler(w http.ResponseWriter, r *http.Request) {
    // /admin/problems/:id/testcases
    if strings.HasSuffix(r.URL.Path, "/testcases") {
        trimmed := strings.TrimSuffix(r.URL.Path, "/testcases")
        id, err := parseID(trimmed, "/admin/problems/")
        if err != nil {
            http.NotFound(w, r)
            return
        }
        s.adminTestCasesForProblemMux(w, r, id)
        return
    }

    // /admin/problems/:id
    id, err := parseID(r.URL.Path, "/admin/problems/")
    if err != nil {
        http.NotFound(w, r)
        return
    }

    switch r.Method {
    case http.MethodGet:
        p, err := s.store.FetchProblemByID(r.Context(), id)
        if err != nil {
            http.NotFound(w, r)
            return
        }
        respondJSON(w, p)
    case http.MethodPut:
        var req problemRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }
        if err := s.store.UpdateProblem(r.Context(), req.toProb(id)); err != nil {
            http.Error(w, "failed to update problem", http.StatusInternalServerError)
            return
        }
        p, _ := s.store.FetchProblemByID(r.Context(), id)
        respondJSON(w, p)
    case http.MethodDelete:
        if err := s.store.DeleteProblem(r.Context(), id); err != nil {
            http.Error(w, "failed to delete problem", http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusNoContent)
    default:
        methodNotAllowed(w)
    }
}

// adminTestCasesForProblemMux handles GET/POST /admin/problems/:id/testcases
func (s *Server) adminTestCasesForProblemMux(w http.ResponseWriter, r *http.Request, problemID int) {
    switch r.Method {
    case http.MethodGet:
        tcs, err := s.store.GetTestCasesByProblemID(r.Context(), problemID)
        if err != nil {
            http.Error(w, "failed to load test cases", http.StatusInternalServerError)
            return
        }
        if tcs == nil {
            tcs = []TestCase{}
        }
        respondJSON(w, tcs)
    case http.MethodPost:
        var req testCaseRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }
        if req.Expected == "" {
            http.Error(w, "expected is required", http.StatusBadRequest)
            return
        }
        id, err := s.store.CreateTestCase(r.Context(), problemID, req.Input, req.Expected)
        if err != nil {
            http.Error(w, "failed to create test case", http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusCreated)
        respondJSON(w, TestCase{ID: id, ProblemID: problemID, Input: req.Input, Expected: req.Expected})
    default:
        methodNotAllowed(w)
    }
}

// --- /admin/testcases/:id ---

func (s *Server) adminTestCasesMuxHandler(w http.ResponseWriter, r *http.Request) {
    id, err := parseID(r.URL.Path, "/admin/testcases/")
    if err != nil {
        http.NotFound(w, r)
        return
    }
    switch r.Method {
    case http.MethodPut:
        var req testCaseRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }
        if err := s.store.UpdateTestCase(r.Context(), id, req.Input, req.Expected); err != nil {
            http.Error(w, "failed to update test case", http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusNoContent)
    case http.MethodDelete:
        if err := s.store.DeleteTestCase(r.Context(), id); err != nil {
            http.Error(w, "failed to delete test case", http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusNoContent)
    default:
        methodNotAllowed(w)
    }
}

// --- /admin/users ---

func (s *Server) adminListUsersHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        methodNotAllowed(w)
        return
    }
    users, err := s.store.ListUsers(r.Context())
    if err != nil {
        http.Error(w, "failed to load users", http.StatusInternalServerError)
        return
    }
    if users == nil {
        users = []UserSummary{}
    }
    respondJSON(w, users)
}

// --- /admin/submissions ---

func (s *Server) adminListSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        methodNotAllowed(w)
        return
    }
    subs, err := s.store.ListAllSubmissions(r.Context())
    if err != nil {
        http.Error(w, "failed to load submissions", http.StatusInternalServerError)
        return
    }
    if subs == nil {
        subs = []SubmissionLogItem{}
    }
    respondJSON(w, subs)
}

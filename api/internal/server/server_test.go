package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/codearena/api/internal/auth"
)

// newTestServer returns a Server wired with an in-memory store and mock judge.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	t.Setenv("JUDGE_MOCK", "true")
	return &Server{store: NewInMemoryStore(), hub: NewHub()}
}

// authedRequest creates a request with claims injected for the given user.
func authedRequest(t *testing.T, method, path string, body *bytes.Reader) *http.Request {
	t.Helper()
	return authedRequestAs(t, method, path, body, 1, "test@example.com", "user")
}

// adminRequest creates a request with admin claims injected.
func adminRequest(t *testing.T, method, path string, body *bytes.Reader) *http.Request {
	t.Helper()
	return authedRequestAs(t, method, path, body, 1, "admin@example.com", "admin")
}

func authedRequestAs(t *testing.T, method, path string, body *bytes.Reader, userID int, email, role string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	claims := &auth.Claims{UserID: userID, Email: email, Role: role}
	ctx := context.WithValue(req.Context(), claimsKey, claims)
	return req.WithContext(ctx)
}

// --- /healthz ---

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	healthzHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", body["status"])
	}
}

// --- /problems ---

func TestGetProblems(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/problems", nil)
	w := httptest.NewRecorder()
	s.problemsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var problems []Problem
	if err := json.NewDecoder(w.Body).Decode(&problems); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(problems) == 0 {
		t.Error("expected at least one problem")
	}
}

func TestGetProblemsWrongMethod(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/problems", nil)
	w := httptest.NewRecorder()
	s.problemsHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestGetProblemByID(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/problems/1", nil)
	w := httptest.NewRecorder()
	s.problemDetailHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var p Problem
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if p.ID != 1 {
		t.Errorf("expected problem id=1, got %d", p.ID)
	}
}

func TestGetProblemByIDNotFound(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/problems/9999", nil)
	w := httptest.NewRecorder()
	s.problemDetailHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetProblemByIDBadPath(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/problems/abc", nil)
	w := httptest.NewRecorder()
	s.problemDetailHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- /submissions ---

func TestPostSubmission(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(SubmissionRequest{
		ProblemID: 1,
		Code:      `#include <bits/stdc++.h>` + "\n" + `int main(){cout<<"Hello World\n";}`,
		Language:  "cpp",
	})
	req := authedRequest(t, http.MethodPost, "/submissions", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.submissionsHandler(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
	var resp SubmissionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.SubmissionID <= 0 {
		t.Errorf("expected positive submission_id, got %d", resp.SubmissionID)
	}
	if resp.Status != "queued" {
		t.Errorf("expected status=queued, got %q", resp.Status)
	}
}

func TestPostSubmissionMissingFields(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"code": "foo", "language": "cpp"})
	req := authedRequest(t, http.MethodPost, "/submissions", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.submissionsHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPostSubmissionInvalidJSON(t *testing.T) {
	s := newTestServer(t)
	req := authedRequest(t, http.MethodPost, "/submissions", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	s.submissionsHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPostSubmissionsWrongMethod(t *testing.T) {
	s := newTestServer(t)
	req := authedRequest(t, http.MethodGet, "/submissions", nil)
	w := httptest.NewRecorder()
	s.submissionsHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- requireAuth middleware ---

func TestRequireAuthMissingHeader(t *testing.T) {
	s := newTestServer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/submissions", requireAuth(s.submissionsHandler))

	req := httptest.NewRequest(http.MethodPost, "/submissions", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuthInvalidToken(t *testing.T) {
	s := newTestServer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/submissions", requireAuth(s.submissionsHandler))

	req := httptest.NewRequest(http.MethodPost, "/submissions", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// --- /submissions/:id ---

func TestGetSubmissionStatus(t *testing.T) {
	s := newTestServer(t)

	postBody, _ := json.Marshal(SubmissionRequest{ProblemID: 1, Code: "x", Language: "cpp"})
	postReq := authedRequest(t, http.MethodPost, "/submissions", bytes.NewReader(postBody))
	postW := httptest.NewRecorder()
	s.submissionsHandler(postW, postReq)

	var postResp SubmissionResponse
	json.NewDecoder(postW.Body).Decode(&postResp)

	getReq := httptest.NewRequest(http.MethodGet, "/submissions/1", nil)
	getW := httptest.NewRecorder()
	s.submissionStatusHandler(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getW.Code)
	}
	var status SubmissionStatus
	if err := json.NewDecoder(getW.Body).Decode(&status); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if status.SubmissionID != postResp.SubmissionID {
		t.Errorf("expected id=%d, got %d", postResp.SubmissionID, status.SubmissionID)
	}
}

func TestGetSubmissionStatusNotFound(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/submissions/9999", nil)
	w := httptest.NewRecorder()
	s.submissionStatusHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetSubmissionStatusBadID(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/submissions/abc", nil)
	w := httptest.NewRecorder()
	s.submissionStatusHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- /auth/register and /auth/login ---

func TestRegister(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"email": "alice@example.com", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.registerHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp authResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.Email != "alice@example.com" {
		t.Errorf("expected email=alice@example.com, got %q", resp.Email)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"email": "alice@example.com", "password": "password123"})

	req1 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	httptest.NewRecorder()
	s.registerHandler(httptest.NewRecorder(), req1)

	body, _ = json.Marshal(map[string]string{"email": "alice@example.com", "password": "password123"})
	req2 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	s.registerHandler(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w2.Code)
	}
}

func TestRegisterShortPassword(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"email": "bob@example.com", "password": "short"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.registerHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLogin(t *testing.T) {
	s := newTestServer(t)

	// register first
	regBody, _ := json.Marshal(map[string]string{"email": "carol@example.com", "password": "password123"})
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	s.registerHandler(httptest.NewRecorder(), regReq)

	// now login
	loginBody, _ := json.Marshal(map[string]string{"email": "carol@example.com", "password": "password123"})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.loginHandler(w, loginReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp authResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	s := newTestServer(t)

	regBody, _ := json.Marshal(map[string]string{"email": "dave@example.com", "password": "correct-password"})
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	s.registerHandler(httptest.NewRecorder(), regReq)

	loginBody, _ := json.Marshal(map[string]string{"email": "dave@example.com", "password": "wrong-password"})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.loginHandler(w, loginReq)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLoginUnknownEmail(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"email": "nobody@example.com", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.loginHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// --- /users/me/submissions ---

func TestUserSubmissionsHandler(t *testing.T) {
	s := newTestServer(t)

	// insert two submissions as user 1
	s.store.InsertSubmission(context.Background(), SubmissionRequest{ProblemID: 1, Code: "x", Language: "cpp"}, 1)
	s.store.InsertSubmission(context.Background(), SubmissionRequest{ProblemID: 2, Code: "y", Language: "cpp"}, 1)

	req := authedRequest(t, http.MethodGet, "/users/me/submissions", nil)
	w := httptest.NewRecorder()
	s.userSubmissionsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var items []SubmissionHistoryItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

// --- CORS middleware ---

func TestCORSPreflight(t *testing.T) {
	s := newTestServer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/problems", s.problemsHandler)
	handler := corsMiddleware(mux)

	req := httptest.NewRequest(http.MethodOptions, "/problems", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected ACAO=*, got %q", got)
	}
}

func TestCORSHeadersOnNormalRequest(t *testing.T) {
	s := newTestServer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/problems", s.problemsHandler)
	handler := corsMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/problems", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected ACAO=*, got %q", got)
	}
}

// --- requireAdmin middleware ---

func TestRequireAdminForbiddenForUser(t *testing.T) {
	s := newTestServer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/problems", requireAdmin(s.adminProblemsMuxHandler))

	// Sign a real token with role=user; middleware reads Authorization header.
	token, _ := auth.Sign(1, "user@example.com", "user")
	req := httptest.NewRequest(http.MethodGet, "/admin/problems", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRequireAdminAllowsAdmin(t *testing.T) {
	s := newTestServer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/problems", requireAdmin(s.adminProblemsMuxHandler))

	token, _ := auth.Sign(1, "admin@example.com", "admin")
	req := httptest.NewRequest(http.MethodGet, "/admin/problems", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- /admin/problems CRUD ---

func TestAdminCreateProblem(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]any{
		"title": "Two Sum", "description": "Find two numbers", "difficulty": "Easy",
		"tags": []string{"array"}, "time_limit_ms": 1000, "memory_limit_mb": 128,
	})
	req := adminRequest(t, http.MethodPost, "/admin/problems", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.adminProblemsMuxHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var p Problem
	json.NewDecoder(w.Body).Decode(&p)
	if p.Title != "Two Sum" {
		t.Errorf("expected title=Two Sum, got %q", p.Title)
	}
	if p.ID <= 0 {
		t.Errorf("expected positive id, got %d", p.ID)
	}
}

func TestAdminCreateProblemMissingFields(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"title": "Only Title"})
	req := adminRequest(t, http.MethodPost, "/admin/problems", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.adminProblemsMuxHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAdminUpdateProblem(t *testing.T) {
	s := newTestServer(t)
	// update seeded problem 1
	body, _ := json.Marshal(map[string]any{
		"title": "Hello World v2", "description": "Updated", "difficulty": "Easy",
		"tags": []string{}, "time_limit_ms": 1000, "memory_limit_mb": 128,
	})
	req := adminRequest(t, http.MethodPut, "/admin/problems/1", bytes.NewReader(body))
	req.URL.Path = "/admin/problems/1"
	w := httptest.NewRecorder()
	s.adminProblemsDetailMuxHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var p Problem
	json.NewDecoder(w.Body).Decode(&p)
	if p.Title != "Hello World v2" {
		t.Errorf("expected updated title, got %q", p.Title)
	}
}

func TestAdminDeleteProblem(t *testing.T) {
	s := newTestServer(t)
	// create then delete
	body, _ := json.Marshal(map[string]any{
		"title": "Tmp", "description": "tmp", "difficulty": "Easy",
	})
	createReq := adminRequest(t, http.MethodPost, "/admin/problems", bytes.NewReader(body))
	createW := httptest.NewRecorder()
	s.adminProblemsMuxHandler(createW, createReq)
	var p Problem
	json.NewDecoder(createW.Body).Decode(&p)

	delReq := adminRequest(t, http.MethodDelete, "/admin/problems/"+itoa(p.ID), nil)
	delReq.URL.Path = "/admin/problems/" + itoa(p.ID)
	delW := httptest.NewRecorder()
	s.adminProblemsDetailMuxHandler(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", delW.Code)
	}
}

// --- /admin/problems/:id/testcases ---

func TestAdminTestCaseCRUD(t *testing.T) {
	s := newTestServer(t)

	// create
	body, _ := json.Marshal(testCaseRequest{Input: "1 2", Expected: "3\n"})
	createReq := adminRequest(t, http.MethodPost, "/admin/problems/1/testcases", bytes.NewReader(body))
	createReq.URL.Path = "/admin/problems/1/testcases"
	createW := httptest.NewRecorder()
	s.adminProblemsDetailMuxHandler(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createW.Code, createW.Body.String())
	}
	var tc TestCase
	json.NewDecoder(createW.Body).Decode(&tc)

	// list
	getReq := adminRequest(t, http.MethodGet, "/admin/problems/1/testcases", nil)
	getReq.URL.Path = "/admin/problems/1/testcases"
	getW := httptest.NewRecorder()
	s.adminProblemsDetailMuxHandler(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getW.Code)
	}
	var tcs []TestCase
	json.NewDecoder(getW.Body).Decode(&tcs)
	if len(tcs) != 1 {
		t.Errorf("expected 1 test case, got %d", len(tcs))
	}

	// update
	updBody, _ := json.Marshal(testCaseRequest{Input: "2 3", Expected: "5\n"})
	updReq := adminRequest(t, http.MethodPut, "/admin/testcases/"+itoa(tc.ID), bytes.NewReader(updBody))
	updReq.URL.Path = "/admin/testcases/" + itoa(tc.ID)
	updW := httptest.NewRecorder()
	s.adminTestCasesMuxHandler(updW, updReq)
	if updW.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", updW.Code)
	}

	// delete
	delReq := adminRequest(t, http.MethodDelete, "/admin/testcases/"+itoa(tc.ID), nil)
	delReq.URL.Path = "/admin/testcases/" + itoa(tc.ID)
	delW := httptest.NewRecorder()
	s.adminTestCasesMuxHandler(delW, delReq)
	if delW.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", delW.Code)
	}
}

// --- /admin/users and /admin/submissions ---

func TestAdminListUsers(t *testing.T) {
	s := newTestServer(t)
	s.store.CreateUser(context.Background(), "a@x.com", "hash")
	req := adminRequest(t, http.MethodGet, "/admin/users", nil)
	w := httptest.NewRecorder()
	s.adminListUsersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var users []UserSummary
	json.NewDecoder(w.Body).Decode(&users)
	if len(users) == 0 {
		t.Error("expected at least one user")
	}
}

func TestAdminListSubmissions(t *testing.T) {
	s := newTestServer(t)
	s.store.InsertSubmission(context.Background(), SubmissionRequest{ProblemID: 1, Code: "x", Language: "cpp"}, 0)
	req := adminRequest(t, http.MethodGet, "/admin/submissions", nil)
	w := httptest.NewRecorder()
	s.adminListSubmissionsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var subs []SubmissionLogItem
	json.NewDecoder(w.Body).Decode(&subs)
	if len(subs) == 0 {
		t.Error("expected at least one submission")
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

// --- Content-Type ---

func TestHandlersRespondWithJSON(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/problems", nil)
	w := httptest.NewRecorder()
	s.problemsHandler(w, req)

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected application/json Content-Type, got %q", ct)
	}
}

// --- CORS ---

func TestCORSAllowsAuthorizationHeader(t *testing.T) {
	s := newTestServer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/problems", s.problemsHandler)
	handler := corsMiddleware(mux)

	req := httptest.NewRequest(http.MethodOptions, "/problems", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	acah := w.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(acah, "Authorization") {
		t.Errorf("ACAH should include Authorization, got %q", acah)
	}
}

// --- Submission goroutine end-to-end via Hub ---

// TestSubmissionFlowNoTestCases exercises the full async path: POST /submissions
// fires the goroutine which discovers no test cases and notifies the hub.
func TestSubmissionFlowNoTestCases(t *testing.T) {
	s := newTestServer(t)

	// Subscribe before POST so we don't miss the buffered notification.
	ch := s.hub.Subscribe(1) // fresh store → first submission is ID 1
	defer s.hub.Unsubscribe(1, ch)

	body, _ := json.Marshal(SubmissionRequest{ProblemID: 1, Code: "x", Language: "cpp"})
	req := authedRequest(t, http.MethodPost, "/submissions", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.submissionsHandler(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}

	select {
	case verdict := <-ch:
		if verdict != "no_test_cases" {
			t.Errorf("want no_test_cases, got %q", verdict)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for verdict")
	}
}

// TestSubmissionFlowWithTestCases verifies the goroutine calls the mock judge
// and delivers its verdict to the hub when test cases exist.
func TestSubmissionFlowWithTestCases(t *testing.T) {
	s := newTestServer(t)
	s.store.CreateTestCase(context.Background(), 1, "", "Hello World\n")

	ch := s.hub.Subscribe(1)
	defer s.hub.Unsubscribe(1, ch)

	code := "#include <bits/stdc++.h>\nusing namespace std;\nint main(){cout<<\"Hello World\\n\";}"
	body, _ := json.Marshal(SubmissionRequest{ProblemID: 1, Code: code, Language: "cpp"})
	req := authedRequest(t, http.MethodPost, "/submissions", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.submissionsHandler(w, req)

	select {
	case verdict := <-ch:
		if verdict != "accepted" {
			t.Errorf("want accepted, got %q", verdict)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for verdict")
	}
}

// TestSubmissionUpdatesStoreAfterJudge verifies the verdict is written to the
// store (not just the hub) so GET /submissions/:id reflects the final verdict.
func TestSubmissionUpdatesStoreAfterJudge(t *testing.T) {
	s := newTestServer(t)
	s.store.CreateTestCase(context.Background(), 1, "", "Hello World\n")

	ch := s.hub.Subscribe(1)

	code := "#include <bits/stdc++.h>\nusing namespace std;\nint main(){cout<<\"Hello World\\n\";}"
	body, _ := json.Marshal(SubmissionRequest{ProblemID: 1, Code: code, Language: "cpp"})
	req := authedRequest(t, http.MethodPost, "/submissions", bytes.NewReader(body))
	s.submissionsHandler(httptest.NewRecorder(), req)

	// Wait for the goroutine to finish
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}

	// Now check the store reflects the verdict
	getReq := httptest.NewRequest(http.MethodGet, "/submissions/1", nil)
	getW := httptest.NewRecorder()
	s.submissionStatusHandler(getW, getReq)

	var st SubmissionStatus
	json.NewDecoder(getW.Body).Decode(&st)
	if st.Status != "complete" {
		t.Errorf("want status=complete, got %q", st.Status)
	}
	if st.Verdict != "accepted" {
		t.Errorf("want verdict=accepted, got %q", st.Verdict)
	}
}

// --- Auth handler edge cases ---

func TestRegisterMissingEmail(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.registerHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegisterMissingPassword(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"email": "e@x.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.registerHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLoginMissingFields(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.loginHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLoginResponseIncludesRole(t *testing.T) {
	s := newTestServer(t)

	// Register, then login, check role in response
	regBody, _ := json.Marshal(map[string]string{"email": "r@x.com", "password": "password123"})
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	s.registerHandler(httptest.NewRecorder(), regReq)

	loginBody, _ := json.Marshal(map[string]string{"email": "r@x.com", "password": "password123"})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.loginHandler(w, loginReq)

	var resp authResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Role == "" {
		t.Error("login response should include role")
	}
}

// --- Admin handler edge cases ---

func TestAdminGetProblemByID(t *testing.T) {
	s := newTestServer(t)
	req := adminRequest(t, http.MethodGet, "/admin/problems/1", nil)
	req.URL.Path = "/admin/problems/1"
	w := httptest.NewRecorder()
	s.adminProblemsDetailMuxHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var p Problem
	json.NewDecoder(w.Body).Decode(&p)
	if p.ID != 1 {
		t.Errorf("expected id=1, got %d", p.ID)
	}
}

func TestAdminGetProblemByIDNotFound(t *testing.T) {
	s := newTestServer(t)
	req := adminRequest(t, http.MethodGet, "/admin/problems/9999", nil)
	req.URL.Path = "/admin/problems/9999"
	w := httptest.NewRecorder()
	s.adminProblemsDetailMuxHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAdminCreateProblemDefaultsZeroLimits(t *testing.T) {
	s := newTestServer(t)
	// Omit time_limit_ms and memory_limit_mb — should default to 2000/256
	body, _ := json.Marshal(map[string]any{
		"title": "DefaultLimits", "description": "d", "difficulty": "Easy",
	})
	req := adminRequest(t, http.MethodPost, "/admin/problems", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.adminProblemsMuxHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var p Problem
	json.NewDecoder(w.Body).Decode(&p)
	if p.TimeLimitMs != 2000 {
		t.Errorf("want TimeLimitMs=2000, got %d", p.TimeLimitMs)
	}
	if p.MemoryLimitMb != 256 {
		t.Errorf("want MemoryLimitMb=256, got %d", p.MemoryLimitMb)
	}
}

func TestAdminTestCaseRequiresExpected(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(testCaseRequest{Input: "1 2"}) // no Expected
	req := adminRequest(t, http.MethodPost, "/admin/problems/1/testcases", bytes.NewReader(body))
	req.URL.Path = "/admin/problems/1/testcases"
	w := httptest.NewRecorder()
	s.adminProblemsDetailMuxHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAdminProblemsMuxWrongMethod(t *testing.T) {
	s := newTestServer(t)
	req := adminRequest(t, http.MethodDelete, "/admin/problems", nil)
	w := httptest.NewRecorder()
	s.adminProblemsMuxHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestAdminTestCaseMuxWrongMethod(t *testing.T) {
	s := newTestServer(t)
	req := adminRequest(t, http.MethodGet, "/admin/testcases/1", nil)
	req.URL.Path = "/admin/testcases/1"
	w := httptest.NewRecorder()
	s.adminTestCasesMuxHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestAdminListUsersWrongMethod(t *testing.T) {
	s := newTestServer(t)
	req := adminRequest(t, http.MethodPost, "/admin/users", nil)
	w := httptest.NewRecorder()
	s.adminListUsersHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestAdminListSubmissionsWrongMethod(t *testing.T) {
	s := newTestServer(t)
	req := adminRequest(t, http.MethodPost, "/admin/submissions", nil)
	w := httptest.NewRecorder()
	s.adminListSubmissionsHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestUserSubmissionsHandlerWrongMethod(t *testing.T) {
	s := newTestServer(t)
	req := authedRequest(t, http.MethodPost, "/users/me/submissions", nil)
	w := httptest.NewRecorder()
	s.userSubmissionsHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestAdminListUsersEmptyStore(t *testing.T) {
	s := newTestServer(t)
	req := adminRequest(t, http.MethodGet, "/admin/users", nil)
	w := httptest.NewRecorder()
	s.adminListUsersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Response must be a valid JSON array (not null)
	var users []UserSummary
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if users == nil {
		t.Error("response body should be [] not null")
	}
}

func TestAdminListSubmissionsEmptyStore(t *testing.T) {
	s := newTestServer(t)
	req := adminRequest(t, http.MethodGet, "/admin/submissions", nil)
	w := httptest.NewRecorder()
	s.adminListSubmissionsHandler(w, req)

	var subs []SubmissionLogItem
	json.NewDecoder(w.Body).Decode(&subs)
	if subs == nil {
		t.Error("response body should be [] not null")
	}
}

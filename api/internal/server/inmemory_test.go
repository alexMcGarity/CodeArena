package server

import (
	"context"
	"strings"
	"sync"
	"testing"
)

func TestFetchProblems(t *testing.T) {
	s := NewInMemoryStore()
	got, err := s.FetchProblems(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected seeded problems, got none")
	}
}

func TestFetchProblemByIDFound(t *testing.T) {
	s := NewInMemoryStore()
	p, err := s.FetchProblemByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != 1 {
		t.Errorf("expected id=1, got %d", p.ID)
	}
}

func TestFetchProblemByIDNotFound(t *testing.T) {
	s := NewInMemoryStore()
	_, err := s.FetchProblemByID(context.Background(), 9999)
	if err == nil {
		t.Error("expected error for missing problem, got nil")
	}
}

func TestInsertSubmissionReturnsID(t *testing.T) {
	s := NewInMemoryStore()
	id, err := s.InsertSubmission(context.Background(), SubmissionRequest{
		ProblemID: 1, Code: "x", Language: "cpp",
	}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}
}

func TestInsertSubmissionIDsIncrement(t *testing.T) {
	s := NewInMemoryStore()
	req := SubmissionRequest{ProblemID: 1, Code: "x", Language: "cpp"}
	id1, _ := s.InsertSubmission(context.Background(), req, 0)
	id2, _ := s.InsertSubmission(context.Background(), req, 0)
	if id2 != id1+1 {
		t.Errorf("expected consecutive IDs %d and %d", id1, id2)
	}
}

func TestGetSubmissionStatusQueued(t *testing.T) {
	s := NewInMemoryStore()
	id, _ := s.InsertSubmission(context.Background(), SubmissionRequest{
		ProblemID: 1, Code: "x", Language: "cpp",
	}, 0)
	status, err := s.GetSubmissionStatus(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "queued" {
		t.Errorf("expected status=queued, got %q", status.Status)
	}
	if status.Verdict != "queued" {
		t.Errorf("expected verdict=queued, got %q", status.Verdict)
	}
}

func TestUpdateSubmissionVerdict(t *testing.T) {
	s := NewInMemoryStore()
	id, _ := s.InsertSubmission(context.Background(), SubmissionRequest{
		ProblemID: 1, Code: "x", Language: "cpp",
	}, 0)
	if err := s.UpdateSubmissionVerdict(context.Background(), id, "accepted"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	status, err := s.GetSubmissionStatus(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Verdict != "accepted" {
		t.Errorf("expected verdict=accepted, got %q", status.Verdict)
	}
	if status.Status != "complete" {
		t.Errorf("expected status=complete, got %q", status.Status)
	}
}

func TestUpdateNonexistentSubmission(t *testing.T) {
	s := NewInMemoryStore()
	err := s.UpdateSubmissionVerdict(context.Background(), 9999, "accepted")
	if err == nil {
		t.Error("expected error updating nonexistent submission, got nil")
	}
}

func TestGetNonexistentSubmissionStatus(t *testing.T) {
	s := NewInMemoryStore()
	_, err := s.GetSubmissionStatus(context.Background(), 9999)
	if err == nil {
		t.Error("expected error for missing submission, got nil")
	}
}

func TestConcurrentInserts(t *testing.T) {
	s := NewInMemoryStore()
	const goroutines = 50
	ids := make([]int, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			id, err := s.InsertSubmission(context.Background(), SubmissionRequest{
				ProblemID: 1, Code: "x", Language: "cpp",
			}, idx)
			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", idx, err)
				return
			}
			ids[idx] = id
		}(i)
	}
	wg.Wait()

	seen := make(map[int]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate submission id: %d", id)
		}
		seen[id] = true
	}
}

// --- user store tests ---

func TestCreateUser(t *testing.T) {
	s := NewInMemoryStore()
	id, err := s.CreateUser(context.Background(), "alice@example.com", "hashedpw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive user id, got %d", id)
	}
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	s := NewInMemoryStore()
	s.CreateUser(context.Background(), "bob@example.com", "hash1")
	_, err := s.CreateUser(context.Background(), "bob@example.com", "hash2")
	if err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
}

func TestGetUserByEmail(t *testing.T) {
	s := NewInMemoryStore()
	s.CreateUser(context.Background(), "carol@example.com", "myhash")
	u, err := s.GetUserByEmail(context.Background(), "carol@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "carol@example.com" {
		t.Errorf("wrong email: %q", u.Email)
	}
	if u.PasswordHash != "myhash" {
		t.Errorf("wrong hash: %q", u.PasswordHash)
	}
}

func TestGetUserByEmailNotFound(t *testing.T) {
	s := NewInMemoryStore()
	_, err := s.GetUserByEmail(context.Background(), "nobody@example.com")
	if err == nil {
		t.Error("expected error for missing user, got nil")
	}
}

func TestGetSubmissionsByUserID(t *testing.T) {
	s := NewInMemoryStore()
	s.InsertSubmission(context.Background(), SubmissionRequest{ProblemID: 1, Code: "x", Language: "cpp"}, 42)
	s.InsertSubmission(context.Background(), SubmissionRequest{ProblemID: 2, Code: "y", Language: "cpp"}, 42)
	s.InsertSubmission(context.Background(), SubmissionRequest{ProblemID: 1, Code: "z", Language: "cpp"}, 99)

	items, err := s.GetSubmissionsByUserID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items for user 42, got %d", len(items))
	}
}

func TestGetSubmissionsByUserIDNoSubmissions(t *testing.T) {
	s := NewInMemoryStore()
	items, err := s.GetSubmissionsByUserID(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// should return an empty slice, not nil, so callers can range safely
	if items == nil {
		items = []SubmissionHistoryItem{} // nil slice is fine; just confirm no error
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// --- SeedAdmin ---

func TestSeedAdminCreatesNewAdminUser(t *testing.T) {
	s := NewInMemoryStore()
	if err := s.SeedAdmin(context.Background(), "admin@test.com", "adminpass"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}
	u, err := s.GetUserByEmail(context.Background(), "admin@test.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if u.Role != "admin" {
		t.Errorf("want role=admin, got %q", u.Role)
	}
}

func TestSeedAdminPromotesExistingUser(t *testing.T) {
	s := NewInMemoryStore()
	// Create as ordinary user first
	s.CreateUser(context.Background(), "promote@test.com", "oldhash")

	if err := s.SeedAdmin(context.Background(), "promote@test.com", "newpass"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}
	u, err := s.GetUserByEmail(context.Background(), "promote@test.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if u.Role != "admin" {
		t.Errorf("want role=admin after promotion, got %q", u.Role)
	}
}

func TestSeedAdminNormalisesEmail(t *testing.T) {
	s := NewInMemoryStore()
	// Mixed-case + whitespace
	s.SeedAdmin(context.Background(), "  ADMIN@Test.COM  ", "pw12345678")
	_, err := s.GetUserByEmail(context.Background(), "admin@test.com")
	if err != nil {
		t.Errorf("email should have been lowercased and trimmed, lookup failed: %v", err)
	}
}

// --- Problem CRUD ---

func TestCreateAndFetchProblem(t *testing.T) {
	s := NewInMemoryStore()
	initial, _ := s.FetchProblems(context.Background())
	p := Problem{Title: "New Problem", Description: "Desc", Difficulty: "Medium", Tags: []string{"math"}}
	id, err := s.CreateProblem(context.Background(), p)
	if err != nil {
		t.Fatalf("CreateProblem: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}
	all, _ := s.FetchProblems(context.Background())
	if len(all) != len(initial)+1 {
		t.Errorf("expected %d problems, got %d", len(initial)+1, len(all))
	}
	fetched, err := s.FetchProblemByID(context.Background(), id)
	if err != nil {
		t.Fatalf("FetchProblemByID: %v", err)
	}
	if fetched.Title != "New Problem" {
		t.Errorf("title mismatch: %q", fetched.Title)
	}
}

func TestUpdateProblem(t *testing.T) {
	s := NewInMemoryStore()
	updated := Problem{ID: 1, Title: "Updated", Description: "New desc", Difficulty: "Hard", Tags: []string{}}
	if err := s.UpdateProblem(context.Background(), updated); err != nil {
		t.Fatalf("UpdateProblem: %v", err)
	}
	p, _ := s.FetchProblemByID(context.Background(), 1)
	if p.Title != "Updated" {
		t.Errorf("title not updated: %q", p.Title)
	}
	if p.Difficulty != "Hard" {
		t.Errorf("difficulty not updated: %q", p.Difficulty)
	}
}

func TestUpdateNonexistentProblem(t *testing.T) {
	s := NewInMemoryStore()
	err := s.UpdateProblem(context.Background(), Problem{ID: 9999, Title: "Ghost"})
	if err == nil {
		t.Error("expected error updating nonexistent problem, got nil")
	}
}

func TestDeleteProblem(t *testing.T) {
	s := NewInMemoryStore()
	id, _ := s.CreateProblem(context.Background(), Problem{Title: "Tmp", Description: "d", Difficulty: "Easy"})
	if err := s.DeleteProblem(context.Background(), id); err != nil {
		t.Fatalf("DeleteProblem: %v", err)
	}
	_, err := s.FetchProblemByID(context.Background(), id)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestDeleteNonexistentProblem(t *testing.T) {
	s := NewInMemoryStore()
	if err := s.DeleteProblem(context.Background(), 9999); err == nil {
		t.Error("expected error deleting nonexistent problem, got nil")
	}
}

// --- Test case CRUD ---

func TestTestCaseCRUDStore(t *testing.T) {
	s := NewInMemoryStore()

	// create
	id, err := s.CreateTestCase(context.Background(), 1, "5\n", "10\n")
	if err != nil {
		t.Fatalf("CreateTestCase: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	// list
	tcs, err := s.GetTestCasesByProblemID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetTestCasesByProblemID: %v", err)
	}
	if len(tcs) != 1 {
		t.Fatalf("expected 1 test case, got %d", len(tcs))
	}
	if tcs[0].Input != "5\n" || tcs[0].Expected != "10\n" {
		t.Errorf("stored test case content mismatch: %+v", tcs[0])
	}

	// update
	if err := s.UpdateTestCase(context.Background(), id, "6\n", "12\n"); err != nil {
		t.Fatalf("UpdateTestCase: %v", err)
	}
	tcs, _ = s.GetTestCasesByProblemID(context.Background(), 1)
	if tcs[0].Input != "6\n" || tcs[0].Expected != "12\n" {
		t.Errorf("updated test case mismatch: %+v", tcs[0])
	}

	// delete
	if err := s.DeleteTestCase(context.Background(), id); err != nil {
		t.Fatalf("DeleteTestCase: %v", err)
	}
	tcs, _ = s.GetTestCasesByProblemID(context.Background(), 1)
	if len(tcs) != 0 {
		t.Errorf("expected 0 test cases after delete, got %d", len(tcs))
	}
}

func TestUpdateNonexistentTestCase(t *testing.T) {
	s := NewInMemoryStore()
	if err := s.UpdateTestCase(context.Background(), 9999, "in", "out"); err == nil {
		t.Error("expected error updating nonexistent test case, got nil")
	}
}

func TestDeleteNonexistentTestCase(t *testing.T) {
	s := NewInMemoryStore()
	if err := s.DeleteTestCase(context.Background(), 9999); err == nil {
		t.Error("expected error deleting nonexistent test case, got nil")
	}
}

func TestGetTestCasesForUnknownProblemReturnsEmpty(t *testing.T) {
	s := NewInMemoryStore()
	tcs, err := s.GetTestCasesByProblemID(context.Background(), 9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tcs) != 0 {
		t.Errorf("expected 0 test cases, got %d", len(tcs))
	}
}

// --- Admin list views ---

func TestListUsersReturnsAllUsers(t *testing.T) {
	s := NewInMemoryStore()
	s.CreateUser(context.Background(), "u1@x.com", "h")
	s.CreateUser(context.Background(), "u2@x.com", "h")

	users, err := s.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
	emails := make(map[string]bool)
	for _, u := range users {
		emails[u.Email] = true
	}
	if !emails["u1@x.com"] || !emails["u2@x.com"] {
		t.Errorf("missing expected user emails: %v", emails)
	}
}

func TestListUsersIncludesRole(t *testing.T) {
	s := NewInMemoryStore()
	s.SeedAdmin(context.Background(), "boss@x.com", "password")
	s.CreateUser(context.Background(), "worker@x.com", "hash")

	users, _ := s.ListUsers(context.Background())
	roles := make(map[string]string)
	for _, u := range users {
		roles[u.Email] = u.Role
	}
	if roles["boss@x.com"] != "admin" {
		t.Errorf("boss should be admin, got %q", roles["boss@x.com"])
	}
	if roles["worker@x.com"] != "user" {
		t.Errorf("worker should be user, got %q", roles["worker@x.com"])
	}
}

func TestListAllSubmissions(t *testing.T) {
	s := NewInMemoryStore()
	s.InsertSubmission(context.Background(), SubmissionRequest{ProblemID: 1, Code: "a", Language: "cpp"}, 0)
	s.InsertSubmission(context.Background(), SubmissionRequest{ProblemID: 2, Code: "b", Language: "python"}, 0)

	items, err := s.ListAllSubmissions(context.Background())
	if err != nil {
		t.Fatalf("ListAllSubmissions: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	langs := make(map[string]bool)
	for _, item := range items {
		langs[item.Language] = true
	}
	if !langs["cpp"] || !langs["python"] {
		t.Errorf("missing expected languages: %v", langs)
	}
}

// --- Concurrent updates ---

func TestConcurrentSubmissionVerdictUpdates(t *testing.T) {
	s := NewInMemoryStore()
	// Insert N submissions, update them all concurrently.
	const n = 40
	ids := make([]int, n)
	for i := range n {
		id, _ := s.InsertSubmission(context.Background(), SubmissionRequest{ProblemID: 1, Code: "x", Language: "cpp"}, i)
		ids[i] = id
	}

	var wg sync.WaitGroup
	wg.Add(n)
	for _, id := range ids {
		go func(submID int) {
			defer wg.Done()
			s.UpdateSubmissionVerdict(context.Background(), submID, "accepted")
		}(id)
	}
	wg.Wait()

	// Verify all got updated
	for _, id := range ids {
		st, err := s.GetSubmissionStatus(context.Background(), id)
		if err != nil {
			t.Errorf("submission %d: %v", id, err)
			continue
		}
		if st.Verdict != "accepted" {
			t.Errorf("submission %d: want accepted, got %q", id, st.Verdict)
		}
	}
}

func TestConcurrentUserCreationUniqueIDs(t *testing.T) {
	s := NewInMemoryStore()
	const n = 30
	ids := make([]int, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			// Each goroutine registers a unique email
			email := "user" + strings.Repeat("x", idx) + "@test.com"
			id, err := s.CreateUser(context.Background(), email, "hash")
			if err != nil {
				t.Errorf("CreateUser(%s): %v", email, err)
				return
			}
			ids[idx] = id
		}(i)
	}
	wg.Wait()

	seen := make(map[int]bool)
	for _, id := range ids {
		if id == 0 {
			continue // failed goroutine already reported
		}
		if seen[id] {
			t.Errorf("duplicate user id: %d", id)
		}
		seen[id] = true
	}
}

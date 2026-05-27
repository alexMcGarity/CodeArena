package server

import (
    "context"
    "fmt"
    "strings"
    "sync"
    "time"

    "golang.org/x/crypto/bcrypt"
)

type Submission struct {
    ID        int
    ProblemID int
    Code      string
    Language  string
    Verdict   string
    UserID    int
    CreatedAt time.Time
}

type InMemoryStore struct {
    mu           sync.RWMutex
    problems     []Problem
    nextProbID   int
    submissions  map[int]Submission
    nextSubID    int
    users        map[int]User
    nextUserID   int
    emailIndex   map[string]int // email → user id
    testCases    map[int]TestCase
    nextTCID     int
}

func NewInMemoryStore() *InMemoryStore {
    s := &InMemoryStore{
        problems:    make([]Problem, len(problems)),
        submissions: make(map[int]Submission),
        nextSubID:   1,
        users:       make(map[int]User),
        nextUserID:  1,
        emailIndex:  make(map[string]int),
        testCases:   make(map[int]TestCase),
        nextTCID:    1,
    }
    copy(s.problems, problems)
    // next problem ID starts after the seeded ones
    for _, p := range s.problems {
        if p.ID >= s.nextProbID {
            s.nextProbID = p.ID + 1
        }
    }
    return s
}

// --- problems ---

func (s *InMemoryStore) FetchProblems(ctx context.Context) ([]Problem, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    out := make([]Problem, len(s.problems))
    copy(out, s.problems)
    return out, nil
}

func (s *InMemoryStore) FetchProblemByID(ctx context.Context, id int) (*Problem, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    for _, p := range s.problems {
        if p.ID == id {
            cp := p
            return &cp, nil
        }
    }
    return nil, fmt.Errorf("problem not found")
}

func (s *InMemoryStore) CreateProblem(ctx context.Context, p Problem) (int, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    p.ID = s.nextProbID
    s.nextProbID++
    s.problems = append(s.problems, p)
    return p.ID, nil
}

func (s *InMemoryStore) UpdateProblem(ctx context.Context, p Problem) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    for i, existing := range s.problems {
        if existing.ID == p.ID {
            s.problems[i] = p
            return nil
        }
    }
    return fmt.Errorf("problem not found")
}

func (s *InMemoryStore) DeleteProblem(ctx context.Context, id int) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    for i, p := range s.problems {
        if p.ID == id {
            s.problems = append(s.problems[:i], s.problems[i+1:]...)
            return nil
        }
    }
    return fmt.Errorf("problem not found")
}

// --- submissions ---

func (s *InMemoryStore) InsertSubmission(ctx context.Context, req SubmissionRequest, userID int) (int, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    id := s.nextSubID
    s.nextSubID++
    s.submissions[id] = Submission{
        ID:        id,
        ProblemID: req.ProblemID,
        Code:      req.Code,
        Language:  req.Language,
        Verdict:   "queued",
        UserID:    userID,
        CreatedAt: time.Now(),
    }
    return id, nil
}

func (s *InMemoryStore) UpdateSubmissionVerdict(ctx context.Context, id int, verdict string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    sub, ok := s.submissions[id]
    if !ok {
        return fmt.Errorf("submission not found")
    }
    sub.Verdict = verdict
    s.submissions[id] = sub
    return nil
}

func (s *InMemoryStore) GetSubmissionStatus(ctx context.Context, id int) (SubmissionStatus, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    sub, ok := s.submissions[id]
    if !ok {
        return SubmissionStatus{}, fmt.Errorf("submission not found")
    }
    status := "complete"
    if sub.Verdict == "queued" {
        status = "queued"
    }
    return SubmissionStatus{SubmissionID: id, Status: status, Verdict: sub.Verdict}, nil
}

// --- users ---

func (s *InMemoryStore) CreateUser(ctx context.Context, email, passwordHash string) (int, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, exists := s.emailIndex[email]; exists {
        return 0, fmt.Errorf("email already registered")
    }
    id := s.nextUserID
    s.nextUserID++
    s.users[id] = User{ID: id, Email: email, PasswordHash: passwordHash, Role: "user"}
    s.emailIndex[email] = id
    return id, nil
}

func (s *InMemoryStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    id, ok := s.emailIndex[email]
    if !ok {
        return nil, fmt.Errorf("user not found")
    }
    u := s.users[id]
    return &u, nil
}

func (s *InMemoryStore) GetSubmissionsByUserID(ctx context.Context, userID int) ([]SubmissionHistoryItem, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    titles := make(map[int]string)
    for _, p := range s.problems {
        titles[p.ID] = p.Title
    }
    var items []SubmissionHistoryItem
    for _, sub := range s.submissions {
        if sub.UserID != userID {
            continue
        }
        items = append(items, SubmissionHistoryItem{
            ID:           sub.ID,
            ProblemID:    sub.ProblemID,
            ProblemTitle: titles[sub.ProblemID],
            Language:     sub.Language,
            Verdict:      sub.Verdict,
            CreatedAt:    sub.CreatedAt.Format(time.RFC3339),
        })
    }
    return items, nil
}

func (s *InMemoryStore) SeedAdmin(ctx context.Context, email, password string) error {
    email = strings.ToLower(strings.TrimSpace(email))
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    s.mu.Lock()
    defer s.mu.Unlock()
    if id, exists := s.emailIndex[email]; exists {
        u := s.users[id]
        u.Role = "admin"
        s.users[id] = u
        return nil
    }
    id := s.nextUserID
    s.nextUserID++
    s.users[id] = User{ID: id, Email: email, PasswordHash: string(hash), Role: "admin"}
    s.emailIndex[email] = id
    return nil
}

// --- test cases ---

func (s *InMemoryStore) GetTestCasesByProblemID(ctx context.Context, problemID int) ([]TestCase, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    var out []TestCase
    for _, tc := range s.testCases {
        if tc.ProblemID == problemID {
            out = append(out, tc)
        }
    }
    return out, nil
}

func (s *InMemoryStore) CreateTestCase(ctx context.Context, problemID int, input, expected string) (int, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    id := s.nextTCID
    s.nextTCID++
    s.testCases[id] = TestCase{ID: id, ProblemID: problemID, Input: input, Expected: expected}
    return id, nil
}

func (s *InMemoryStore) UpdateTestCase(ctx context.Context, id int, input, expected string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    tc, ok := s.testCases[id]
    if !ok {
        return fmt.Errorf("test case not found")
    }
    tc.Input = input
    tc.Expected = expected
    s.testCases[id] = tc
    return nil
}

func (s *InMemoryStore) DeleteTestCase(ctx context.Context, id int) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, ok := s.testCases[id]; !ok {
        return fmt.Errorf("test case not found")
    }
    delete(s.testCases, id)
    return nil
}

// --- admin views ---

func (s *InMemoryStore) ListUsers(ctx context.Context) ([]UserSummary, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    out := make([]UserSummary, 0, len(s.users))
    for _, u := range s.users {
        out = append(out, UserSummary{ID: u.ID, Email: u.Email, Role: u.Role})
    }
    return out, nil
}

func (s *InMemoryStore) ListAllSubmissions(ctx context.Context) ([]SubmissionLogItem, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    titles := make(map[int]string)
    for _, p := range s.problems {
        titles[p.ID] = p.Title
    }
    emails := make(map[int]string)
    for _, u := range s.users {
        emails[u.ID] = u.Email
    }
    out := make([]SubmissionLogItem, 0, len(s.submissions))
    for _, sub := range s.submissions {
        out = append(out, SubmissionLogItem{
            ID:           sub.ID,
            ProblemTitle: titles[sub.ProblemID],
            UserEmail:    emails[sub.UserID],
            Language:     sub.Language,
            Verdict:      sub.Verdict,
            CreatedAt:    sub.CreatedAt.Format(time.RFC3339),
        })
    }
    return out, nil
}

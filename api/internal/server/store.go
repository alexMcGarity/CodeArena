package server

import (
    "context"
    "fmt"
    "os"
    "strings"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "golang.org/x/crypto/bcrypt"
)

type PostgresStore struct {
    db *pgxpool.Pool
}

func connectDB(ctx context.Context) (*pgxpool.Pool, error) {
    databaseURL := os.Getenv("DATABASE_URL")
    if databaseURL == "" {
        databaseURL = "postgres://postgres:postgres@localhost:5432/codearena?sslmode=disable"
    }
    config, err := pgxpool.ParseConfig(databaseURL)
    if err != nil {
        return nil, fmt.Errorf("parse database url: %w", err)
    }
    config.MaxConns = 5
    config.MaxConnIdleTime = 5 * time.Minute
    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return nil, fmt.Errorf("connect to database: %w", err)
    }
    return pool, nil
}

func seedProblems(ctx context.Context, db *pgxpool.Pool) error {
    for _, p := range problems {
        _, err := db.Exec(ctx,
            `INSERT INTO problems (id, title, description, difficulty, tags, time_limit_ms, memory_limit_mb)
             VALUES ($1, $2, $3, $4, $5, $6, $7)
             ON CONFLICT (id) DO NOTHING`,
            p.ID, p.Title, p.Description, p.Difficulty, p.Tags, p.TimeLimitMs, p.MemoryLimitMb,
        )
        if err != nil {
            return fmt.Errorf("seed problems: %w", err)
        }
    }
    return nil
}

// --- problems ---

func (s *PostgresStore) FetchProblems(ctx context.Context) ([]Problem, error) {
    rows, err := s.db.Query(ctx,
        `SELECT id, title, difficulty, description, tags, time_limit_ms, memory_limit_mb
         FROM problems ORDER BY id`)
    if err != nil {
        return nil, fmt.Errorf("fetch problems: %w", err)
    }
    defer rows.Close()
    var list []Problem
    for rows.Next() {
        var p Problem
        if err := rows.Scan(&p.ID, &p.Title, &p.Difficulty, &p.Description, &p.Tags,
            &p.TimeLimitMs, &p.MemoryLimitMb); err != nil {
            return nil, fmt.Errorf("scan problem: %w", err)
        }
        list = append(list, p)
    }
    return list, nil
}

func (s *PostgresStore) FetchProblemByID(ctx context.Context, id int) (*Problem, error) {
    var p Problem
    err := s.db.QueryRow(ctx,
        `SELECT id, title, difficulty, description, tags, time_limit_ms, memory_limit_mb
         FROM problems WHERE id=$1`, id).
        Scan(&p.ID, &p.Title, &p.Difficulty, &p.Description, &p.Tags, &p.TimeLimitMs, &p.MemoryLimitMb)
    if err != nil {
        return nil, fmt.Errorf("problem not found")
    }
    return &p, nil
}

func (s *PostgresStore) CreateProblem(ctx context.Context, p Problem) (int, error) {
    var id int
    err := s.db.QueryRow(ctx,
        `INSERT INTO problems (title, description, difficulty, tags, time_limit_ms, memory_limit_mb)
         VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
        p.Title, p.Description, p.Difficulty, p.Tags, p.TimeLimitMs, p.MemoryLimitMb,
    ).Scan(&id)
    if err != nil {
        return 0, fmt.Errorf("create problem: %w", err)
    }
    return id, nil
}

func (s *PostgresStore) UpdateProblem(ctx context.Context, p Problem) error {
    _, err := s.db.Exec(ctx,
        `UPDATE problems SET title=$1, description=$2, difficulty=$3, tags=$4,
         time_limit_ms=$5, memory_limit_mb=$6 WHERE id=$7`,
        p.Title, p.Description, p.Difficulty, p.Tags, p.TimeLimitMs, p.MemoryLimitMb, p.ID,
    )
    if err != nil {
        return fmt.Errorf("update problem: %w", err)
    }
    return nil
}

func (s *PostgresStore) DeleteProblem(ctx context.Context, id int) error {
    _, err := s.db.Exec(ctx, `DELETE FROM problems WHERE id=$1`, id)
    if err != nil {
        return fmt.Errorf("delete problem: %w", err)
    }
    return nil
}

// --- submissions ---

func (s *PostgresStore) InsertSubmission(ctx context.Context, req SubmissionRequest, userID int) (int, error) {
    var id int
    err := s.db.QueryRow(ctx,
        `INSERT INTO submissions (problem_id, language, code, verdict, user_id)
         VALUES ($1, $2, $3, 'queued', $4) RETURNING id`,
        req.ProblemID, req.Language, req.Code, userID,
    ).Scan(&id)
    if err != nil {
        return 0, fmt.Errorf("insert submission: %w", err)
    }
    return id, nil
}

func (s *PostgresStore) UpdateSubmissionVerdict(ctx context.Context, id int, verdict string) error {
    _, err := s.db.Exec(ctx, `UPDATE submissions SET verdict=$1 WHERE id=$2`, verdict, id)
    return err
}

func (s *PostgresStore) GetSubmissionStatus(ctx context.Context, id int) (SubmissionStatus, error) {
    var verdict string
    err := s.db.QueryRow(ctx, `SELECT verdict FROM submissions WHERE id=$1`, id).Scan(&verdict)
    if err != nil {
        return SubmissionStatus{}, err
    }
    status := "complete"
    if verdict == "queued" {
        status = "queued"
    }
    return SubmissionStatus{SubmissionID: id, Status: status, Verdict: verdict}, nil
}

// --- users ---

func (s *PostgresStore) CreateUser(ctx context.Context, email, passwordHash string) (int, error) {
    var id int
    err := s.db.QueryRow(ctx,
        `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
        email, passwordHash,
    ).Scan(&id)
    if err != nil {
        return 0, fmt.Errorf("create user: %w", err)
    }
    return id, nil
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
    var u User
    err := s.db.QueryRow(ctx,
        `SELECT id, email, password_hash, role FROM users WHERE email=$1`, email,
    ).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role)
    if err != nil {
        return nil, fmt.Errorf("user not found")
    }
    return &u, nil
}

func (s *PostgresStore) GetSubmissionsByUserID(ctx context.Context, userID int) ([]SubmissionHistoryItem, error) {
    rows, err := s.db.Query(ctx,
        `SELECT s.id, s.problem_id, p.title, s.language, s.verdict, s.created_at
         FROM submissions s JOIN problems p ON s.problem_id = p.id
         WHERE s.user_id = $1 ORDER BY s.created_at DESC`, userID)
    if err != nil {
        return nil, fmt.Errorf("fetch user submissions: %w", err)
    }
    defer rows.Close()
    var items []SubmissionHistoryItem
    for rows.Next() {
        var item SubmissionHistoryItem
        var createdAt time.Time
        if err := rows.Scan(&item.ID, &item.ProblemID, &item.ProblemTitle,
            &item.Language, &item.Verdict, &createdAt); err != nil {
            return nil, fmt.Errorf("scan submission: %w", err)
        }
        item.CreatedAt = createdAt.Format(time.RFC3339)
        items = append(items, item)
    }
    return items, nil
}

func (s *PostgresStore) SeedAdmin(ctx context.Context, email, password string) error {
    email = strings.ToLower(strings.TrimSpace(email))
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    _, err = s.db.Exec(ctx,
        `INSERT INTO users (email, password_hash, role) VALUES ($1, $2, 'admin')
         ON CONFLICT (email) DO UPDATE SET role = 'admin'`,
        email, string(hash),
    )
    return err
}

// --- test cases ---

func (s *PostgresStore) GetTestCasesByProblemID(ctx context.Context, problemID int) ([]TestCase, error) {
    rows, err := s.db.Query(ctx,
        `SELECT id, problem_id, input, expected FROM test_cases WHERE problem_id=$1 ORDER BY id`,
        problemID)
    if err != nil {
        return nil, fmt.Errorf("fetch test cases: %w", err)
    }
    defer rows.Close()
    var out []TestCase
    for rows.Next() {
        var tc TestCase
        if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.Input, &tc.Expected); err != nil {
            return nil, fmt.Errorf("scan test case: %w", err)
        }
        out = append(out, tc)
    }
    return out, nil
}

func (s *PostgresStore) CreateTestCase(ctx context.Context, problemID int, input, expected string) (int, error) {
    var id int
    err := s.db.QueryRow(ctx,
        `INSERT INTO test_cases (problem_id, input, expected) VALUES ($1, $2, $3) RETURNING id`,
        problemID, input, expected,
    ).Scan(&id)
    if err != nil {
        return 0, fmt.Errorf("create test case: %w", err)
    }
    return id, nil
}

func (s *PostgresStore) UpdateTestCase(ctx context.Context, id int, input, expected string) error {
    _, err := s.db.Exec(ctx,
        `UPDATE test_cases SET input=$1, expected=$2 WHERE id=$3`, input, expected, id)
    return err
}

func (s *PostgresStore) DeleteTestCase(ctx context.Context, id int) error {
    _, err := s.db.Exec(ctx, `DELETE FROM test_cases WHERE id=$1`, id)
    return err
}

// --- admin views ---

func (s *PostgresStore) ListUsers(ctx context.Context) ([]UserSummary, error) {
    rows, err := s.db.Query(ctx, `SELECT id, email, role FROM users ORDER BY id`)
    if err != nil {
        return nil, fmt.Errorf("list users: %w", err)
    }
    defer rows.Close()
    var out []UserSummary
    for rows.Next() {
        var u UserSummary
        if err := rows.Scan(&u.ID, &u.Email, &u.Role); err != nil {
            return nil, fmt.Errorf("scan user: %w", err)
        }
        out = append(out, u)
    }
    return out, nil
}

func (s *PostgresStore) ListAllSubmissions(ctx context.Context) ([]SubmissionLogItem, error) {
    rows, err := s.db.Query(ctx,
        `SELECT s.id, p.title, COALESCE(u.email,''), s.language, s.verdict, s.created_at
         FROM submissions s
         JOIN problems p ON s.problem_id = p.id
         LEFT JOIN users u ON s.user_id = u.id
         ORDER BY s.created_at DESC`)
    if err != nil {
        return nil, fmt.Errorf("list submissions: %w", err)
    }
    defer rows.Close()
    var out []SubmissionLogItem
    for rows.Next() {
        var item SubmissionLogItem
        var createdAt time.Time
        if err := rows.Scan(&item.ID, &item.ProblemTitle, &item.UserEmail,
            &item.Language, &item.Verdict, &createdAt); err != nil {
            return nil, fmt.Errorf("scan submission log: %w", err)
        }
        item.CreatedAt = createdAt.Format(time.RFC3339)
        out = append(out, item)
    }
    return out, nil
}

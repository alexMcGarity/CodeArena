package server

import "context"

type Store interface {
	// problems (public)
	FetchProblems(ctx context.Context) ([]Problem, error)
	FetchProblemByID(ctx context.Context, id int) (*Problem, error)

	// submissions (user)
	InsertSubmission(ctx context.Context, req SubmissionRequest, userID int) (int, error)
	UpdateSubmissionVerdict(ctx context.Context, id int, verdict string) error
	GetSubmissionStatus(ctx context.Context, id int) (SubmissionStatus, error)

	// users
	CreateUser(ctx context.Context, email, passwordHash string) (int, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetSubmissionsByUserID(ctx context.Context, userID int) ([]SubmissionHistoryItem, error)
	SeedAdmin(ctx context.Context, email, password string) error

	// admin — problems
	CreateProblem(ctx context.Context, p Problem) (int, error)
	UpdateProblem(ctx context.Context, p Problem) error
	DeleteProblem(ctx context.Context, id int) error

	// admin — test cases
	GetTestCasesByProblemID(ctx context.Context, problemID int) ([]TestCase, error)
	CreateTestCase(ctx context.Context, problemID int, input, expected string) (int, error)
	UpdateTestCase(ctx context.Context, id int, input, expected string) error
	DeleteTestCase(ctx context.Context, id int) error

	// admin — views
	ListUsers(ctx context.Context) ([]UserSummary, error)
	ListAllSubmissions(ctx context.Context) ([]SubmissionLogItem, error)
}

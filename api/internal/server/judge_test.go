package server

import (
	"context"
	"testing"
)

var mockTestCases = []TestCase{
	{ID: 1, ProblemID: 1, Input: "", Expected: "Hello World\n"},
}

// --- runJudgeMock ---

func TestMockJudgeAccepted(t *testing.T) {
	code := "#include <bits/stdc++.h>\nusing namespace std;\nint main(){cout<<\"Hello World\\n\";}"
	if got := runJudgeMock("cpp", code); got != "accepted" {
		t.Errorf("expected accepted, got %q", got)
	}
}

func TestMockJudgeWrongAnswer(t *testing.T) {
	code := "#include <iostream>\nint main(){return 0;}"
	if got := runJudgeMock("cpp", code); got != "wrong_answer" {
		t.Errorf("expected wrong_answer, got %q", got)
	}
}

func TestMockJudgeCompileError(t *testing.T) {
	if got := runJudgeMock("cpp", "not valid C++"); got != "compile_error" {
		t.Errorf("expected compile_error, got %q", got)
	}
}

func TestMockJudgePythonAccepted(t *testing.T) {
	code := `print("Hello World")`
	if got := runJudgeMock("python", code); got != "accepted" {
		t.Errorf("expected accepted, got %q", got)
	}
}

func TestMockJudgePythonWrongAnswer(t *testing.T) {
	code := `print("wrong")`
	if got := runJudgeMock("python", code); got != "wrong_answer" {
		t.Errorf("expected wrong_answer, got %q", got)
	}
}

// --- runJudge language validation ---

func TestRunJudgeUnsupportedLanguage(t *testing.T) {
	t.Setenv("JUDGE_MOCK", "true")
	verdict, err := runJudge(context.Background(), "java", "public class Main{}", mockTestCases, 2000, 256)
	if err == nil {
		t.Error("expected error for unsupported language, got nil")
	}
	if verdict != "unsupported_language" {
		t.Errorf("expected unsupported_language, got %q", verdict)
	}
}

func TestRunJudgeCppAlias(t *testing.T) {
	t.Setenv("JUDGE_MOCK", "true")
	for _, lang := range []string{"cpp", "c++", "CPP", "C++"} {
		code := "#include <bits/stdc++.h>\nusing namespace std;\nint main(){cout<<\"Hello World\\n\";}"
		verdict, err := runJudge(context.Background(), lang, code, mockTestCases, 2000, 256)
		if err != nil {
			t.Errorf("lang=%q: unexpected error: %v", lang, err)
		}
		if verdict != "accepted" {
			t.Errorf("lang=%q: expected accepted, got %q", lang, verdict)
		}
	}
}

func TestRunJudgePythonMockMode(t *testing.T) {
	t.Setenv("JUDGE_MOCK", "true")
	code := `print("Hello World")`
	verdict, err := runJudge(context.Background(), "python", code, mockTestCases, 2000, 256)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict != "accepted" {
		t.Errorf("expected accepted, got %q", verdict)
	}
}

func TestRunJudgeMockMode(t *testing.T) {
	t.Setenv("JUDGE_MOCK", "true")
	code := "#include <bits/stdc++.h>\nusing namespace std;\nint main(){cout<<\"Hello World\\n\";}"
	verdict, err := runJudge(context.Background(), "cpp", code, mockTestCases, 2000, 256)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict != "accepted" {
		t.Errorf("expected accepted, got %q", verdict)
	}
}

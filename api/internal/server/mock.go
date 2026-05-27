package server

import "strings"

// runJudgeMock simulates judge verdicts for testing without the real binary.
// Heuristics: if code contains "Hello World" + "cout" or "print" → accepted;
// if it has an include/import → wrong_answer; otherwise → compile_error.
func runJudgeMock(language, code string) string {
    code = strings.TrimSpace(code)
    lang := strings.ToLower(language)

    if lang == "python" || lang == "python3" {
        if strings.Contains(code, "Hello World") && strings.Contains(code, "print") {
            return "accepted"
        }
        if strings.Contains(code, "print") {
            return "wrong_answer"
        }
        return "compile_error"
    }

    // C++
    if strings.Contains(code, "Hello World") && strings.Contains(code, "cout") {
        return "accepted"
    }
    if strings.Contains(code, "#include") {
        return "wrong_answer"
    }
    return "compile_error"
}

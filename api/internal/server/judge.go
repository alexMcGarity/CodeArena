package server

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
)

type judgePayload struct {
    Language      string      `json:"language"`
    Code          string      `json:"code"`
    TestCases     []TestCase  `json:"test_cases"`
    TimeLimitMs   int         `json:"time_limit_ms"`
    MemoryLimitMb int         `json:"memory_limit_mb"`
}

type judgeResult struct {
    Verdict string `json:"verdict"`
}

func runJudge(ctx context.Context, language, code string, testCases []TestCase, timeLimitMs, memLimitMb int) (string, error) {
    normalized := strings.ToLower(strings.TrimSpace(language))
    if normalized != "cpp" && normalized != "c++" && normalized != "python" && normalized != "python3" {
        return "unsupported_language", fmt.Errorf("unsupported language: %s", language)
    }

    if os.Getenv("JUDGE_MOCK") == "true" {
        return runJudgeMock(language, code), nil
    }

    judgeBinary := os.Getenv("JUDGE_BINARY_PATH")
    if judgeBinary == "" {
        judgeBinary = "./codearena-judge"
    }
    judgeBinary = filepath.Clean(judgeBinary)

    payload := judgePayload{
        Language:      normalized,
        Code:          code,
        TestCases:     testCases,
        TimeLimitMs:   timeLimitMs,
        MemoryLimitMb: memLimitMb,
    }
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return "judge_error", fmt.Errorf("marshal payload: %w", err)
    }

    cmd := exec.CommandContext(ctx, judgeBinary)
    cmd.Stdin = strings.NewReader(string(payloadBytes))
    cmd.Env = append(os.Environ(), "LANG=C")

    output, err := cmd.Output()
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            output = exitErr.Stderr
        } else {
            return "judge_error", fmt.Errorf("run judge: %w", err)
        }
    }

    var result judgeResult
    if err := json.Unmarshal(output, &result); err != nil {
        return "judge_error", fmt.Errorf("invalid judge output: %w, raw=%s", err, output)
    }
    return result.Verdict, nil
}

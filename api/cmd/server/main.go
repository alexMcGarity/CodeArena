package main

import (
    "log"

    "github.com/codearena/api/internal/server"
)

func main() {
    app := server.New()
    log.Println("starting CodeArena API on :8080")
    if err := app.ListenAndServe(); err != nil {
        log.Fatalf("server failed: %v", err)
    }
}

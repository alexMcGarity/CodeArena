# Stage 1 — build C++ judge (needs internet for nlohmann/json FetchContent)
FROM ubuntu:24.04 AS judge-builder
RUN apt-get update && apt-get install -y --no-install-recommends \
        g++ cmake make git ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /judge
COPY judge/ .
RUN cmake -B build -DCMAKE_BUILD_TYPE=Release \
    && cmake --build build -j"$(nproc)"

# Stage 2 — build Go API
FROM golang:1.25-bookworm AS api-builder
WORKDIR /app
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# Stage 3 — runtime image
FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends \
        g++ python3 ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=judge-builder /judge/build/codearena-judge ./codearena-judge
COPY --from=api-builder   /app/server                  ./server
COPY db/migrations ./migrations

ENV JUDGE_BINARY_PATH=/app/codearena-judge
EXPOSE 8080
CMD ["./server"]

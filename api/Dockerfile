# Development convenience image — builds judge + API together.
# For production use the root Dockerfile instead.
FROM golang:1.25-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential cmake python3 git ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY . .
RUN go mod download
RUN cmake -S /app/judge -B /tmp/judge-build -DCMAKE_BUILD_TYPE=Release \
    && cmake --build /tmp/judge-build -j"$(nproc)" \
    && mv /tmp/judge-build/codearena-judge /app/codearena-judge
ENV JUDGE_BINARY_PATH=/app/codearena-judge
WORKDIR /app/api
CMD ["go", "run", "./cmd/server"]

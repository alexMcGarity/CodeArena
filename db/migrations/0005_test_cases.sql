-- Phase 3: test cases per problem

CREATE TABLE test_cases (
    id         SERIAL PRIMARY KEY,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    input      TEXT NOT NULL DEFAULT '',
    expected   TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

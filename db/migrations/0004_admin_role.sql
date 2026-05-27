-- Phase 3: admin role on users, time/memory limits on problems

ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user';

ALTER TABLE problems
    ADD COLUMN time_limit_ms   INTEGER NOT NULL DEFAULT 2000,
    ADD COLUMN memory_limit_mb INTEGER NOT NULL DEFAULT 256;

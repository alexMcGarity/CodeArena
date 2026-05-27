-- Phase 2: link submissions to users

ALTER TABLE submissions ADD COLUMN user_id INTEGER REFERENCES users(id);

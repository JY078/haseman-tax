ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMP;
UPDATE user_sessions SET last_active_at = created_at WHERE last_active_at IS NULL;

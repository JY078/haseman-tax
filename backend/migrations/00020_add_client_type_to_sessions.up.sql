ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS client_type VARCHAR(20) DEFAULT 'web';
UPDATE user_sessions SET client_type = 'web' WHERE client_type IS NULL OR client_type = '';

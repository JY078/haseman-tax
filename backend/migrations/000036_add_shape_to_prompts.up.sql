ALTER TABLE prompts ADD COLUMN IF NOT EXISTS shape VARCHAR(20) NOT NULL DEFAULT 'square';
CREATE INDEX IF NOT EXISTS idx_prompts_shape ON prompts(shape);

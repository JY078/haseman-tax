-- Drop foreign key constraint to allow group_id = 0 (ungrouped)
ALTER TABLE prompts DROP CONSTRAINT IF EXISTS prompts_group_id_fkey;

-- Allow group_id to be 0 (ungrouped sentinel)
ALTER TABLE prompts ALTER COLUMN group_id DROP NOT NULL;
ALTER TABLE prompts ALTER COLUMN group_id SET DEFAULT 0;

-- Add status: approved | pending | offline
ALTER TABLE prompts ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'approved';

-- Add source: admin | user
ALTER TABLE prompts ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'admin';

CREATE INDEX IF NOT EXISTS idx_prompts_status ON prompts(status);

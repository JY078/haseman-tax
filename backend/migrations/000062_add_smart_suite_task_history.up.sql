ALTER TABLE smart_suite_conversations
    ADD COLUMN IF NOT EXISTS task_ids jsonb NOT NULL DEFAULT '[]'::jsonb;


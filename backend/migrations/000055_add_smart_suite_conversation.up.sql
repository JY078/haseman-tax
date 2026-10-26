ALTER TABLE ecom_suite_tasks
    ADD COLUMN IF NOT EXISTS visual_style text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS style_requirements text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS style_reference_urls jsonb NOT NULL DEFAULT '[]'::jsonb;

CREATE TABLE IF NOT EXISTS smart_suite_conversations (
    id bigserial PRIMARY KEY,
    session_id varchar(64) NOT NULL,
    user_id bigint NOT NULL,
    task_id varchar(64) NOT NULL DEFAULT '',
    context jsonb NOT NULL DEFAULT '{}'::jsonb,
    messages jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_smart_suite_conversations_session_id
    ON smart_suite_conversations(session_id);
CREATE INDEX IF NOT EXISTS idx_smart_suite_conversations_user_id
    ON smart_suite_conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_smart_suite_conversations_updated_at
    ON smart_suite_conversations(updated_at);

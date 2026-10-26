CREATE TABLE IF NOT EXISTS work_requests (
    id bigserial PRIMARY KEY,
    work_no varchar(32) NOT NULL UNIQUE,
    type varchar(20) NOT NULL DEFAULT 'video' CHECK (type = 'video'),
    title varchar(200) NOT NULL,
    status varchar(30) NOT NULL DEFAULT 'pending',
    creator_id bigint NOT NULL,
    assignee_id bigint,
    progress integer NOT NULL DEFAULT 0,
    progress_before_review integer NOT NULL DEFAULT 10,
    content jsonb NOT NULL DEFAULT '{}'::jsonb,
    delivery_note text NOT NULL DEFAULT '',
    delivery_paths jsonb NOT NULL DEFAULT '[]'::jsonb,
    rejection_reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_work_requests_type ON work_requests(type);
CREATE INDEX IF NOT EXISTS idx_work_requests_status ON work_requests(status);
CREATE INDEX IF NOT EXISTS idx_work_requests_creator_id ON work_requests(creator_id);
CREATE INDEX IF NOT EXISTS idx_work_requests_assignee_id ON work_requests(assignee_id);
CREATE INDEX IF NOT EXISTS idx_work_requests_created_at ON work_requests(created_at DESC);

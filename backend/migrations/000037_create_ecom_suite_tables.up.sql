CREATE TABLE IF NOT EXISTS ecom_suite_tasks (
    id BIGSERIAL PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL DEFAULT 0,
    platform VARCHAR(50) NOT NULL DEFAULT '',
    language VARCHAR(20) NOT NULL DEFAULT '',
    country_code VARCHAR(20) NOT NULL DEFAULT '',
    country_name VARCHAR(100) NOT NULL DEFAULT '',
    output_ratio VARCHAR(20) NOT NULL DEFAULT '1:1',
    selected_a_plus_template_id BIGINT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    error_message TEXT NOT NULL DEFAULT '',
    plan_snapshot JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ecom_suite_tasks_user_id ON ecom_suite_tasks (user_id);
CREATE INDEX IF NOT EXISTS idx_ecom_suite_tasks_status ON ecom_suite_tasks (status);
CREATE INDEX IF NOT EXISTS idx_ecom_suite_tasks_created_at ON ecom_suite_tasks (created_at);

CREATE TABLE IF NOT EXISTS ecom_suite_images (
    id BIGSERIAL PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    image_type VARCHAR(30) NOT NULL DEFAULT '',
    image_index INTEGER NOT NULL DEFAULT 1,
    role VARCHAR(100) NOT NULL DEFAULT '',
    title VARCHAR(500) NOT NULL DEFAULT '',
    subtitle VARCHAR(500) NOT NULL DEFAULT '',
    layout_instruction TEXT NOT NULL DEFAULT '',
    reference_rule JSONB NULL,
    prompt_draft TEXT NOT NULL DEFAULT '',
    prompt_final TEXT NOT NULL DEFAULT '',
    a_plus_template_id BIGINT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    image_url VARCHAR(1000) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ecom_suite_images_task_id ON ecom_suite_images (task_id);
CREATE INDEX IF NOT EXISTS idx_ecom_suite_images_task_type ON ecom_suite_images (task_id, image_type);

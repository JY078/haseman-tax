-- 产品替换任务表
CREATE TABLE IF NOT EXISTS product_replace_tasks (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    task_id VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(20) DEFAULT 'queued',
    error_message TEXT DEFAULT '',
    is_saved_to_design BOOLEAN DEFAULT FALSE,
    saved_at TIMESTAMP NULL,
    mode VARCHAR(20) DEFAULT 'water',
    aspect_ratio VARCHAR(20) DEFAULT '1:1',
    image_count INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_product_replace_tasks_user_id ON product_replace_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_product_replace_tasks_task_id ON product_replace_tasks(task_id);
CREATE INDEX IF NOT EXISTS idx_product_replace_tasks_created_at ON product_replace_tasks(created_at);

-- 产品替换任务图片表
CREATE TABLE IF NOT EXISTS product_replace_images (
    id BIGSERIAL PRIMARY KEY,
    task_id_ref VARCHAR(64) NOT NULL,
    sort INT DEFAULT 0,
    image_url VARCHAR(500) NOT NULL,
    error_msg TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_product_replace_images_task_id_ref ON product_replace_images(task_id_ref);

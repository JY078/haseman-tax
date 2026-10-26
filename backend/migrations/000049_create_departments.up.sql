CREATE TABLE IF NOT EXISTS departments (
    id SERIAL PRIMARY KEY,
    platform VARCHAR(50) NOT NULL DEFAULT '',
    name VARCHAR(100) NOT NULL,
    sub_name VARCHAR(100) NOT NULL DEFAULT '',
    nas_region_path VARCHAR(200) NOT NULL DEFAULT '',
    sort INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_platform_name_sub_name
    ON departments(platform, name, sub_name);

CREATE INDEX IF NOT EXISTS idx_departments_sort
    ON departments(sort, id);

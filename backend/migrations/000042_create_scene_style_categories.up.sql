CREATE TABLE IF NOT EXISTS scene_style_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    icon VARCHAR(50) NOT NULL DEFAULT '',
    sort INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_scene_style_categories_name ON scene_style_categories (name);
CREATE INDEX IF NOT EXISTS idx_scene_style_categories_sort ON scene_style_categories (sort);

ALTER TABLE scene_style_tags ADD COLUMN IF NOT EXISTS scene_id INT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_scene_style_tags_scene_id ON scene_style_tags (scene_id);
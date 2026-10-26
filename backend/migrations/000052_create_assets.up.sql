CREATE TABLE IF NOT EXISTS assets (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL DEFAULT '',
    type VARCHAR(80) NOT NULL DEFAULT '',
    category VARCHAR(120) NOT NULL DEFAULT '',
    owner_type VARCHAR(30) NOT NULL DEFAULT 'personal',
    owner_id BIGINT NOT NULL DEFAULT 0,
    user_id BIGINT NOT NULL DEFAULT 0,
    department VARCHAR(100) NOT NULL DEFAULT '',
    visibility VARCHAR(30) NOT NULL DEFAULT 'private',
    file_url VARCHAR(1000) NOT NULL DEFAULT '',
    cover_url VARCHAR(1000) NOT NULL DEFAULT '',
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    file_type VARCHAR(80) NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '[]',
    usage_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assets_owner_type ON assets(owner_type);
CREATE INDEX IF NOT EXISTS idx_assets_owner_id ON assets(owner_id);
CREATE INDEX IF NOT EXISTS idx_assets_user_id ON assets(user_id);
CREATE INDEX IF NOT EXISTS idx_assets_department ON assets(department);
CREATE INDEX IF NOT EXISTS idx_assets_visibility ON assets(visibility);
CREATE INDEX IF NOT EXISTS idx_assets_updated_at ON assets(updated_at DESC);

CREATE TABLE IF NOT EXISTS asset_favorites (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    source_type VARCHAR(80) NOT NULL,
    source_id VARCHAR(120) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_favorites_unique
    ON asset_favorites(user_id, source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_asset_favorites_user_id
    ON asset_favorites(user_id);

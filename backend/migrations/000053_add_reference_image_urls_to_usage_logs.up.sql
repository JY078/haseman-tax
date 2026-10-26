ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS reference_image_urls TEXT NOT NULL DEFAULT '[]';

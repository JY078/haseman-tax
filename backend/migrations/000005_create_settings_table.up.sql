CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO settings (key, value, description) VALUES
('site_name', 'AD Image 管理系统', 'System Name'),
('site_description', '专业的广告图片管理平台', 'System Description'),
('access_url', 'http://localhost:5173', 'Access URL'),
('chat_models', '["gpt-3.5-turbo","gpt-4"]', 'Chat Models'),
('drawing_models', '["dall-e-3","midjourney"]', 'Drawing Models'),
('video_models', '["sora","runway"]', 'Video Models')
ON CONFLICT (key) DO NOTHING;

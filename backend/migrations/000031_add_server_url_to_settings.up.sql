INSERT INTO settings (key, value, description)
VALUES ('server_url', '', 'API Server URL (system backend public URL, used for Tencent AIGC reference image access)')
ON CONFLICT (key) DO NOTHING;

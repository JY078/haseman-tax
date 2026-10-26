-- 将 ai_models 计价/用量字段由 INTEGER 改为 NUMERIC(12,6)，与 GORM decimal(12,6) 对齐
ALTER TABLE ai_models ALTER COLUMN price_per_use TYPE NUMERIC(12, 6);
ALTER TABLE ai_models ALTER COLUMN price_input TYPE NUMERIC(12, 6);
ALTER TABLE ai_models ALTER COLUMN price_output TYPE NUMERIC(12, 6);
ALTER TABLE ai_models ALTER COLUMN usage_count TYPE NUMERIC(12, 6);

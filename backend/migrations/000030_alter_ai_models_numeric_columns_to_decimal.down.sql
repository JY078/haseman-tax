-- 回滚：恢复为 INTEGER（小数按四舍五入取整）
ALTER TABLE ai_models ALTER COLUMN price_per_use TYPE INTEGER USING (ROUND(price_per_use)::INTEGER);
ALTER TABLE ai_models ALTER COLUMN price_input TYPE INTEGER USING (ROUND(price_input)::INTEGER);
ALTER TABLE ai_models ALTER COLUMN price_output TYPE INTEGER USING (ROUND(price_output)::INTEGER);
ALTER TABLE ai_models ALTER COLUMN usage_count TYPE INTEGER USING (ROUND(usage_count)::INTEGER);

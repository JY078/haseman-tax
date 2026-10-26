ALTER TABLE ai_models RENAME COLUMN price TO price_per_use;
ALTER TABLE ai_models ADD COLUMN billing_mode VARCHAR(20) DEFAULT 'per_use'; -- 'per_use', 'per_token'
ALTER TABLE ai_models ADD COLUMN price_input INTEGER DEFAULT 0; -- per 1k tokens or similar unit
ALTER TABLE ai_models ADD COLUMN price_output INTEGER DEFAULT 0; -- per 1k tokens or similar unit

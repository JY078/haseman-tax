ALTER TABLE ai_models DROP COLUMN price_output;
ALTER TABLE ai_models DROP COLUMN price_input;
ALTER TABLE ai_models DROP COLUMN billing_mode;
ALTER TABLE ai_models RENAME COLUMN price_per_use TO price;

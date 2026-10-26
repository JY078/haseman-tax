ALTER TABLE ai_models RENAME COLUMN type TO types;
ALTER TABLE ai_models ALTER COLUMN types TYPE TEXT;
UPDATE ai_models SET types = '["' || types || '"]';

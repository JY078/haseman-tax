ALTER TABLE ecom_suite_tasks
ADD COLUMN IF NOT EXISTS product_reference_urls JSONB NOT NULL DEFAULT '[]'::jsonb;

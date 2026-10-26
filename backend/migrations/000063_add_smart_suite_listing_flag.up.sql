ALTER TABLE ecom_suite_tasks
    ADD COLUMN IF NOT EXISTS include_listing boolean NOT NULL DEFAULT false;

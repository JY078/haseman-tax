ALTER TABLE ecom_suite_tasks
    ADD COLUMN IF NOT EXISTS drawing_model varchar(255) NOT NULL DEFAULT '';

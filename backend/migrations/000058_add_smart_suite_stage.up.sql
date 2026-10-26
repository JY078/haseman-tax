ALTER TABLE ecom_suite_tasks
    ADD COLUMN IF NOT EXISTS smart_stage varchar(40) NOT NULL DEFAULT '';

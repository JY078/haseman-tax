ALTER TABLE ecom_suite_tasks
    ADD COLUMN IF NOT EXISTS current_action text NOT NULL DEFAULT '';

-- Revert: add back the constraint
ALTER TABLE ecom_suite_tasks ADD CONSTRAINT uni_ecom_suite_tasks_task_id UNIQUE (task_id);

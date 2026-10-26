-- Drop the legacy constraint/index if it exists, then recreate properly
DO $$
BEGIN
    -- Try to drop constraint if it exists (legacy GORM naming)
    ALTER TABLE ecom_suite_tasks DROP CONSTRAINT IF EXISTS uni_ecom_suite_tasks_task_id;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Constraint not found, skipping: %', SQLERRM;
END
$$;

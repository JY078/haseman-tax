DROP TABLE IF EXISTS smart_suite_conversations;

ALTER TABLE ecom_suite_tasks
    DROP COLUMN IF EXISTS style_reference_urls,
    DROP COLUMN IF EXISTS style_requirements,
    DROP COLUMN IF EXISTS visual_style;

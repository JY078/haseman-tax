-- 添加 visual_strategy、typography_instruction 字段到 ecom_suite_images 表
ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS visual_strategy TEXT NOT NULL DEFAULT '';
ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS typography_instruction TEXT NOT NULL DEFAULT '';
ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS primary_color VARCHAR(200) NOT NULL DEFAULT '';
ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS secondary_color VARCHAR(200) NOT NULL DEFAULT '';
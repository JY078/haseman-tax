-- 回滚：删除 ecom_suite_images 表新增的字段
ALTER TABLE ecom_suite_images DROP COLUMN IF EXISTS visual_strategy;
ALTER TABLE ecom_suite_images DROP COLUMN IF EXISTS typography_instruction;
ALTER TABLE ecom_suite_images DROP COLUMN IF EXISTS primary_color;
ALTER TABLE ecom_suite_images DROP COLUMN IF EXISTS secondary_color;
-- 回滚 cta_direction、style、include_logo 字段
ALTER TABLE ecom_suite_images DROP COLUMN IF EXISTS cta_direction;
ALTER TABLE ecom_suite_images DROP COLUMN IF EXISTS style;
ALTER TABLE ecom_suite_images DROP COLUMN IF EXISTS include_logo;

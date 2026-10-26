-- 添加 cta_direction、style、include_logo 字段到 ecom_suite_images 表
ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS cta_direction TEXT NOT NULL DEFAULT '';
ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS style TEXT NOT NULL DEFAULT '';
ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS include_logo BOOLEAN NOT NULL DEFAULT FALSE;

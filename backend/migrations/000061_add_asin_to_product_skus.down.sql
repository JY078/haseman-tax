DROP INDEX IF EXISTS idx_product_skus_asin;
ALTER TABLE product_skus DROP COLUMN IF EXISTS asin;

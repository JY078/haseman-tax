ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS asin varchar(20) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_product_skus_asin ON product_skus(asin);

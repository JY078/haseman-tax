DROP TABLE IF EXISTS tiktok_sku_prices;
ALTER TABLE product_skus DROP COLUMN IF EXISTS price_updated_at;
ALTER TABLE product_skus DROP COLUMN IF EXISTS amazon_sale_price;
ALTER TABLE product_skus DROP COLUMN IF EXISTS amazon_price;

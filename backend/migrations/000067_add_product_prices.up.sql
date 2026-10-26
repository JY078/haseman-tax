ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS amazon_price numeric(10,2);
ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS amazon_sale_price numeric(10,2);
ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS price_updated_at timestamptz;
CREATE TABLE IF NOT EXISTS tiktok_sku_prices (
    platform_sku varchar(160) PRIMARY KEY,
    pid varchar(32) NOT NULL DEFAULT '',
    price numeric(10,2),
    updated_by bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT NOW()
);

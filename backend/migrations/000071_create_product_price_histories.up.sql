CREATE TABLE IF NOT EXISTS product_price_histories (
  id BIGSERIAL PRIMARY KEY,
  product_sku_id BIGINT NOT NULL REFERENCES product_skus(id) ON DELETE CASCADE,
  sku VARCHAR(120) NOT NULL,
  price_date DATE NOT NULL,
  amazon_price NUMERIC(10,2),
  amazon_sale_price NUMERIC(10,2),
  amazon_best_deal NUMERIC(10,2),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT product_price_histories_sku_day_unique UNIQUE (product_sku_id, price_date)
);
CREATE INDEX IF NOT EXISTS idx_product_price_histories_date ON product_price_histories(price_date);

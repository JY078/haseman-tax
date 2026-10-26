CREATE TABLE IF NOT EXISTS product_color_families (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    image_url VARCHAR(500) NOT NULL DEFAULT '',
    sort INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_product_color_families_sort ON product_color_families (sort);
CREATE INDEX IF NOT EXISTS idx_product_color_families_created_at ON product_color_families (created_at);

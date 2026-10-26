CREATE TABLE IF NOT EXISTS product_prompts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    main_image_prompt TEXT NOT NULL DEFAULT '',
    detail_page_prompt TEXT NOT NULL DEFAULT '',
    custom_prompt_suffix TEXT NOT NULL DEFAULT '',
    sort INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_product_prompts_sort ON product_prompts(sort);

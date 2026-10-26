-- Rollback: Remove white background image URL column
ALTER TABLE product_prompts DROP COLUMN IF EXISTS white_bg_image_url;
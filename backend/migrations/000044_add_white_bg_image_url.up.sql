-- Add white background image URL for AI image generation reference
ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS white_bg_image_url VARCHAR(500) DEFAULT '';
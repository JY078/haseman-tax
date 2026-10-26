ALTER TABLE ecom_suite_images
    ADD COLUMN IF NOT EXISTS review_source_image_url varchar(1000) NOT NULL DEFAULT '';

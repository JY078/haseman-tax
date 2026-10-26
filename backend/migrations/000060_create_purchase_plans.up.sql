CREATE TABLE IF NOT EXISTS purchase_plans (
    id bigserial PRIMARY KEY,
    product_sku_id bigint NOT NULL,
    product_name varchar(255) NOT NULL,
    sku varchar(120) NOT NULL,
    planned_quantity integer NOT NULL CHECK (planned_quantity > 0),
    units_per_carton integer NOT NULL CHECK (units_per_carton > 0),
    carton_count integer NOT NULL CHECK (carton_count > 0),
    expected_arrival_date date NOT NULL,
    remark text NOT NULL DEFAULT '',
    applicant_id bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_purchase_plans_product_sku_id ON purchase_plans(product_sku_id);
CREATE INDEX IF NOT EXISTS idx_purchase_plans_sku ON purchase_plans(sku);
CREATE INDEX IF NOT EXISTS idx_purchase_plans_applicant_id ON purchase_plans(applicant_id);
CREATE INDEX IF NOT EXISTS idx_purchase_plans_created_at ON purchase_plans(created_at DESC);

CREATE TABLE IF NOT EXISTS recharge_orders (
    id SERIAL PRIMARY KEY,
    out_trade_no VARCHAR(64) NOT NULL UNIQUE,
    user_id INTEGER NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    points DECIMAL(18,6) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    qr_code_url TEXT NOT NULL DEFAULT '',
    alipay_trade_no VARCHAR(64) NOT NULL DEFAULT '',
    paid_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_recharge_orders_user_id ON recharge_orders(user_id);
CREATE INDEX IF NOT EXISTS idx_recharge_orders_out_trade_no ON recharge_orders(out_trade_no);
CREATE INDEX IF NOT EXISTS idx_recharge_orders_status ON recharge_orders(status);

CREATE TABLE point_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    amount INTEGER NOT NULL,
    balance INTEGER NOT NULL,
    type VARCHAR(30) NOT NULL,
    source VARCHAR(50) DEFAULT '',
    remark VARCHAR(255) DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_point_logs_user_id ON point_logs(user_id);
CREATE INDEX idx_point_logs_created_at ON point_logs(created_at);

INSERT INTO users (
    name,
    phone,
    password,
    available_balance,
    frozen_balance,
    created_at,
    updated_at
) VALUES (
    'admin',
    '0987654321',
    '$2a$10$aOrQBf5WiR3uyM19nYUH6OcKa7iuwhT7npKTzT.04wgpUp..qW8PW', -- 'a12345678'
    10000.00,  -- 預設可用餘額 10,000
    0.00,      -- 預設凍結餘額 0
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
);

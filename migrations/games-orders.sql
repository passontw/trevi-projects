CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,                -- 系統ID
    order_id VARCHAR(50) NOT NULL UNIQUE,    -- 業務訂單ID (例如: SLOT202502230001)
    user_id INTEGER NOT NULL,                -- 用戶ID
    type order_type NOT NULL,                -- 訂單類型
    status order_status NOT NULL,            -- 訂單狀態
    bet_amount DECIMAL(20,2) NOT NULL,       -- 下注金額
    win_amount DECIMAL(20,2) NOT NULL,       -- 獲勝金額
    game_result JSONB NOT NULL,              -- 遊戲結果
    balance_record_ids _int4,            -- 相關餘額記錄ID
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,    -- 完成時間
    remark JSONB,                            -- 備註信息
    
    CONSTRAINT fk_orders_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE RESTRICT
);

CREATE UNIQUE INDEX idx_orders_order_id ON orders(order_id);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_orders_type_status ON orders(type, status);

CREATE SEQUENCE IF NOT EXISTS order_seq_slot
    INCREMENT 1
    MINVALUE 1
    MAXVALUE 9999999999
    CYCLE;

CREATE OR REPLACE FUNCTION generate_order_id(prefix TEXT)
RETURNS TEXT AS $$
DECLARE
    seq_num BIGINT;
    date_part TEXT;
BEGIN
    SELECT nextval('order_seq_slot') INTO seq_num;
    date_part := to_char(CURRENT_TIMESTAMP, 'YYYYMMDD');    
    RETURN prefix || date_part || LPAD(seq_num::TEXT, 4, '0');
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TYPE balance_operation_type AS ENUM (
    'ADD',      -- 新增額度
    'DEDUCT',   -- 扣除額度
    'FREEZE',   -- 凍結額度
    'UNFREEZE'  -- 解凍額度
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    name VARCHAR(20) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    password VARCHAR(200) NOT NULL,
    available_balance DECIMAL(20,2) NOT NULL DEFAULT 0.00 CHECK (available_balance >= 0),
    frozen_balance DECIMAL(20,2) NOT NULL DEFAULT 0.00 CHECK (frozen_balance >= 0)
);

CREATE UNIQUE INDEX idx_users_phone ON users (phone) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_balances ON users (available_balance, frozen_balance);

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS balance_records (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type balance_operation_type NOT NULL,
    amount DECIMAL(20,2) NOT NULL,
    before_balance DECIMAL(20,2) NOT NULL,
    after_balance DECIMAL(20,2) NOT NULL,
    before_frozen DECIMAL(20,2) NOT NULL,
    after_frozen DECIMAL(20,2) NOT NULL,
    description TEXT,
    operator VARCHAR(50),
    reference_id VARCHAR(100),
    remark JSONB  -- 額外資訊，使用 JSONB 以支援彈性擴展
);

CREATE INDEX idx_balance_records_user ON balance_records (user_id);
CREATE INDEX idx_balance_records_type ON balance_records (type);
CREATE INDEX idx_balance_records_created ON balance_records (created_at);
CREATE INDEX idx_balance_records_reference ON balance_records (reference_id);
CREATE INDEX idx_balance_records_remark ON balance_records USING GIN (remark);

CREATE TRIGGER update_balance_records_updated_at
    BEFORE UPDATE ON balance_records
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE users IS '用戶資料表';
COMMENT ON COLUMN users.available_balance IS '可用餘額';
COMMENT ON COLUMN users.frozen_balance IS '凍結餘額';

COMMENT ON TABLE balance_records IS '餘額變動記錄表';
COMMENT ON COLUMN balance_records.type IS '操作類型: ADD(新增), DEDUCT(扣除), FREEZE(凍結), UNFREEZE(解凍)';
COMMENT ON COLUMN balance_records.amount IS '操作金額';
COMMENT ON COLUMN balance_records.before_balance IS '操作前可用餘額';
COMMENT ON COLUMN balance_records.after_balance IS '操作後可用餘額';
COMMENT ON COLUMN balance_records.before_frozen IS '操作前凍結餘額';
COMMENT ON COLUMN balance_records.after_frozen IS '操作後凍結餘額';
COMMENT ON COLUMN balance_records.description IS '操作描述';
COMMENT ON COLUMN balance_records.operator IS '操作人';
COMMENT ON COLUMN balance_records.reference_id IS '關聯交易ID';
COMMENT ON COLUMN balance_records.remark IS '額外資訊(JSON格式)';

CREATE OR REPLACE FUNCTION process_balance_change(
    p_user_id INTEGER,
    p_type balance_operation_type,
    p_amount DECIMAL,
    p_description TEXT,
    p_operator VARCHAR,
    p_reference_id VARCHAR,
    p_remark JSONB DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    v_before_balance DECIMAL;
    v_before_frozen DECIMAL;
    v_after_balance DECIMAL;
    v_after_frozen DECIMAL;

BEGIN
    SELECT available_balance, frozen_balance
    INTO v_before_balance, v_before_frozen
    FROM users
    WHERE id = p_user_id
    FOR UPDATE;

    CASE p_type
        WHEN 'ADD' THEN
            UPDATE users
            SET available_balance = available_balance + p_amount
            WHERE id = p_user_id;
            v_after_balance := v_before_balance + p_amount;
            v_after_frozen := v_before_frozen;
            
        WHEN 'DEDUCT' THEN
            IF v_before_balance < p_amount THEN
                RAISE EXCEPTION '餘額不足';
            END IF;
            UPDATE users
            SET available_balance = available_balance - p_amount
            WHERE id = p_user_id;
            v_after_balance := v_before_balance - p_amount;
            v_after_frozen := v_before_frozen;
            
        WHEN 'FREEZE' THEN
            IF v_before_balance < p_amount THEN
                RAISE EXCEPTION '可用餘額不足以凍結';
            END IF;
            UPDATE users
            SET available_balance = available_balance - p_amount,
                frozen_balance = frozen_balance + p_amount
            WHERE id = p_user_id;
            v_after_balance := v_before_balance - p_amount;
            v_after_frozen := v_before_frozen + p_amount;
            
        WHEN 'UNFREEZE' THEN
            IF v_before_frozen < p_amount THEN
                RAISE EXCEPTION '凍結餘額不足';
            END IF;
            UPDATE users
            SET available_balance = available_balance + p_amount,
                frozen_balance = frozen_balance - p_amount
            WHERE id = p_user_id;
            v_after_balance := v_before_balance + p_amount;
            v_after_frozen := v_before_frozen - p_amount;
    END CASE;

    INSERT INTO balance_records (
        user_id, type, amount,
        before_balance, after_balance,
        before_frozen, after_frozen,
        description, operator, reference_id, remark
    ) VALUES (
        p_user_id, p_type, p_amount,
        v_before_balance, v_after_balance,
        v_before_frozen, v_after_frozen,
        p_description, p_operator, p_reference_id, p_remark
    );

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
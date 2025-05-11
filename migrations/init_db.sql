-- 遊戲基本信息表
CREATE TABLE games (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    game_id VARCHAR(100) NOT NULL UNIQUE,
    room_id VARCHAR(20) NOT NULL,            -- 房間 ID，如 SG01, SG02 等
    state VARCHAR(30) NOT NULL,             -- 遊戲狀態如 PREPARATION, BETTING, DRAWING 等
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NULL,
    
    -- 遊戲設定相關
    has_jackpot BOOLEAN NOT NULL DEFAULT FALSE,
    jackpot_amount DECIMAL(18, 2) DEFAULT 0,
    extra_ball_count INT NOT NULL DEFAULT 3,
    
    -- 遊戲進度相關
    current_state_start_time TIMESTAMP NOT NULL,  -- 當前狀態開始時間
    max_timeout INT NOT NULL DEFAULT 60,          -- 當前狀態最大持續時間(秒)
    
    -- 統計相關
    total_cards INT DEFAULT 0,                    -- 總售出彩卡數
    total_players INT DEFAULT 0,                  -- 總參與玩家數
    total_bet_amount DECIMAL(18, 2) DEFAULT 0,    -- 總下注金額
    total_win_amount DECIMAL(18, 2) DEFAULT 0,    -- 總派彩金額
    
    -- 遊戲取消相關
    cancelled BOOLEAN NOT NULL DEFAULT FALSE,
    cancel_time TIMESTAMP NULL,
    cancelled_by VARCHAR(50) NULL,
    cancel_reason VARCHAR(255) NULL,
    
    -- 紀錄與快照
    game_snapshot JSON NULL,                      -- 遊戲結束時的完整狀態快照
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_state (state),
    INDEX idx_start_time (start_time),
    INDEX idx_room_id (room_id)
);

-- 已抽出的球數據表
CREATE TABLE drawn_balls (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    game_id VARCHAR(100) NOT NULL,
    number INT NOT NULL,                     -- 球號
    sequence INT NOT NULL,                   -- 序號，第幾顆球
    ball_type ENUM('REGULAR', 'EXTRA', 'JACKPOT', 'LUCKY') NOT NULL,  -- 球類型
    drawn_time TIMESTAMP NOT NULL,           -- 抽出時間
    side VARCHAR(10) NULL,                   -- 額外球的選邊 (LEFT/RIGHT/NULL)
    is_last_ball BOOLEAN NOT NULL DEFAULT FALSE,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_game_type_sequence (game_id, ball_type, sequence),
    FOREIGN KEY (game_id) REFERENCES games(game_id),
    INDEX idx_game_id (game_id)
);

-- 幸運號碼表
CREATE TABLE lucky_balls (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    game_id VARCHAR(100) NULL,                -- 關聯的遊戲 ID (可能為空)
    draw_date TIMESTAMP NOT NULL,            -- 抽出日期
    number1 INT NOT NULL,
    number2 INT NOT NULL,
    number3 INT NOT NULL,
    number4 INT NOT NULL,
    number5 INT NOT NULL,
    number6 INT NOT NULL,
    number7 INT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,    -- 是否為當前活躍的幸運號碼
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_active (active),
    INDEX idx_game_id (game_id)
);

-- Jackpot 遊戲表
CREATE TABLE jackpot_games (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    game_id VARCHAR(100) NOT NULL,            -- 對應的主遊戲 ID
    jackpot_id VARCHAR(50) NOT NULL UNIQUE,  -- Jackpot 遊戲 ID
    start_time TIMESTAMP NULL,               -- 開始時間
    end_time TIMESTAMP NULL,                 -- 結束時間
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (game_id) REFERENCES games(game_id),
    INDEX idx_jackpot_id (jackpot_id)
);

-- 遊戲階段記錄表
CREATE TABLE game_state_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    game_id VARCHAR(100) NOT NULL,
    state VARCHAR(30) NOT NULL,              -- 狀態名稱
    start_time TIMESTAMP NOT NULL,           -- 開始時間
    end_time TIMESTAMP NULL,                 -- 結束時間
    duration_seconds INT NULL,               -- 持續時間(秒)
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (game_id) REFERENCES games(game_id),
    INDEX idx_game_id_state (game_id, state)
);

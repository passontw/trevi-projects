-- 創建資料庫
CREATE DATABASE IF NOT EXISTS g38_lottery;
USE g38_lottery;

-- 設置字符集和排序規則
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 遊戲記錄表
CREATE TABLE IF NOT EXISTS `games` (
  `id` VARCHAR(36) NOT NULL COMMENT '遊戲唯一標識符',
  `state` VARCHAR(20) NOT NULL COMMENT '遊戲狀態',
  `start_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '遊戲開始時間',
  `end_time` TIMESTAMP NULL DEFAULT NULL COMMENT '遊戲結束時間',
  `has_jackpot` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否有JP遊戲',
  `extra_ball_count` INT NOT NULL DEFAULT 0 COMMENT '額外球數量',
  `state_start_time` TIMESTAMP NULL COMMENT '當前狀態開始時間',
  `max_timeout` INT NOT NULL DEFAULT 60 COMMENT '當前狀態最大超時時間(秒)',
  `lucky_numbers_json` JSON NULL COMMENT '幸運號碼JSON格式',
  `drawn_balls_json` JSON NULL COMMENT '已抽出球JSON格式',
  `extra_balls_json` JSON NULL COMMENT '額外球JSON格式',
  `jackpot_info_json` JSON NULL COMMENT 'Jackpot資訊JSON格式',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新時間',
  PRIMARY KEY (`id`),
  INDEX `idx_state` (`state`),
  INDEX `idx_start_time` (`start_time`),
  INDEX `idx_end_time` (`end_time`),
  INDEX `idx_state_start_time` (`state_start_time`),
  INDEX `idx_game_state` (`state`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '遊戲記錄表';

-- 幸運號碼表
CREATE TABLE IF NOT EXISTS `lucky_numbers` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `game_id` VARCHAR(36) NOT NULL COMMENT '關聯的遊戲ID',
  `number` INT NOT NULL COMMENT '幸運號碼',
  `sequence` INT NOT NULL COMMENT '序號',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_game_id` (`game_id`),
  INDEX `idx_game_id_sequence` (`game_id`, `sequence`),
  CONSTRAINT `fk_lucky_numbers_game_id` FOREIGN KEY (`game_id`) REFERENCES `games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '幸運號碼表';

-- 開獎球表
CREATE TABLE IF NOT EXISTS `drawn_balls` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `game_id` VARCHAR(36) NOT NULL COMMENT '關聯的遊戲ID',
  `number` INT NOT NULL COMMENT '球號',
  `sequence` INT NOT NULL COMMENT '抽出順序',
  `drawn_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '抽出時間',
  `is_extra_ball` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否為額外球',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_game_id` (`game_id`),
  INDEX `idx_sequence` (`sequence`),
  CONSTRAINT `fk_drawn_balls_game_id` FOREIGN KEY (`game_id`) REFERENCES `games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '開獎球表';

-- 額外球表
CREATE TABLE IF NOT EXISTS `extra_balls` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `game_id` VARCHAR(36) NOT NULL COMMENT '關聯的遊戲ID',
  `number` INT NOT NULL COMMENT '球號',
  `side` VARCHAR(10) NOT NULL COMMENT '左側或右側(LEFT或RIGHT)',
  `drawn_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '抽出時間',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_game_id` (`game_id`),
  CONSTRAINT `fk_extra_balls_game_id` FOREIGN KEY (`game_id`) REFERENCES `games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '額外球表';

-- 玩家表
CREATE TABLE IF NOT EXISTS `players` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `user_id` VARCHAR(36) NOT NULL COMMENT '玩家ID',
  `nickname` VARCHAR(50) NULL DEFAULT NULL COMMENT '玩家暱稱',
  `balance` BIGINT NOT NULL DEFAULT 0 COMMENT '玩家餘額',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新時間',
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_user_id` (`user_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '玩家表';

-- 投注記錄表
CREATE TABLE IF NOT EXISTS `bets` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `game_id` VARCHAR(36) NOT NULL COMMENT '關聯的遊戲ID',
  `user_id` VARCHAR(36) NOT NULL COMMENT '玩家ID',
  `bet_amount` INT NOT NULL COMMENT '投注金額',
  `selected_numbers` VARCHAR(300) NOT NULL COMMENT '選擇的號碼，逗號分隔',
  `bet_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '投注時間',
  `win_amount` INT NULL DEFAULT 0 COMMENT '贏取金額',
  `is_extra_bet` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否為額外球投注',
  `extra_side` VARCHAR(10) NULL DEFAULT NULL COMMENT '額外球左側或右側',
  `status` VARCHAR(20) NOT NULL DEFAULT 'PENDING' COMMENT '狀態：PENDING, SETTLED, CANCELED',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新時間',
  PRIMARY KEY (`id`),
  INDEX `idx_game_id` (`game_id`),
  INDEX `idx_user_id` (`user_id`),
  INDEX `idx_status` (`status`),
  CONSTRAINT `fk_bets_game_id` FOREIGN KEY (`game_id`) REFERENCES `games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '投注記錄表';

-- JP遊戲表
CREATE TABLE IF NOT EXISTS `jp_games` (
  `id` VARCHAR(36) NOT NULL COMMENT 'JP遊戲唯一標識符',
  `game_id` VARCHAR(36) NOT NULL COMMENT '關聯的主遊戲ID',
  `start_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '開始時間',
  `end_time` TIMESTAMP NULL DEFAULT NULL COMMENT '結束時間',
  `jackpot_amount` BIGINT NOT NULL DEFAULT 0 COMMENT 'JP金額',
  `active` TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'JP遊戲是否激活',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新時間',
  PRIMARY KEY (`id`),
  INDEX `idx_game_id` (`game_id`),
  CONSTRAINT `fk_jp_games_game_id` FOREIGN KEY (`game_id`) REFERENCES `games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'JP遊戲表';

-- JP球表
CREATE TABLE IF NOT EXISTS `jp_balls` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `jp_game_id` VARCHAR(36) NOT NULL COMMENT '關聯的JP遊戲ID',
  `number` INT NOT NULL COMMENT '球號',
  `sequence` INT NOT NULL COMMENT '抽出順序',
  `drawn_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '抽出時間',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_jp_game_id` (`jp_game_id`),
  INDEX `idx_sequence` (`sequence`),
  CONSTRAINT `fk_jp_balls_jp_game_id` FOREIGN KEY (`jp_game_id`) REFERENCES `jp_games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'JP球表';

-- JP參與記錄表
CREATE TABLE IF NOT EXISTS `jp_participations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `jp_game_id` VARCHAR(36) NOT NULL COMMENT '關聯的JP遊戲ID',
  `user_id` VARCHAR(36) NOT NULL COMMENT '玩家ID',
  `card_id` VARCHAR(50) NOT NULL COMMENT '卡片ID',
  `join_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '參與時間',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_jp_game_id` (`jp_game_id`),
  INDEX `idx_user_id` (`user_id`),
  INDEX `idx_card_id` (`card_id`),
  CONSTRAINT `fk_jp_participations_jp_game_id` FOREIGN KEY (`jp_game_id`) REFERENCES `jp_games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'JP參與記錄表';

-- JP獲勝者表
CREATE TABLE IF NOT EXISTS `jp_winners` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主鍵ID',
  `jp_game_id` VARCHAR(36) NOT NULL COMMENT '關聯的JP遊戲ID',
  `user_id` VARCHAR(36) NOT NULL COMMENT '玩家ID',
  `card_id` VARCHAR(50) NOT NULL COMMENT '卡片ID',
  `win_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '獲勝時間',
  `amount` BIGINT NOT NULL COMMENT '贏取金額',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
  PRIMARY KEY (`id`),
  INDEX `idx_jp_game_id` (`jp_game_id`),
  INDEX `idx_user_id` (`user_id`),
  CONSTRAINT `fk_jp_winners_jp_game_id` FOREIGN KEY (`jp_game_id`) REFERENCES `jp_games` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'JP獲勝者表';

-- 創建自動更新 JSON 欄位的函數
DELIMITER //
CREATE PROCEDURE update_game_json_fields(IN game_id VARCHAR(36))
BEGIN
  -- 更新幸運號碼 JSON
  UPDATE games g
  SET g.lucky_numbers_json = (
    SELECT JSON_ARRAYAGG(number)
    FROM lucky_numbers
    WHERE game_id = g.id
    ORDER BY sequence
  )
  WHERE g.id = game_id;
  
  -- 更新已抽出球 JSON
  UPDATE games g
  SET g.drawn_balls_json = (
    SELECT JSON_ARRAYAGG(
      JSON_OBJECT(
        'number', number,
        'drawnTime', drawn_time,
        'sequence', sequence
      )
    )
    FROM drawn_balls
    WHERE game_id = g.id
    ORDER BY sequence
  )
  WHERE g.id = game_id;
  
  -- 更新額外球 JSON
  UPDATE games g
  SET g.extra_balls_json = (
    SELECT JSON_ARRAYAGG(
      JSON_OBJECT(
        'number', number,
        'drawnTime', drawn_time,
        'sequence', id,
        'side', side
      )
    )
    FROM extra_balls
    WHERE game_id = g.id
  )
  WHERE g.id = game_id;
  
  -- 更新 Jackpot 資訊 JSON
  UPDATE games g
  SET g.jackpot_info_json = (
    SELECT JSON_OBJECT(
      'active', jp.active,
      'gameId', jp.id,
      'amount', jp.jackpot_amount,
      'startTime', jp.start_time,
      'endTime', jp.end_time,
      'drawnBalls', (
        SELECT JSON_ARRAYAGG(
          JSON_OBJECT(
            'number', number,
            'drawnTime', drawn_time,
            'sequence', sequence
          )
        )
        FROM jp_balls
        WHERE jp_game_id = jp.id
        ORDER BY sequence
      ),
      'winner', (
        SELECT JSON_OBJECT(
          'userId', user_id,
          'cardId', card_id,
          'winTime', win_time,
          'amount', amount
        )
        FROM jp_winners
        WHERE jp_game_id = jp.id
        LIMIT 1
      )
    )
    FROM jp_games jp
    WHERE jp.game_id = g.id
    LIMIT 1
  )
  WHERE g.id = game_id;
END //
DELIMITER ;

-- 創建更新 JSON 欄位的觸發器
DELIMITER //
CREATE TRIGGER lucky_numbers_after_insert AFTER INSERT ON lucky_numbers
FOR EACH ROW
BEGIN
  CALL update_game_json_fields(NEW.game_id);
END //

CREATE TRIGGER drawn_balls_after_insert AFTER INSERT ON drawn_balls
FOR EACH ROW
BEGIN
  CALL update_game_json_fields(NEW.game_id);
END //

CREATE TRIGGER extra_balls_after_insert AFTER INSERT ON extra_balls
FOR EACH ROW
BEGIN
  CALL update_game_json_fields(NEW.game_id);
END //

CREATE TRIGGER jp_balls_after_insert AFTER INSERT ON jp_balls
FOR EACH ROW
BEGIN
  SELECT game_id INTO @game_id FROM jp_games WHERE id = NEW.jp_game_id;
  CALL update_game_json_fields(@game_id);
END //

CREATE TRIGGER jp_winners_after_insert AFTER INSERT ON jp_winners
FOR EACH ROW
BEGIN
  SELECT game_id INTO @game_id FROM jp_games WHERE id = NEW.jp_game_id;
  CALL update_game_json_fields(@game_id);
END //

-- 添加時間軸相關欄位到指定操作的觸發器
CREATE TRIGGER game_state_update_trigger BEFORE UPDATE ON games
FOR EACH ROW
BEGIN
  IF NEW.state != OLD.state THEN
    SET NEW.state_start_time = CURRENT_TIMESTAMP;
  END IF;
END //
DELIMITER ;

SET FOREIGN_KEY_CHECKS = 1;
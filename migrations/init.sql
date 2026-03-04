-- B站UP主运营管理平台 数据库初始化脚本
-- 创建数据库
CREATE DATABASE IF NOT EXISTS bili_admin DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE bili_admin;

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `bili_uid` BIGINT NOT NULL COMMENT 'B站UID',
    `bili_name` VARCHAR(100) COMMENT 'B站用户名',
    `bili_face` VARCHAR(500) COMMENT '头像URL',
    `sess_data` VARCHAR(500) COMMENT '登录凭证',
    `bili_jct` VARCHAR(100) COMMENT 'CSRF Token',
    `is_logged_in` TINYINT(1) DEFAULT 0 COMMENT '是否已登录',
    `last_login_at` DATETIME COMMENT '最后登录时间',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME,
    UNIQUE KEY `idx_bili_uid` (`bili_uid`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 评论表
CREATE TABLE IF NOT EXISTS `comments` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `comment_id` BIGINT NOT NULL COMMENT 'B站评论ID',
    `video_bvid` VARCHAR(20) NOT NULL COMMENT '视频BV号',
    `video_aid` BIGINT COMMENT '视频AV号',
    `content` TEXT NOT NULL COMMENT '评论内容',
    `author_id` BIGINT COMMENT '评论者ID',
    `author_name` VARCHAR(100) COMMENT '评论者名称',
    `reply_id` BIGINT DEFAULT 0 COMMENT '回复的评论ID',
    `reply_status` TINYINT DEFAULT 0 COMMENT '回复状态 0=未回复 1=已回复 2=忽略',
    `reply_content` TEXT COMMENT '回复内容',
    `is_ai_reply` TINYINT(1) DEFAULT 0 COMMENT '是否AI回复',
    `comment_time` DATETIME COMMENT '评论时间',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME,
    UNIQUE KEY `idx_comment_id` (`comment_id`),
    KEY `idx_video_bvid` (`video_bvid`),
    KEY `idx_author_id` (`author_id`),
    KEY `idx_reply_status` (`reply_status`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='评论表';

-- 私信表
CREATE TABLE IF NOT EXISTS `messages` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `message_id` BIGINT NOT NULL COMMENT 'B站消息ID',
    `sender_id` BIGINT NOT NULL COMMENT '发送者ID',
    `sender_name` VARCHAR(100) COMMENT '发送者名称',
    `content` TEXT NOT NULL COMMENT '消息内容',
    `reply_status` TINYINT DEFAULT 0 COMMENT '回复状态',
    `reply_content` TEXT COMMENT '回复内容',
    `is_ai_reply` TINYINT(1) DEFAULT 0 COMMENT '是否AI回复',
    `is_read` TINYINT(1) DEFAULT 0 COMMENT '是否已读',
    `message_time` DATETIME COMMENT '消息时间',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME,
    UNIQUE KEY `idx_message_id` (`message_id`),
    KEY `idx_sender_id` (`sender_id`),
    KEY `idx_reply_status` (`reply_status`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='私信表';

-- 互动记录表
CREATE TABLE IF NOT EXISTS `interactions` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `video_bvid` VARCHAR(20) NOT NULL COMMENT '视频BV号',
    `video_title` VARCHAR(500) COMMENT '视频标题',
    `video_owner_id` BIGINT COMMENT '视频作者ID',
    `video_owner` VARCHAR(100) COMMENT '视频作者名',
    `action_type` VARCHAR(20) NOT NULL COMMENT '操作类型 like/coin/favorite/triple',
    `coin_count` INT DEFAULT 0 COMMENT '投币数量',
    `success` TINYINT(1) DEFAULT 1 COMMENT '是否成功',
    `error_message` TEXT COMMENT '错误信息',
    `action_time` DATETIME COMMENT '操作时间',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME,
    KEY `idx_video_bvid` (`video_bvid`),
    KEY `idx_action_type` (`action_type`),
    KEY `idx_video_owner_id` (`video_owner_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='互动记录表';

-- 标签热度表
CREATE TABLE IF NOT EXISTS `tag_rankings` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `tag_name` VARCHAR(100) NOT NULL COMMENT '标签名',
    `tag_id` BIGINT COMMENT '标签ID',
    `hot_value` BIGINT COMMENT '热度值',
    `video_count` INT COMMENT '视频数量',
    `rank` INT COMMENT '排名',
    `category` VARCHAR(50) COMMENT '分类',
    `record_date` DATE COMMENT '记录日期',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME,
    UNIQUE KEY `idx_tag_date` (`tag_name`, `record_date`),
    KEY `idx_record_date` (`record_date`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='标签热度表';

-- 大模型对话日志表
CREATE TABLE IF NOT EXISTS `llm_chat_logs` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `provider` VARCHAR(50) COMMENT '提供者',
    `model` VARCHAR(100) COMMENT '模型',
    `input_type` VARCHAR(20) COMMENT '输入类型 comment/message',
    `input_id` BIGINT COMMENT '关联ID',
    `input_content` TEXT COMMENT '输入内容',
    `output_content` TEXT COMMENT '输出内容',
    `prompt_tokens` INT COMMENT '输入token数',
    `output_tokens` INT COMMENT '输出token数',
    `total_tokens` INT COMMENT '总token数',
    `success` TINYINT(1) DEFAULT 1 COMMENT '是否成功',
    `error_message` TEXT COMMENT '错误信息',
    `duration` BIGINT COMMENT '耗时(毫秒)',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME,
    KEY `idx_provider` (`provider`),
    KEY `idx_input_type` (`input_type`),
    KEY `idx_input_id` (`input_id`),
    KEY `idx_created_at` (`created_at`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='大模型对话日志表';

-- 系统设置表
CREATE TABLE IF NOT EXISTS `settings` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `key` VARCHAR(100) NOT NULL COMMENT '设置键',
    `value` TEXT COMMENT '设置值',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME,
    UNIQUE KEY `idx_key` (`key`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统设置表';

-- 任务队列表
CREATE TABLE IF NOT EXISTS `tasks` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `task_type` VARCHAR(50) NOT NULL COMMENT '任务类型',
    `target_id` BIGINT COMMENT '目标ID',
    `target_data` TEXT COMMENT '目标数据JSON',
    `status` TINYINT DEFAULT 0 COMMENT '状态 0=pending 1=running 2=success 3=failed',
    `result` TEXT COMMENT '执行结果',
    `retry_count` INT DEFAULT 0 COMMENT '重试次数',
    `max_retry` INT DEFAULT 3 COMMENT '最大重试次数',
    `run_at` DATETIME COMMENT '计划执行时间',
    `started_at` DATETIME COMMENT '开始时间',
    `finished_at` DATETIME COMMENT '完成时间',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME,
    KEY `idx_task_type` (`task_type`),
    KEY `idx_target_id` (`target_id`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='任务队列表';

-- 插入默认设置
INSERT INTO `settings` (`key`, `value`) VALUES
('llm_default_provider', 'deepseek'),
('ai_reply_enabled', 'true'),
('ai_reply_prompt', '你是一个B站UP主的助手。请根据用户的评论生成一个友善、有趣的回复。'),
('batch_reply_limit', '10'),
('interaction_delay', '2000')
ON DUPLICATE KEY UPDATE `value` = VALUES(`value`);

-- 大模型配置表
CREATE TABLE IF NOT EXISTS `llm_providers` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(100) NOT NULL COMMENT '配置名称',
    `provider` VARCHAR(50) NOT NULL COMMENT '渠道(openai等)',
    `api_key` VARCHAR(255) COMMENT 'API密钥',
    `base_url` VARCHAR(255) COMMENT '代理URL',
    `model` VARCHAR(100) COMMENT '具体模型名称',
    `max_tokens` INT DEFAULT 1000 COMMENT '最大回复长度',
    `temperature` DECIMAL(5,2) DEFAULT 0.70 COMMENT '温度',
    `enabled` TINYINT(1) DEFAULT 1 COMMENT '是否启用',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME,
    UNIQUE KEY `idx_name` (`name`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='大模型配置表';

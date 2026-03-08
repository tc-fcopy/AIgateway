-- =============================================
-- AI网关数据库表设计
-- 创建日期：2026-03-07
-- =============================================

-- AI Consumer 表
-- 用于存储AI消费者的认证信息
CREATE TABLE IF NOT EXISTS `ai_consumer` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `consumer_name` varchar(255) NOT NULL COMMENT 'Consumer名称（唯一）',
    `credential` varchar(500) NOT NULL COMMENT '访问凭证（API Key / JWT 标识）',
    `consumer_type` varchar(20) NOT NULL DEFAULT 'key' COMMENT '类型：key / jwt',
    `status` tinyint(1) NOT NULL DEFAULT 1 COMMENT '状态：0-禁用 1-启用',
    `create_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_consumer_name` (`consumer_name`),
    KEY `idx_credential` (`credential`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI消费者表';

-- AI Service Config 表
-- 用于存储服务的AI功能配置
CREATE TABLE IF NOT EXISTS `ai_service_config` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `service_id` bigint(20) NOT NULL COMMENT '服务ID（关联gateway_service_info.id）',
    `enable_key_auth` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否开启Key Auth',
    `enable_jwt_auth` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否开启JWT Auth',
    `enable_token_ratelimit` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否开启Token限流',
    `enable_quota` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否开启配额',
    `enable_model_router` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否开启模型路由',
    `enable_model_mapper` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否开启模型映射',
    `enable_cache` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否开启AI缓存',
    `enable_loadbalancer` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否开启AI负载均衡',
    `enable_observability` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否开启可观测性',
    `enable_prompt_decorator` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否开启Prompt装饰',
    `enable_ip_restriction` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否开启IP限制',
    `enable_cors` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否开启CORS',
    `create_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_service_id` (`service_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI服务配置表';

-- AI Quota 表
-- 用于存储消费者的配额信息
CREATE TABLE IF NOT EXISTS `ai_quota` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `consumer_id` bigint(20) NOT NULL COMMENT 'Consumer ID（关联ai_consumer.id）',
    `quota_total` bigint(20) NOT NULL DEFAULT 100000 COMMENT '配额总额（tokens）',
    `quota_used` bigint(20) NOT NULL DEFAULT 0 COMMENT '已使用配额（tokens）',
    `quota_remaining` bigint(20) NOT NULL DEFAULT 100000 COMMENT '剩余配额（tokens）',
    `reset_cycle` varchar(20) NOT NULL DEFAULT 'month' COMMENT '重置周期：day/week/month/year/never',
    `last_reset_time` datetime NULL COMMENT '上次重置时间',
    `create_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_consumer_id` (`consumer_id`),
    KEY `idx_reset_cycle` (`reset_cycle`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI配额表';

-- AI Model Config 表
-- 用于存储模型路由和映射配置
CREATE TABLE IF NOT EXISTS `ai_model_config` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `service_id` bigint(20) NOT NULL COMMENT '服务ID',
    `config_type` varchar(20) NOT NULL COMMENT '配置类型：router/mapper',
    `source_pattern` varchar(255) NOT NULL COMMENT '源模式（正则或模型名）',
    `target_model` varchar(255) NOT NULL COMMENT '目标模型',
    `priority` int(11) NOT NULL DEFAULT 0 COMMENT '优先级（数字越大优先级越高）',
    `enable` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    `create_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_service_type` (`service_id`, `config_type`),
    KEY `idx_priority` (`priority`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI模型配置表';

-- AI Cache Config 表
-- 用于存储缓存配置
CREATE TABLE IF NOT EXISTS `ai_cache_config` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `service_id` bigint(20) NOT NULL COMMENT '服务ID',
    `cache_ttl` int(11) NOT NULL DEFAULT 3600 COMMENT '缓存TTL（秒）',
    `max_cache_size` int(11) NOT NULL DEFAULT 10000 COMMENT '最大缓存条目数',
    `cache_stream` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否缓存流式响应',
    `enable` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    `create_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_service_id` (`service_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI缓存配置表';

-- 初始化示例数据
-- =============================================

-- 插入示例Consumer
INSERT INTO `ai_consumer` (`consumer_name`, `credential`, `consumer_type`, `status`) VALUES
('default_consumer', 'sk-test-default-api-key-2026', 'key', 1)
ON DUPLICATE KEY UPDATE `status` = VALUES(`status`);

-- 插入示例配额
INSERT INTO `ai_quota` (`consumer_id`, `quota_total`, `quota_remaining`, `reset_cycle`) VALUES
(1, 100000, 100000, 'month')
ON DUPLICATE KEY UPDATE `quota_total` = VALUES(`quota_total`);

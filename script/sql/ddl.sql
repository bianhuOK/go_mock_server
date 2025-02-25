-- 规则主表
CREATE TABLE `mock_rules` (
    `id` VARCHAR(36) NOT NULL COMMENT '规则ID',
    `name` VARCHAR(50) NOT NULL COMMENT '规则名称',
    `protocol` VARCHAR(20) NOT NULL COMMENT '协议类型',
    `match_config` JSON NOT NULL COMMENT '匹配配置',
    `action_config` JSON NOT NULL COMMENT '响应动作配置',
    `priority` INT DEFAULT 0 COMMENT '匹配优先级',
    `status` VARCHAR(20) NOT NULL COMMENT '规则状态',
    `version` INT DEFAULT 1 COMMENT '版本号',
    `created_at` INT NOT NULL COMMENT '创建时间',
    `updated_at` INT NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`id`),
    INDEX `idx_protocol` (`protocol`),
    INDEX `idx_status` (`status`),
    INDEX `idx_priority` (`priority`),
    INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Mock规则表';

-- 规则标签关联表
CREATE TABLE `mock_rule_tags` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `rule_id` VARCHAR(36) NOT NULL COMMENT '规则ID',
    `tag_id` INT NOT NULL COMMENT '标签ID',
    `created_at` INT NOT NULL COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_rule_tag` (`rule_id`, `tag_id`),
    INDEX `idx_tag_id` (`tag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='规则标签关联表';

-- 标签表
CREATE TABLE `tags` (
    `id` INT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(50) NOT NULL COMMENT '标签名称',
    `created_at` INT NOT NULL COMMENT '创建时间',
    `updated_at` INT NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='标签表';

-- 规则变更历史表
CREATE TABLE `mock_rule_histories` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `rule_id` VARCHAR(36) NOT NULL COMMENT '规则ID',
    `version` INT NOT NULL COMMENT '版本号',
    `change_type` VARCHAR(20) NOT NULL COMMENT '变更类型：create/update/delete',
    `content` JSON NOT NULL COMMENT '规则完整内容',
    `created_by` INT NOT NULL COMMENT '操作人ID',
    `created_at` INT NOT NULL COMMENT '创建时间',
    PRIMARY KEY (`id`),
    INDEX `idx_rule_id` (`rule_id`),
    INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='规则变更历史表';
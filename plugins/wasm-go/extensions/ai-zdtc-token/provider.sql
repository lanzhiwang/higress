/*
 Navicat Premium Dump SQL

 Source Server         : 正式环境
 Source Server Type    : MySQL
 Source Server Version : 80033 (8.0.33)
 Source Host           : 10.100.100.103:3306
 Source Schema         : model

 Target Server Type    : MySQL
 Target Server Version : 80033 (8.0.33)
 File Encoding         : 65001

 Date: 28/09/2026 14:42:14
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for provider
-- ----------------------------
DROP TABLE IF EXISTS `provider`;
CREATE TABLE `provider` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `provider_name` varchar(100) COLLATE utf8mb4_general_ci NOT NULL COMMENT '供应商账号名称',
  `provider_type` varchar(50) COLLATE utf8mb4_general_ci NOT NULL COMMENT '供应商类型: baidu, deepseek, aliyun, minimax, zhipu, moonshot',
  `protocol` varchar(50) COLLATE utf8mb4_general_ci NOT NULL COMMENT '连接协议: baidu-qianfan, deepseek-preset, openai-compatible',
  `base_url` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'API Base URL',
  `credentials` text COLLATE utf8mb4_general_ci COMMENT 'API凭证',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '账号状态: 0-停用 1-启用 2-已删除',
  `description` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '账号备注说明',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `connection_status` tinyint NOT NULL DEFAULT '1' COMMENT '连接状态: 0-未知 1-可用 2-不可用',
  `connection_status_time` datetime DEFAULT NULL COMMENT '最近连接检查时间',
  `sync_last_time` datetime DEFAULT NULL COMMENT '最近同步时间',
  `cost_mode` varchar(20) COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'post-billing' COMMENT '成本模式: post-billing(后计价), standard-price(标准单价)',
  `quota_total` decimal(15,2) DEFAULT NULL COMMENT '采购总额度(元)',
  `quota_used` decimal(15,2) DEFAULT '0.00' COMMENT '已使用额度(元)',
  `currency` varchar(10) COLLATE utf8mb4_general_ci DEFAULT 'CNY' COMMENT '币种',
  `alert_threshold` int DEFAULT '80' COMMENT '预警阈值百分比，如80表示用量达到80%%时预警',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_provider_name` (`provider_name`),
  KEY `idx_provider_type` (`provider_type`),
  KEY `idx_connection_status` (`connection_status`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=71 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='供应商账号';

SET FOREIGN_KEY_CHECKS = 1;

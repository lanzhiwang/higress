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

 Date: 21/09/2026 15:45:42
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for token_audit_logs
-- ----------------------------
DROP TABLE IF EXISTS `token_audit_logs`;
CREATE TABLE `token_audit_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `uuid` varchar(36) COLLATE utf8mb4_general_ci NOT NULL COMMENT '唯一标识UUID',
  `request_id` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '请求ID',
  `llm_model` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'LLM模型名称',
  `llm_model_final` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '最终LLM模型名称',
  `original_auth` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '原始认证信息',
  `authorization` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '认证信息',
  `mse_consumer` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'MSE消费者信息',
  `response_id` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '响应ID',
  `status_code` int DEFAULT NULL COMMENT '服务端响应HTTP状态码',
  `error_message` text COLLATE utf8mb4_general_ci COMMENT '服务端返回的错误信息',
  `start_time_milli` bigint DEFAULT NULL COMMENT '开始时间(毫秒)',
  `end_time_milli` bigint DEFAULT NULL COMMENT '结束时间(毫秒)',
  `duration_ms` bigint DEFAULT NULL COMMENT '耗时(毫秒)',
  `input_token` bigint NOT NULL COMMENT '输入Token数',
  `output_token` bigint NOT NULL COMMENT '输出Token数',
  `cached_token` bigint NOT NULL DEFAULT '0' COMMENT '缓存Token数',
  `total_token` bigint NOT NULL COMMENT '总Token数',
  `model` varchar(128) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '模型',
  `provider_id` bigint DEFAULT NULL COMMENT '调用发生时实际供应商ID',
  `tenant_from` tinyint DEFAULT NULL COMMENT '用户来源 1 全部租户可见, 2 不分租户可见, 3 紫东太初, 4 后台新增, 5 百度注册, 6 公众号注册, 7 自然注册',
  `is_billed` tinyint NOT NULL DEFAULT '0' COMMENT '是否计费: 0-未计费, 1-已计费, 2-价格配置缺失待重试, 3-格式错误不予计费',
  `unit_price_input` decimal(10,6) DEFAULT NULL COMMENT '输入单价(元/千Token)',
  `unit_price_output` decimal(10,6) DEFAULT NULL COMMENT '输出单价(元/千Token)',
  `unit_price_cache` decimal(10,6) DEFAULT NULL COMMENT '输入单价(元/百万Token)',
  `cost_input` decimal(12,6) DEFAULT NULL COMMENT '计算后的输入价格(元)',
  `cost_output` decimal(12,6) DEFAULT NULL COMMENT '计算后的输出价格(元)',
  `cost_cache` decimal(12,6) DEFAULT NULL COMMENT '计算后的缓存价格(元)',
  `cost_total` decimal(12,6) DEFAULT NULL COMMENT '总价格(元)',
  `endpoint_id` bigint DEFAULT NULL COMMENT '服务端点ID',
  `provider_catalog_id` bigint DEFAULT NULL COMMENT '供应商模型ID',
  `provider_price_version_id` bigint DEFAULT NULL COMMENT '供应商价格版本ID',
  `provider_cost_status` tinyint NOT NULL DEFAULT '0' COMMENT '0-待计算 1-成功 2-模型映射缺失 3-价格版本缺失 4-Token异常 5-provider_id缺失',
  `provider_unit_price_input` decimal(12,6) DEFAULT NULL COMMENT '未缓存输入采购单价,元/百万Token',
  `provider_unit_price_cache_input` decimal(12,6) DEFAULT NULL COMMENT '缓存输入采购单价,元/百万Token',
  `provider_unit_price_output` decimal(12,6) DEFAULT NULL COMMENT '输出采购单价,元/百万Token',
  `provider_cost_input` decimal(12,6) DEFAULT NULL COMMENT '未缓存输入成本,元',
  `provider_cost_cache_input` decimal(12,6) DEFAULT NULL COMMENT '缓存输入成本,元',
  `provider_cost_output` decimal(12,6) DEFAULT NULL COMMENT '输出成本,元',
  `provider_cost_total` decimal(12,6) DEFAULT NULL COMMENT '供应商总成本,元',
  `provider_cost_calculated_at` datetime(3) DEFAULT NULL COMMENT '供应商成本计算时间',
  `provider_cost_error` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '供应商成本计算失败原因',
  `user_id` bigint DEFAULT NULL COMMENT '用户ID',
  `tenant_id` bigint DEFAULT NULL COMMENT '租户ID',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '修改时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_uuid` (`uuid`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_is_billed` (`is_billed`),
  KEY `idx_user_created` (`user_id`,`created_at`),
  KEY `idx_created_user` (`created_at`,`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=151461 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Token审计日志表';

SET FOREIGN_KEY_CHECKS = 1;

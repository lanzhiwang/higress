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

 Date: 29/09/2026 13:44:13
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for service_endpoint
-- ----------------------------
DROP TABLE IF EXISTS `service_endpoint`;
CREATE TABLE `service_endpoint` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `endpoint_name` varchar(100) COLLATE utf8mb4_general_ci NOT NULL COMMENT '端点名称',
  `endpoint_code` varchar(100) COLLATE utf8mb4_general_ci NOT NULL COMMENT '端点编码(唯一标识)',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态: 0-停用 1-启用 2-已删除',
  `endpoint_status` varchar(20) COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'unknown' COMMENT '端点状态: available(可用), unavailable(不可用), unknown(未知)',
  `endpoint_status_time` datetime DEFAULT NULL COMMENT '最近端点状态检查时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `source_type` varchar(20) COLLATE utf8mb4_general_ci NOT NULL COMMENT '来源类型: local(本地部署), cloud-api(公有云API)',
  `provider_id` bigint DEFAULT NULL COMMENT '供应商ID',
  `provider_model_id` bigint DEFAULT NULL COMMENT '供应商模型ID',
  `provider_model_name` varchar(200) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '供应商模型名称',
  `endpoint_version` varchar(50) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '端点版本',
  `api_endpoint` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'API端点地址',
  `api_model_code` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'API调用的模型编码',
  `qps_limit` int DEFAULT NULL COMMENT 'QPS上限',
  `concurrency_limit` int DEFAULT NULL COMMENT '并发上限',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_endpoint_code` (`endpoint_code`),
  KEY `idx_source_type` (`source_type`),
  KEY `idx_provider_id` (`provider_id`),
  KEY `idx_provider_model_id` (`provider_model_id`),
  KEY `idx_endpoint_status` (`endpoint_status`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=79 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='服务端点表';

SET FOREIGN_KEY_CHECKS = 1;

#!/usr/bin/env bash
set -eux

REGISTRY="higress-registry.cn-hangzhou.cr.aliyuncs.com"
REPO="higress/all-in-one"

# 1. 访问 v2 基础接口, 获取 401 响应头中的认证服务器 URL (realm) 和服务名称 (service)
WWW_AUTH=$(curl -sI "https://${REGISTRY}/v2/" | grep -i "www-authenticate")

REALM=$(echo "$WWW_AUTH" | grep -oE 'realm="[^"]+"' | cut -d'"' -f2)
SERVICE=$(echo "$WWW_AUTH" | grep -oE 'service="[^"]+"' | cut -d'"' -f2)

# 2. 匿名请求临时 Bearer Token
TOKEN=$(curl -s "${REALM}?service=${SERVICE}&scope=repository:${REPO}:pull" | jq -r '.token')

# 3. 携带 Token 请求 Tag 列表并使用 jq 格式化输出
curl -s -H "Authorization: Bearer ${TOKEN}" "https://${REGISTRY}/v2/${REPO}/tags/list" | jq -r '.tags[]'

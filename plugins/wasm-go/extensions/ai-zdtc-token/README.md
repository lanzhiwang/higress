# AI Token 审计记录插件 (ai-zdtc-token)

## 1. 插件简介

`ai-zdtc-token` 是一款专为 Higress 网关对接大模型服务(如 vLLM、OpenAI、Claude 等)设计的自定义审计插件.

该插件的主要功能包括:

1. 身份与元数据追踪: 在请求头阶段提取用户与追踪信息(`x-request-id`、`x-higress-llm-model`、`x-higress-llm-model-final`、`x-hi-original-auth`、`x-mse-consumer`), 以及响应头阶段的响应标识(`x-request-id`), 并将它们暂存入上下文(Context)中.

2. 端到端耗时统计: 自动计算从请求进入网关到模型流式响应完全结束的毫秒级端到端耗时.

3. Token 使用率统计: 在流的最后一帧(`isLastChunk == true`), 利用内置的 `tokenusage` 机制自动提取模型推理所产生的 `input_token`、`output_token` 和 `total_token`.

4. 异步 Redis 写入: 以防并发阻塞的异步非阻塞方式(Asynchronous Callback), 将统计结果序列化为标准的 JSON 字符串, 并将其写入指定的 Redis 中.

---

## 2. 工作原理与数据链路

```text
  Client           Higress Gateway             vLLM Upstream
    |                     |                         |
    |----(1) Request----->|                         |  [onHttpRequestHeaders] 阶段:
    |    - Headers        |                         |  获取并保存: request_id, llm_model,
    |    - Body (Stream)  |                         |  mse_consumer, startTimeMilli.
    |                     |-----(2) Forward-------->|
    |                     |<----(3) SSE Response----|  [onHttpStreamResponseBody] 阶段:
    |                     |                         |  - 实时透传 Chunk.
    |                     |                         |  - 自动在各帧累积或解析 Token Usage.
    |                     |                         |  [isLastChunk == true] (流结束):
    |                     |                         |  - 计算推理耗时 (DurationMs).
    |                     |                         |  - 组合所有审计项, 转成 JSON 字符串.
    |<---(4) Last Chunk---|                         |  - 异步向 Redis 发起 SET 指令.
    |                     |===(5) Async Redis SET==>|  (Key: ai-zdtc-token|consumer|reqID|respID)
```

---

## 3. 配置说明 (Configuration Schema)

插件提供了标准的 `redis` 配置项:

| 参数名               | 类型    | 是否必填 | 默认值 | 描述 |
| :------------------- | :------ | :------- | :----- | :--- |
| `redis.service_name` | string  | 是       | -      | Redis 服务在网关中发现的 FQDN 域名或静态注册名(例如 `redis-service.default.svc.cluster.local`) |
| `redis.service_port` | integer | 否       | `6379` | Redis 连接端口. 注意: 如果是静态服务(.static), 须将其填写为 80! |
| `redis.username`     | string  | 否       | -      | Redis 账号(无认证可置空) |
| `redis.password`     | string  | 否       | -      | Redis 密码(无认证可置空) |
| `redis.timeout`      | integer | 否       | `1000` | 连接与读写超时时间(单位: 毫秒) |
| `redis.database`     | integer | 否       | `0`    | 指定操作的 Redis DB 数据库 |

---

## 4. WasmPlugin 配置示例 (WasmPlugin Custom Resource)

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  annotations:
    higress.io/wasm-plugin-title: ai-zdtc-token
  labels:
    higress.io/resource-definer: higress
    higress.io/wasm-plugin-built-in: "false"
    higress.io/wasm-plugin-category: custom
    higress.io/wasm-plugin-name: ai-zdtc-token
    higress.io/wasm-plugin-version: 1.0.0
  name: ai-zdtc-token-1.0.0
  namespace: higress-system
spec:
  defaultConfig:
    redis:
      service_name: redis.dns
      service_port: 6379
      timeout: 2000
  defaultConfigDisable: false
  failStrategy: FAIL_OPEN
  imagePullPolicy: IfNotPresent
  imagePullSecret: aliyun-registry-key
  matchRules: []
  phase: UNSPECIFIED_PHASE
  priority: 95
  url: oci://tck-xinan-registry.cn-chengdu.cr.aliyuncs.com/taichu-studio/higress-plugin:ai-zdtc-token-20260709-111511-35df208a
```

---

## 5. Redis 存储数据结构说明

### 5.1 Redis Key 规则

Redis 写入时采用格式化命名空间隔离.

* 格式: `pluginName|mseConsumer|requestID|responseID`
* 示例: `ai-zdtc-token|1-13833583|10c3b60c-0598-4c09-bc85-49bbb876dc8c|6972de9fe61f85f7888ac9a35445f6d5`

### 5.2 Redis Value JSON Schema 示例

存储的 Value 结构中集成了请求上下文中的用户信息、模型名称、模型最终产出、Token 使用总量以及端到端延迟开销:

```json
{
  "request_id": "10c3b60c-0598-4c09-bc85-49bbb876dc8c",
  "llm_model": "claude",
  "llm_model_final": "claude-opus-4-7",
  "original_auth": "Bearer a0s4asfkits7wgvs6jhr5vc5",
  "mse_consumer": "1-13833583",
  "response_id": "6972de9fe61f85f7888ac9a35445f6d5",
  "start_time_milli": 1783567458952,
  "end_time_milli": 1783567460981,
  "duration_ms": 2029,
  "input_token": 3,
  "output_token": 17,
  "total_token": 20,
  "model": "claude-opus-4-7"
}
```

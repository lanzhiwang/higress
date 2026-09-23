package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/tokenusage"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	pluginName = "ai-zdtc-token"

	ctxKeyRequestUUID      = "request_uuid"
	ctxKeyRequestStartTime = "request_start_time"

	// 请求头信息
	headerXHigressLLMModel        = "x-higress-llm-model"
	headerXHigressLLMModelFin     = "x-higress-llm-model-final"
	headerAuthorization           = "authorization"
	headerXHiOriginalAuth         = "x-hi-original-auth"
	headerMseConsumer             = "x-mse-cuser"
	headerMseApi                  = "x-mse-consumer"
	headerXRequestID              = "x-request-id"
	headerEnvoyDecoratorOperation = "x-envoy-decorator-operation"

	// 响应头信息
	headerXResponseID = "x-response-id"

	ctxKeySkipPlugin = "skip_plugin"

	// 用户来源渠道/租户来源 与 供应商ID
	ctxKeyTenantFrom = "tenant_from"
	ctxKeyProviderID = "provider_id"

	// 用于在请求生命周期中暂存缓存 Token、状态码及异常错误信息
	ctxKeyCachedToken    = "cached_token"
	ctxKeyResponseStatus = "response_status"
	ctxKeyErrorMessage   = "error_message"

	// 用于流式传输标识与完整性检查
	ctxKeyIsStream        = "is_stream"          // 标识当前请求是否为流式(SSE)请求
	ctxKeyHasSeenDone     = "has_seen_done"      // 标识流式请求是否已正常收到 [DONE] 结束标志
	ctxKeyIsLastChunkSeen = "is_last_chunk_seen" // 标识流式请求是否已接收到最后一个数据块(isLastChunk=true)
	ctxKeyChunkCount      = "chunk_count"        // 已接收的数据块计数
	ctxKeyTotalChunkBytes = "total_chunk_bytes"  // 已接收的数据块累计总字节数
)

type PluginConfig struct {
	Debug       bool      `yaml:"debug" json:"debug"`
	RedisInfo   RedisInfo `yaml:"redis" json:"redis"`
	redisClient wrapper.RedisClient
	PayInfo     PayInfo `yaml:"pay" json:"pay"`
	payClient   wrapper.HttpClient
	probeOnce   *sync.Once
}

type RedisInfo struct {
	ServiceName string `required:"true" yaml:"service_name" json:"service_name"`
	ServicePort int    `required:"false" yaml:"service_port" json:"service_port"`
	Username    string `required:"false" yaml:"username" json:"username"`
	Password    string `required:"false" yaml:"password" json:"password"`
	Timeout     int    `required:"false" yaml:"timeout" json:"timeout"`
	Database    int    `required:"false" yaml:"database" json:"database"`
}

type PayInfo struct {
	ServiceName     string `required:"true" yaml:"service_name" json:"service_name"`
	ServicePort     int    `required:"false" yaml:"service_port" json:"service_port"`
	Host            string `required:"false" yaml:"host" json:"host"`
	Timeout         int    `required:"false" yaml:"timeout" json:"timeout"`
	BalanceCacheTTL int    `required:"false" yaml:"balance_cache_ttl" json:"balance_cache_ttl"` // 余额数据在 Redis 中的缓存过期时间(秒)
}

type TokenAuditLog struct {
	UUID           string `json:"uuid"`
	RequestID      string `json:"request_id"`
	LLMModel       string `json:"llm_model"`
	LLMModelFinal  string `json:"llm_model_final"`
	OriginalAuth   string `json:"original_auth"`
	Authorization  string `json:"authorization"`
	MseConsumer    string `json:"mse_consumer"`
	TenantFrom     int64  `json:"tenant_from"`
	ProviderID     int64  `json:"provider_id"`
	ResponseID     string `json:"response_id"`
	StartTimeMilli int64  `json:"start_time_milli"`
	EndTimeMilli   int64  `json:"end_time_milli"`
	DurationMs     int64  `json:"duration_ms"`
	InputToken     int64  `json:"input_token"`
	OutputToken    int64  `json:"output_token"`
	TotalToken     int64  `json:"total_token"`
	// 记录上游返回的缓存命中 Token 数量 (对应 usage.prompt_tokens_details.cached_tokens)
	CachedToken int64  `json:"cached_token"`
	Model       string `json:"model"`
	// 记录 HTTP 响应状态码, 用于分析客户端 4xx 参数错误或上游 504 超时等情况, 保证任何情况下(包括 200)都始终输出 status_code 字段
	StatusCode int `json:"status_code"`
	// 记录异常报错信息(提取自响应 body 中的 error 结构、截断提示或网关错误说明), 正常请求为 "", 异常请求为错误信息, 始终保证 JSON Schema 结构对齐
	ErrorMessage string `json:"error_message"`
}

func main() {}

func init() {
	wrapper.SetCtx(
		pluginName,
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamResponseBody),
		// [新增]注册请求生命周期终结回调(无论请求正常完成、上游异常崩溃/重置、超时还是客户端主动断开, Envoy 必将回调此处)
		wrapper.ProcessStreamDoneBy(onHttpStreamDone),
	)
}

// 1. 解析配置阶段
func parseConfig(json gjson.Result, config *PluginConfig, log log.Log) error {
	log.Infof("[ai-zdtc-token parseConfig] === [ParseConfig] 阶段开始 ===")
	log.Infof("[ai-zdtc-token parseConfig] 收到原始 JSON 配置: %s", json.Raw)

	debugResult := json.Get("debug")
	if debugResult.Exists() {
		config.Debug = debugResult.Bool()
	} else {
		config.Debug = true
	}
	log.Infof("[ai-zdtc-token parseConfig] 解析后的 Debug: %t", config.Debug)

	// 解析 redis 服务配置
	config.RedisInfo.ServiceName = json.Get("redis.service_name").String()
	config.RedisInfo.ServicePort = int(json.Get("redis.service_port").Int())
	if config.RedisInfo.ServicePort == 0 {
		config.RedisInfo.ServicePort = 6379 // 默认 Redis 端口
	}
	config.RedisInfo.Username = json.Get("redis.username").String()
	config.RedisInfo.Password = json.Get("redis.password").String()
	config.RedisInfo.Timeout = int(json.Get("redis.timeout").Int())
	if config.RedisInfo.Timeout == 0 {
		config.RedisInfo.Timeout = 1000 // 默认超时时间 1000ms
	}
	config.RedisInfo.Database = int(json.Get("redis.database").Int())

	log.Infof("[ai-zdtc-token parseConfig] 解析后的 Redis 配置 -> 服务名: %s, 端口: %d, 数据库: %d, 超时: %dms",
		config.RedisInfo.ServiceName,
		config.RedisInfo.ServicePort,
		config.RedisInfo.Database,
		config.RedisInfo.Timeout,
	)

	config.redisClient = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
		FQDN: config.RedisInfo.ServiceName,
		Port: int64(config.RedisInfo.ServicePort),
	})

	err := config.redisClient.Init(
		config.RedisInfo.Username,
		config.RedisInfo.Password,
		int64(config.RedisInfo.Timeout),
		wrapper.WithDataBase(config.RedisInfo.Database),
	)
	if err != nil {
		log.Errorf("[ai-zdtc-token parseConfig] Redis 客户端初始化失败: %#v", err)
		return err
	}

	log.Infof("[ai-zdtc-token parseConfig] Redis 客户端成功注册且完成初始化")

	// 解析 pay 服务配置
	payResult := json.Get("pay")
	if payResult.Exists() {
		if payResult.IsObject() {
			config.PayInfo.ServiceName = payResult.Get("service_name").String()
			config.PayInfo.ServicePort = int(payResult.Get("service_port").Int())
			config.PayInfo.Host = payResult.Get("host").String()
			config.PayInfo.Timeout = int(payResult.Get("timeout").Int())
			config.PayInfo.BalanceCacheTTL = int(payResult.Get("balance_cache_ttl").Int())
		} else if payResult.Type == gjson.String {
			payStr := payResult.String()
			if parts := strings.Split(payStr, ":"); len(parts) == 2 {
				config.PayInfo.ServiceName = parts[0]
				port, _ := strconv.Atoi(parts[1])
				config.PayInfo.ServicePort = port
			} else {
				config.PayInfo.ServiceName = payStr
			}
		}
	}

	if config.PayInfo.ServicePort == 0 {
		config.PayInfo.ServicePort = 80
	}
	if config.PayInfo.Timeout == 0 {
		config.PayInfo.Timeout = 2000 // 默认超时 2000ms
	}
	// 如果配置中未指定 balance_cache_ttl 或指定值小于等于 0, 则默认缓存 120 秒 (2分钟)
	if config.PayInfo.BalanceCacheTTL <= 0 {
		config.PayInfo.BalanceCacheTTL = 120
	}

	if config.PayInfo.ServiceName != "" {
		config.payClient = wrapper.NewClusterClient(wrapper.FQDNCluster{
			FQDN: config.PayInfo.ServiceName,
			Port: int64(config.PayInfo.ServicePort),
			Host: config.PayInfo.Host,
		})
		log.Infof("[ai-zdtc-token parseConfig] Pay 服务客户端成功初始化 -> 服务名: %s, 端口: %d, Host: %s, 超时: %dms, 余额缓存TTL: %ds",
			config.PayInfo.ServiceName,
			config.PayInfo.ServicePort,
			config.PayInfo.Host,
			config.PayInfo.Timeout,
			config.PayInfo.BalanceCacheTTL,
		)
	} else {
		log.Warnf("[ai-zdtc-token parseConfig] 警告: 未配置 pay 服务, 余额校验功能将无法正常调用远程接口")
	}

	// 初始化 probeOnce 指针
	config.probeOnce = new(sync.Once)

	log.Infof("[ai-zdtc-token parseConfig] === [ParseConfig] 阶段结束 ===")
	return nil
}

// 2. 请求头处理阶段
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {
	log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段开始 ===")

	path, _ := proxywasm.GetHttpRequestHeader(":path")
	method, _ := proxywasm.GetHttpRequestHeader(":method")
	log.Infof("[ai-zdtc-token onHttpRequestHeaders] 当前请求 - 方法: %s, 路径: %s", method, path)

	// 记录请求 ID 与请求起始时间戳
	reqID, _ := proxywasm.GetHttpRequestHeader(headerXRequestID)
	ctx.SetContext(ctxKeyRequestUUID, reqID)
	ctx.SetContext(ctxKeyRequestStartTime, time.Now().UnixMilli())

	// 检查客户端 Accept 头, 若声明 text/event-stream 则初步标记为流式候选
	acceptHeader, _ := proxywasm.GetHttpRequestHeader("accept")
	if strings.Contains(acceptHeader, "text/event-stream") {
		ctx.SetContext(ctxKeyIsStream, true)
		log.Infof("[ai-zdtc-token onHttpRequestHeaders] 客户端 Accept 声明包含 text/event-stream, 预设流式模式")
	}

	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Errorf("[ai-zdtc-token onHttpRequestHeaders] 无法获取请求头: %#v", err)
	} else {
		log.Infof("[ai-zdtc-token onHttpRequestHeaders] 请求头总数: %d", len(headers))
		for _, h := range headers {
			key := h[0]
			val := h[1]
			if config.Debug {
				log.Infof("[ai-zdtc-token onHttpRequestHeaders]   Header -> %s: %s", key, val)
			}
			// Header -> :path: /v1/chat/completions
			// Header -> :scheme: http
			// Header -> :authority: api.gpugeek.com
			// Header -> :method: POST
			// Header -> user-agent: Go-http-client/1.1
			// Header -> authorization: Bearer w0yqnzgpwosc5601000dk0nqp225kav6x0q98h0h
			// Header -> x-request-id: 001e6c2b-ab43-40a6-b68d-1b33db1b81a6
			// Header -> x-envoy-original-host: higress-gateway.higress-system.svc.cluster.local
			// Header -> x-envoy-decorator-operation: llm-provider-1.internal.dns:443/*
			// Header -> x-higress-original-model: capital
			// Header -> x-higress-llm-model-final: Vendor3/DeepSeek-V4-Flash
			// Header -> x-forwarded-proto: http
			// Header -> x-mse-consumer: 1-87894137
			// Header -> x-hi-original-auth: Bearer ryvsk3zz73419gkgubrnvufp
			// Header -> content-type: application/json
			// Header -> x-forwarded-for: 10.42.1.55
			// Header -> x-envoy-external-address: 10.42.1.55
			// Header -> x-mse-cuser: 1-1
			// Header -> x-higress-llm-model: capital
			// Header -> x-envoy-original-path: /v1/chat/completions
			// Header -> accept: text/event-stream
		}
	}
	log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段结束 ===")
	return types.ActionContinue
}

// 3. 请求体处理阶段
func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log log.Log) types.Action {
	log.Infof("[ai-zdtc-token onHttpRequestBody] === [OnHttpRequestBody] 阶段开始 ===")

	log.Infof("[ai-zdtc-token onHttpRequestBody] 请求体大小: %d 字节", len(body))

	if len(body) > 0 {
		// 解析请求体中的 stream 字段, 精确确认客户端是否要求流式响应
		streamVal := gjson.GetBytes(body, "stream")
		if streamVal.Exists() && streamVal.Bool() {
			ctx.SetContext(ctxKeyIsStream, true)
			log.Infof("[ai-zdtc-token onHttpRequestBody] 解析请求体 JSON stream=true, 确认当前为流式(SSE)传输请求")
		}

		if config.Debug {
			log.Infof("[ai-zdtc-token onHttpRequestBody] 请求体内容: %s", string(body))
		}
	} else {
		if config.Debug {
			log.Infof("[ai-zdtc-token onHttpRequestBody] 请求体为空")
		}
	}

	log.Infof("[ai-zdtc-token onHttpRequestBody] === [OnHttpRequestBody] 阶段结束 ===")
	return types.ActionContinue
}

// 4. 响应头处理阶段
func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {
	log.Infof("[ai-zdtc-token onHttpResponseHeaders] === [OnHttpResponseHeaders] 阶段开始 ===")

	statusStr, err := proxywasm.GetHttpResponseHeader(":status")
	if err == nil && statusStr != "" {
		if statusCode, convErr := strconv.Atoi(statusStr); convErr == nil {
			// 暂存响应状态码至请求上下文
			ctx.SetContext(ctxKeyResponseStatus, statusCode)

			if statusCode >= 400 {
				log.Warnf("[ai-zdtc-token onHttpResponseHeaders] 捕获到异常响应状态码: %d (客户端参数错误/鉴权失败/上游超时/服务异常)", statusCode)
			} else {
				log.Infof("[ai-zdtc-token onHttpResponseHeaders] 响应状态码: %d", statusCode)
			}
		}
	} else if err != nil {
		log.Warnf("[ai-zdtc-token onHttpResponseHeaders] 无法获取 :status 响应头: %#v", err)
	}

	// 依据上游实际返回的 Content-Type 判断是否为流式传输
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if strings.Contains(contentType, "text/event-stream") {
		ctx.SetContext(ctxKeyIsStream, true)
		log.Infof("[ai-zdtc-token onHttpResponseHeaders] 响应头 Content-Type 包含 text/event-stream, 锁定流式响应状态机")
	}

	// 初始化流传输统计与完整性监控状态
	ctx.SetContext(ctxKeyChunkCount, 0)
	ctx.SetContext(ctxKeyTotalChunkBytes, int64(0))
	ctx.SetContext(ctxKeyHasSeenDone, false)
	ctx.SetContext(ctxKeyIsLastChunkSeen, false)

	headers, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		log.Errorf("[ai-zdtc-token onHttpResponseHeaders] 无法获取响应头: %#v", err)
		// 获取响应头失败时跳过分析, 返回 ActionContinue 允许响应正常返回给客户端.
	} else {
		log.Infof("[ai-zdtc-token onHttpResponseHeaders] 响应头总数: %d", len(headers))
		for _, h := range headers {
			key := h[0]
			val := h[1]
			if config.Debug {
				log.Infof("[ai-zdtc-token onHttpResponseHeaders]   Header -> %s: %s", key, val)
			}
			// Header -> :status: 200
			// Header -> cache-control: no-cache
			// Header -> content-type: text/event-stream
			// Header -> server: istio-envoy
			// Header -> trace-id: 160f4175c0184349b4b596b65d829ac0
			// Header -> x-cds-request-id: 160f4175-c018-4349-b4b5-96b65d829ac0
			// Header -> date: Thu, 27 Aug 2026 06:57:49 GMT
			// Header -> x-envoy-upstream-service-time: 1181
			// Header -> access-control-allow-origin: *
			// Header -> transfer-encoding: chunked
			// Header -> req-cost-time: 1267
			// Header -> req-arrive-time: 1787813868630
			// Header -> resp-start-time: 1787813869897
		}
	}

	log.Infof("[ai-zdtc-token onHttpResponseHeaders] === [OnHttpResponseHeaders] 阶段结束 ===")
	return types.ActionContinue
}

// 5. 流式响应体分块处理阶段
func onHttpStreamResponseBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isLastChunk bool, log log.Log) []byte {
	log.Infof("[ai-zdtc-token onHttpStreamResponseBody] === [OnHttpStreamResponseBody] 阶段触发 ===")

	log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 当前数据块大小: %d 字节, 是否为最后一个数据块: %t", len(chunk), isLastChunk)

	// 1. 累计接收到的 Chunk 计数与数据字节数
	chunkCount, _ := ctx.GetContext(ctxKeyChunkCount).(int)
	chunkCount++
	ctx.SetContext(ctxKeyChunkCount, chunkCount)

	totalBytes, _ := ctx.GetContext(ctxKeyTotalChunkBytes).(int64)
	totalBytes += int64(len(chunk))
	ctx.SetContext(ctxKeyTotalChunkBytes, totalBytes)

	// 2. 监测是否收到 HTTP 传输层的结束块 (EOF)
	if isLastChunk {
		ctx.SetContext(ctxKeyIsLastChunkSeen, true)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 捕获到流式传输终止标识 (isLastChunk=true), 累计 Chunk 数: %d, 累计字节: %d", chunkCount, totalBytes)
	}

	// 3. 监测是否包含大模型 SSE 协议定义的结束标志 data: [DONE]
	if bytes.Contains(chunk, []byte("[DONE]")) {
		ctx.SetContext(ctxKeyHasSeenDone, true)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 成功检测到大模型流式终止符: [DONE]")
	}

	// 4. 监测数据流中可能包裹的异常报错 (如: data: {"error": {"message": ...}})
	if bytes.Contains(chunk, []byte(`"error":`)) {
		errMsg := gjson.GetBytes(chunk, "error.message").String()
		if errMsg != "" {
			ctx.SetContext(ctxKeyErrorMessage, errMsg)
			log.Warnf("[ai-zdtc-token onHttpStreamResponseBody] 捕获到 SSE 数据流内的异常错误说明: %s", errMsg)
		}
	}

	if config.Debug {
		if len(chunk) > 0 {
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 数据块内容: %s", string(chunk))
			if usage := tokenusage.GetTokenUsage(ctx, chunk); usage.TotalToken > 0 {
				log.Infof("[ai-zdtc-token onHttpStreamResponseBody] In-flight token usage intercepted! Model: '%s', TotalToken: %d, InputToken: %d, OutputToken: %d", usage.Model, usage.TotalToken, usage.InputToken, usage.OutputToken)
			}
		}
	}

	log.Infof("[ai-zdtc-token onHttpStreamResponseBody] === [OnHttpStreamResponseBody] 阶段结束 ===")
	return chunk
}

// 6. [核心新增]请求生命周期完全结束/清理阶段 (无论正常结束、网络断开、上游崩溃 Reset、超时均保证触发)
func onHttpStreamDone(ctx wrapper.HttpContext, config PluginConfig, log log.Log) {
	log.Infof("[ai-zdtc-token onHttpStreamDone] === [OnHttpStreamDone] 请求生命周期收尾阶段触发 ===")

	reqID, _ := ctx.GetContext(ctxKeyRequestUUID).(string)
	statusCode, _ := ctx.GetContext(ctxKeyResponseStatus).(int)
	isStream, _ := ctx.GetContext(ctxKeyIsStream).(bool)
	hasSeenDone, _ := ctx.GetContext(ctxKeyHasSeenDone).(bool)
	isLastChunkSeen, _ := ctx.GetContext(ctxKeyIsLastChunkSeen).(bool)
	chunkCount, _ := ctx.GetContext(ctxKeyChunkCount).(int)
	totalBytes, _ := ctx.GetContext(ctxKeyTotalChunkBytes).(int64)

	// 从 Envoy 内核属性中提取流终止原因、标志位与网络故障明细
	// 对应 Envoy Access Log 中的 %RESPONSE_CODE_DETAILS%、%RESPONSE_FLAGS%、%UPSTREAM_TRANSPORT_FAILURE_REASON%
	codeDetails := getEnvoyPropertyString([]string{"response", "code_details"})
	flagsUint := getEnvoyPropertyUint64([]string{"response", "flags"})
	flagsStr := parseResponseFlags(flagsUint)
	transportFailure := getEnvoyPropertyString([]string{"upstream", "transport_failure_reason"})

	log.Infof("[ai-zdtc-token onHttpStreamDone] 最终流状态汇总 -> RequestID: %s, HTTP状态码: %d, 流式(SSE): %t, 已收Chunk数: %d, 已收总字节: %d, [DONE]收到: %t, LastChunk收到: %t, Envoy终止原因: '%s', EnvoyFlags: '%s'",
		reqID, statusCode, isStream, chunkCount, totalBytes, hasSeenDone, isLastChunkSeen, codeDetails, flagsStr)

	// 判断当前请求是否发生异常中断
	isAbnormal := false
	var interruptReasons []string

	// 1. 流式完整性检查：对于流式请求, 未收到 [DONE] 或未收到最后的 EOF Chunk, 即视为传输被提前腰斩
	if isStream {
		if !hasSeenDone {
			isAbnormal = true
			interruptReasons = append(interruptReasons, "未收到大模型协议结束标记[DONE]")
		}
		if !isLastChunkSeen {
			isAbnormal = true
			interruptReasons = append(interruptReasons, "未收到HTTP传输层结束数据块(isLastChunk=false)")
		}
	}

	// 2. Envoy 终止细节检查：正常完成通常为 "via_upstream", 其他异常如 "upstream_reset_after_response_started{protocol_error}" 等
	if codeDetails != "" && codeDetails != "via_upstream" {
		isAbnormal = true
		interruptReasons = append(interruptReasons, fmt.Sprintf("Envoy终止细节: %s", codeDetails))
	}

	// 3. Envoy 响应标志检查：检查是否包含 UPE(上游协议错误)、UF(上游连接故障)、UT(上游超时)、DC(客户端断开) 等
	if flagsStr != "-" && flagsStr != "" {
		if strings.Contains(flagsStr, "UPE") || strings.Contains(flagsStr, "UF") ||
			strings.Contains(flagsStr, "UT") || strings.Contains(flagsStr, "UC") ||
			strings.Contains(flagsStr, "DC") || strings.Contains(flagsStr, "UR") {
			isAbnormal = true
			interruptReasons = append(interruptReasons, fmt.Sprintf("Envoy标志位异常: %s", flagsStr))
		}
	}

	// 结合错误类型进行详细诊断打印
	if isAbnormal {
		category := "流式传输异常"
		if strings.Contains(codeDetails, "upstream_reset") || strings.Contains(flagsStr, "UPE") || strings.Contains(flagsStr, "UF") {
			// 典型场景：如日志所示, vLLM / 上游服务在生成中途崩溃、显存溢出或对端突然关闭连接, 报 HPE_INVALID_EOF_STATE
			category = "上游服务异常崩溃/断开重置连接 (Upstream Protocol/Connection Error)"
		} else if strings.Contains(codeDetails, "downstream") || strings.Contains(flagsStr, "DC") {
			// 典型场景：客户端/前端用户主动点击“停止生成”按钮, 或前端网络切换断开
			category = "客户端主动断开/取消连接 (Downstream Client Disconnected)"
		} else if strings.Contains(codeDetails, "timeout") || strings.Contains(flagsStr, "UT") || strings.Contains(flagsStr, "SI") {
			// 典型场景：大模型推理卡顿超过网关超时阈值
			category = "流式响应超时中断 (Upstream/Stream Timeout)"
		}

		fullReason := strings.Join(interruptReasons, "; ")
		log.Errorf("[ai-zdtc-token onHttpStreamDone] [%s] RequestID: %s, 响应状态码: %d, EnvoyFlags: %s, Envoy终止原因: %s, 已收Chunk数: %d, 已收总字节: %d, [DONE]收到: %t, LastChunk收到: %t, 底层传输失败原因: '%s', 判定中断详情: [%s]",
			category, reqID, statusCode, flagsStr, codeDetails, chunkCount, totalBytes, hasSeenDone, isLastChunkSeen, transportFailure, fullReason)

		// 存入异常信息, 便于后续写日志审计落库 (TokenAuditLog.ErrorMessage)
		ctx.SetContext(ctxKeyErrorMessage, fmt.Sprintf("%s: %s (flags:%s, reasons:%s)", category, codeDetails, flagsStr, fullReason))
	} else {
		log.Infof("[ai-zdtc-token onHttpStreamDone] [请求正常结束] RequestID: %s, 响应状态码: %d, 流式(SSE): %t, 已收Chunk数: %d, 已收总字节: %d, [DONE]收到: %t, LastChunk收到: %t",
			reqID, statusCode, isStream, chunkCount, totalBytes, hasSeenDone, isLastChunkSeen)
	}

	log.Infof("[ai-zdtc-token onHttpStreamDone] === [OnHttpStreamDone] 阶段结束 ===")
}

// --- Envoy 属性获取与标志位解析辅助函数 ---

// 获取 Envoy 字符串属性
func getEnvoyPropertyString(path []string) string {
	val, err := proxywasm.GetProperty(path)
	if err != nil || len(val) == 0 {
		return ""
	}
	return string(val)
}

// 获取 Envoy 整数属性 (按小端字节序解析 uint64, 避免越界 panic)
func getEnvoyPropertyUint64(path []string) uint64 {
	val, err := proxywasm.GetProperty(path)
	if err != nil || len(val) == 0 {
		return 0
	}
	if len(val) >= 8 {
		return binary.LittleEndian.Uint64(val[:8])
	}
	if len(val) >= 4 {
		return uint64(binary.LittleEndian.Uint32(val[:4]))
	}
	return uint64(val[0])
}

// 将 Envoy response.flags 的整型位掩码解码为标准易读缩写 (完全对齐 Envoy Access Log 的 %RESPONSE_FLAGS%)
func parseResponseFlags(flags uint64) string {
	if flags == 0 {
		return "-"
	}
	flagMap := []struct {
		mask uint64
		name string
	}{
		{0x1, "UH"},        // No healthy upstream (无健康上游)
		{0x2, "UF"},        // Upstream connection failure (上游连接失败)
		{0x4, "NR"},        // No route (未匹配到路由)
		{0x8, "URX"},       // Upstream retry limit exceeded (超过上游重试次数)
		{0x10, "DC"},       // Downstream connection termination (客户端主动断开/取消)
		{0x20, "LH"},       // Local health check failed (本地健康检查失败)
		{0x40, "UT"},       // Upstream request timeout (上游请求超时)
		{0x80, "LR"},       // Local reset (本地重置)
		{0x100, "UR"},      // Upstream remote reset (上游远端重置)
		{0x200, "UC"},      // Upstream connection termination (上游连接终止)
		{0x400, "DI"},      // Delayed injection (延迟注入)
		{0x800, "FI"},      // Fault injection (故障注入)
		{0x1000, "RL"},     // Rate limited (被本地或全局限流)
		{0x2000, "UAEX"},   // Unauthorized external service (鉴权未通过)
		{0x4000, "RLSE"},   // Rate limit service error (限流服务内部错误)
		{0x8000, "IH"},     // Invalid header (非法请求头)
		{0x10000, "SI"},    // Stream idle timeout (流空闲超时)
		{0x20000, "DPE"},   // Downstream protocol error (下游协议错误)
		{0x40000, "UPE"},   // Upstream protocol error (上游协议错误 - 对应本次日志中的中断)
		{0x80000, "UMSDR"}, // Upstream max stream duration reached (达到上游最长流时长)
	}

	var res []string
	for _, item := range flagMap {
		if flags&item.mask != 0 {
			res = append(res, item.name)
		}
	}
	if len(res) == 0 {
		return fmt.Sprintf("0x%x", flags)
	}
	return strings.Join(res, ",")
}

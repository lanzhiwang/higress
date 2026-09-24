package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/tokenusage"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

// 全局日志搜索前缀标识, 方便在海量网关日志中精确定位和检索当前插件的执行链路
const traceLogPrefix = "[ai-zdtc-lanzhiwang] "

const (
	// 请求上下文 (HttpContext) 存储 Key
	ctxKeyUUID      = "ai-zdtc-lanzhiwang-req-uuid"
	ctxKeyStartTime = "ai-zdtc-lanzhiwang-start-time"

	// 默认最大缓冲上限 (100MB)
	defaultMaxBodyBytes uint32 = 100 * 1024 * 1024
)

// PluginConfig 插件配置结构
type PluginConfig struct {
	MaxBodyLogLen int  `json:"max_body_log_len"` // Body 打印的最大字节长度截断保护
	PrintHeaders  bool `json:"print_headers"`    // 是否打印 Headers 详情
	PrintBody     bool `json:"print_body"`       // 是否打印 Body 详情
}

func main() {}

// init 函数在 WebAssembly 模块加载阶段执行, 注册所有 HTTP 阶段生命周期回调
func init() {
	wrapper.SetCtx(
		// 1. 插件唯一标识符 (Plugin Name)
		"ai-zdtc-lanzhiwang",

		// 2. 插件配置解析回调
		wrapper.ParseConfig(parseConfig),

		// 3. HTTP 请求头处理阶段钩子 (生成并绑定全局唯一 UUID)
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),

		// 4. HTTP 流式请求体处理阶段钩子 (客户端发送的每个 Chunk)
		wrapper.ProcessStreamingRequestBody(onHttpStreamingRequestBody),

		// 5. HTTP 完整请求体处理阶段钩子 (请求体全部接收完毕后触发)
		wrapper.ProcessRequestBody(onHttpRequestBody),

		// 6. HTTP 响应头处理阶段钩子 (判断 SSE 流式或 Blocking 响应)
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),

		// 7. HTTP 流式响应体处理阶段钩子 (上游返回的每个 SSE/Chunk 分片)
		wrapper.ProcessStreamingResponseBody(onHttpStreamingResponseBody),

		// 8. HTTP 完整响应体处理阶段钩子 (非流式响应收集完毕后触发)
		wrapper.ProcessResponseBody(onHttpResponseBody),

		// 9. 请求流结束清理阶段钩子 (统计端到端全生命周期耗时)
		wrapper.ProcessStreamDone(onStreamDone),
	)
}

// ---------------- 阶段 1: 配置解析 ----------------
func parseConfig(configJson gjson.Result, config *PluginConfig) error {
	log.Infof(traceLogPrefix+"[parseConfig] Starting configuration parse. Raw JSON: %s", configJson.Raw)

	// 默认值设置
	config.MaxBodyLogLen = 100 * 1024 * 1024
	config.PrintHeaders = true
	config.PrintBody = true

	if configJson.Get("max_body_log_len").Exists() {
		config.MaxBodyLogLen = int(configJson.Get("max_body_log_len").Int())
	}
	if configJson.Get("print_headers").Exists() {
		config.PrintHeaders = configJson.Get("print_headers").Bool()
	}
	if configJson.Get("print_body").Exists() {
		config.PrintBody = configJson.Get("print_body").Bool()
	}

	log.Infof(traceLogPrefix+"[parseConfig] Configuration parsed successfully: MaxBodyLogLen=%d, PrintHeaders=%v, PrintBody=%v",
		config.MaxBodyLogLen, config.PrintHeaders, config.PrintBody)
	return nil
}

// ---------------- 阶段 2: HTTP 请求头处理 ----------------
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
	// 1. 生成标准的 RFC 4122 v4 UUID, 贯穿整个请求链路
	reqUUID := uuid.NewString()
	startTime := time.Now().UnixMilli()

	// 2. 存入当前请求的上下文 Context
	ctx.SetContext(ctxKeyUUID, reqUUID)
	ctx.SetContext(ctxKeyStartTime, startTime)

	// 3. 提取基础请求信息
	path, _ := proxywasm.GetHttpRequestHeader(":path")
	method, _ := proxywasm.GetHttpRequestHeader(":method")
	authority, _ := proxywasm.GetHttpRequestHeader(":authority")

	var headerLog string
	if config.PrintHeaders {
		allReqHeaders, _ := proxywasm.GetHttpRequestHeaders()
		headerLog = fmt.Sprintf(" | Headers: [%s]", formatHeaders(allReqHeaders))
	}

	log.Infof(traceLogPrefix+"[%s][onHttpRequestHeaders] >>> Received HTTP Request: %s %s, Authority: %s%s",
		reqUUID, method, path, authority, headerLog)

	// 4. 设置请求体最大缓冲上限, 确保后续 ProcessRequestBody 能够完整抓取 Payload
	ctx.SetRequestBodyBufferLimit(defaultMaxBodyBytes)

	return types.ActionContinue
}

// ---------------- 阶段 3: 流式请求体处理 (Chunk) ----------------
func onHttpStreamingRequestBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, endOfStream bool) []byte {
	reqUUID := getUUID(ctx)

	var preview string
	if config.PrintBody {
		preview = fmt.Sprintf(" | Content: %s", formatBodyPreview(chunk, config.MaxBodyLogLen))
	}

	log.Infof(traceLogPrefix+"[%s][onHttpStreamingRequestBody] >>> Received Request Chunk. Length: %d bytes, endOfStream: %v%s",
		reqUUID, len(chunk), endOfStream, preview)

	// 原样透传数据块
	return chunk
}

// ---------------- 阶段 4: 完整请求体处理 (Non-Streaming) ----------------
func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte) types.Action {
	reqUUID := getUUID(ctx)

	var preview string
	if config.PrintBody {
		preview = fmt.Sprintf(" | Content: %s", formatBodyPreview(body, config.MaxBodyLogLen))
	}

	log.Infof(traceLogPrefix+"[%s][onHttpRequestBody] >>> Received Full HTTP Request Body. Length: %d bytes%s",
		reqUUID, len(body), preview)

	return types.ActionContinue
}

// ---------------- 阶段 5: HTTP 响应头处理 ----------------
func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
	reqUUID := getUUID(ctx)

	status, _ := proxywasm.GetHttpResponseHeader(":status")
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")

	var headerLog string
	if config.PrintHeaders {
		allRespHeaders, _ := proxywasm.GetHttpResponseHeaders()
		headerLog = fmt.Sprintf(" | Response Headers: [%s]", formatHeaders(allRespHeaders))
	}

	log.Infof(traceLogPrefix+"[%s][onHttpResponseHeaders] >>> Received HTTP Response: Status: %s, Content-Type: '%s'%s",
		reqUUID, status, contentType, headerLog)

	// 调度逻辑: 判断是否为流式 SSE (text/event-stream)
	if !strings.Contains(contentType, "text/event-stream") {
		// 非流式响应: 显式指示 Envoy 开启缓冲, 确保 onHttpResponseBody 能够拿到完整报文
		log.Infof(traceLogPrefix+"[%s][onHttpResponseHeaders] Non-streaming response detected. Calling BufferResponseBody()", reqUUID)
		ctx.BufferResponseBody()
	} else {
		// 流式 SSE 响应: 不缓冲, 保持打字机毫秒级流式下发
		log.Infof(traceLogPrefix+"[%s][onHttpResponseHeaders] Streaming SSE response detected ('text/event-stream'). Allowing direct streaming", reqUUID)
	}
	ctx.SetResponseBodyBufferLimit(defaultMaxBodyBytes)

	return types.ActionContinue
}

// ---------------- 阶段 6: 流式响应体处理 (Chunk) ----------------
func onHttpStreamingResponseBody(ctx wrapper.HttpContext, config PluginConfig, data []byte, endOfStream bool) []byte {
	reqUUID := getUUID(ctx)

	var preview string
	if config.PrintBody {
		preview = fmt.Sprintf(" | Chunk Data: %s", formatBodyPreview(data, config.MaxBodyLogLen))
	}

	// 标准 Token 消耗解析 (提取 input_token, output_token, total_token, model)
	if usage := tokenusage.GetTokenUsage(ctx, data); usage.TotalToken > 0 {
		log.Infof(traceLogPrefix+"[%s][onHttpStreamingResponseBody] >>> Received SSE Response Chunk. Length: %d bytes, endOfStream: %v%s | [TokenUsage] Model: '%s', InputTokens: %d, OutputTokens: %d, TotalTokens: %d",
			reqUUID, len(data), endOfStream, preview,
			usage.Model, usage.InputToken, usage.OutputToken, usage.TotalToken)
	} else {
		log.Infof(traceLogPrefix+"[%s][onHttpStreamingResponseBody] >>> Received SSE Response Chunk. Length: %d bytes, endOfStream: %v%s",
			reqUUID, len(data), endOfStream, preview)
	}

	// 原样透传数据块给客户端
	return data
}

// ---------------- 阶段 7: 完整响应体处理 (Non-Streaming) ----------------
func onHttpResponseBody(ctx wrapper.HttpContext, config PluginConfig, body []byte) types.Action {
	reqUUID := getUUID(ctx)

	var preview string
	if config.PrintBody {
		preview = fmt.Sprintf(" | Content: %s", formatBodyPreview(body, config.MaxBodyLogLen))
	}

	// 1. 调用 Higress SDK 标准解析完整响应体中的 Token 消耗
	usage := tokenusage.GetTokenUsage(ctx, body)

	// 2. 检查是否有额外 Token 明细（例如思考链推理 Token、Prompt 缓存命中 Token）
	var extraTokenDetails string
	if rTokens, ok := usage.OutputTokenDetails["reasoning_tokens"]; ok && rTokens > 0 {
		extraTokenDetails += fmt.Sprintf(", ReasoningTokens: %d", rTokens)
	}
	if cTokens, ok := usage.InputTokenDetails["cached_tokens"]; ok && cTokens > 0 {
		extraTokenDetails += fmt.Sprintf(", CachedTokens: %d", cTokens)
	}

	// 3. 分流打印日志（确保占位符严格匹配）
	if usage.TotalToken > 0 {
		log.Infof(traceLogPrefix+"[%s][onHttpResponseBody] >>> Received Full HTTP Response Body. Length: %d bytes | [TokenUsage] Model: '%s', InputTokens: %d, OutputTokens: %d, TotalTokens: %d%s%s",
			reqUUID, len(body), usage.Model, usage.InputToken, usage.OutputToken, usage.TotalToken, extraTokenDetails, preview)
	} else {
		// 上游报错 (如 4xx/5xx) 或非 LLM 响应时无 Token 统计
		log.Infof(traceLogPrefix+"[%s][onHttpResponseBody] >>> Received Full HTTP Response Body. Length: %d bytes (No Token Usage)%s",
			reqUUID, len(body), preview)
	}

	return types.ActionContinue
}

// ---------------- 阶段 8: 请求流结束清理阶段 ----------------
func onStreamDone(ctx wrapper.HttpContext, config PluginConfig) {
	reqUUID := getUUID(ctx)

	var totalLatency int64
	if val := ctx.GetContext(ctxKeyStartTime); val != nil {
		if startTime, ok := val.(int64); ok {
			totalLatency = time.Now().UnixMilli() - startTime
		}
	}

	log.Infof(traceLogPrefix+"[%s][onStreamDone] === REQUEST LIFECYCLE COMPLETED === Total Duration: %d ms",
		reqUUID, totalLatency)
}

// ======================== 工具函数 ========================

// getUUID 从 Context 中提取全局请求 UUID, 兜底防空
func getUUID(ctx wrapper.HttpContext) string {
	if val := ctx.GetContext(ctxKeyUUID); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return "UNKNOWN-UUID"
}

// formatHeaders 将 Proxy-Wasm 提取的 HTTP 键值对数组格式化为易读的字符串
func formatHeaders(headers [][2]string) string {
	if len(headers) == 0 {
		return "<empty>"
	}
	var sb strings.Builder
	for i, h := range headers {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%s: %s", h[0], h[1]))
	}
	return sb.String()
}

// formatBodyPreview 截断长文本并安全格式化, 防止超大报文刷爆日志系统
func formatBodyPreview(data []byte, maxLen int) string {
	if len(data) == 0 {
		return "<EMPTY>"
	}
	// 超过阈值执行首尾截断保留
	if maxLen > 0 && len(data) > maxLen {
		half := maxLen / 2
		return fmt.Sprintf("%q ...[truncated %d bytes]... %q", data[:half], len(data)-maxLen, data[len(data)-half:])
	}
	return fmt.Sprintf("%q", data)
}

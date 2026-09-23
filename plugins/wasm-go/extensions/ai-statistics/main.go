package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/tokenusage"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

// 全局日志搜索前缀标识, 便于在海量网关日志中精确定位和检索 AI 统计插件的执行链路
const traceLogPrefix = "[AI-STATISTICS-TRACE] "

// formatHeaders 辅助函数: 将 Proxy-Wasm 提取的 HTTP 键值对数组格式化为易读的字符串, 便于日志排查
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

const (
	// Envoy log levels
	LogLevelTrace = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelCritical
)

func main() {}

// init 函数在 Go 包加载阶段执行(早于 main 函数).
// 在 WebAssembly 模块被 Envoy/Higress 实例化时, init 会自动注册当前插件的所有钩子、配置解析器与运行策略.
func init() {
	// wrapper.SetCtx 是 Higress 插件的核心注册入口.
	// 它使用函数式选项模式(Functional Options Pattern), 声明插件的唯一标识以及在 HTTP 各阶段的回调.
	wrapper.SetCtx(
		// 1. 插件唯一标识符 (Plugin Name)
		// 用于日志打印、指标度量 (Metrics)、Tracing 上下文以及控制面 CRD (WasmPlugin) 的匹配关联.
		"ai-statistics",

		// 2. 插件配置解析回调
		// 当网关启动或控制面 (K8s CRD / 控制台) 动态下发配置时触发, 将 JSON/YAML 转换为强类型的 Go 结构体.
		wrapper.ParseConfig(parseConfig),

		// 3. HTTP 请求头处理阶段钩子
		// 对应 Envoy 的 proxy_on_http_request_headers 阶段, 用于识别请求路由、鉴权身份、Content-Type、判断是否为 AI 流量等.
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),

		// 4. HTTP 请求体处理阶段钩子
		// 对应 Envoy 的 proxy_on_http_request_body 阶段, 用于截获并解析 LLM 请求载荷(如 prompt、model、stream 参数、tokens 预估等).
		wrapper.ProcessRequestBody(onHttpRequestBody),

		// 5. HTTP 响应头处理阶段钩子
		// 对应 Envoy 的 proxy_on_http_response_headers 阶段, 用于读取上游 LLM 的响应状态码、区分是否为 SSE 流式响应(text/event-stream).
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),

		// 6. HTTP 流式响应体处理阶段钩子 (AI 流式场景核心)
		// 针对 SSE (Server-Sent Events) 的每个 chunk 增量回调, 实现流式实时 Token 计算、首字时延 (TTFT) 记录与流式数据透传.
		wrapper.ProcessStreamingResponseBody(onHttpStreamingBody),

		// 7. HTTP 完整响应体处理阶段钩子
		// 针对非流式 (stream: false) 调用的完整响应体回调, 直接解析最终返回的 JSON 数据(如 usage.total_tokens).
		wrapper.ProcessResponseBody(onHttpResponseBody),

		// 8. Wasm 虚拟机生命周期策略: 请求数阈值自动重建
		// 泛型参数绑定配置结构体 [AIStatisticsConfig]. 当当前 Wasm 实例累计处理达到 1000 次请求后, 优雅触发 VM 重建.
		wrapper.WithRebuildAfterRequests[AIStatisticsConfig](1000),

		// 9. Wasm 虚拟机生命周期策略: 内存上限自动重建
		// 当当前 Wasm VM 实例占用的线性内存达到 200MB 时, 立即触发安全重建, 防止内存泄漏导致 Envoy 发生 OOM.
		wrapper.WithRebuildMaxMemBytes[AIStatisticsConfig](200*1024*1024),
	)
}

const (
	defaultMaxBodyBytes uint32 = 100 * 1024 * 1024
	// Context consts
	StatisticsRequestStartTime = "ai-statistics-request-start-time"
	StatisticsFirstTokenTime   = "ai-statistics-first-token-time"
	// CtxGeneralAtrribute        = "attributes"
	// CtxLogAtrribute            = "logAttributes"
	CtxStreamingBodyBuffer = "streamingBodyBuffer"
	RouteName              = "route"
	ClusterName            = "cluster"
	APIName                = "api"
	ConsumerKey            = "x-mse-consumer"
	RequestPath            = "request_path"
	SkipProcessing         = "skip_processing"

	// Session ID related
	SessionID = "session_id"

	// AI API Paths
	// PathOpenAIChatCompletions       = "/v1/chat/completions"
	// PathOpenAICompletions           = "/v1/completions"
	// PathOpenAIEmbeddings            = "/v1/embeddings"
	// PathOpenAIModels                = "/v1/models"
	// PathGeminiGenerateContent       = "/generateContent"
	// PathGeminiStreamGenerateContent = "/streamGenerateContent"

	// Source Type
	FixedValue            = "fixed_value"
	RequestHeader         = "request_header"
	RequestBody           = "request_body"
	ResponseHeader        = "response_header"
	ResponseStreamingBody = "response_streaming_body"
	ResponseBody          = "response_body"

	// Inner metric & log attributes
	LLMFirstTokenDuration  = "llm_first_token_duration"
	LLMServiceDuration     = "llm_service_duration"
	LLMDurationCount       = "llm_duration_count"
	LLMStreamDurationCount = "llm_stream_duration_count"
	ResponseType           = "response_type"
	ChatID                 = "chat_id"
	ChatRound              = "chat_round"

	// Inner span attributes
	ArmsSpanKind     = "gen_ai.span.kind"
	ArmsModelName    = "gen_ai.model_name"
	ArmsRequestModel = "gen_ai.request.model"
	ArmsInputToken   = "gen_ai.usage.input_tokens"
	ArmsOutputToken  = "gen_ai.usage.output_tokens"
	ArmsTotalToken   = "gen_ai.usage.total_tokens"

	// Extract Rule
	RuleFirst   = "first"
	RuleReplace = "replace"
	RuleAppend  = "append"

	// Built-in attributes
	// 插件提供了一些内置属性键(key), 可以直接使用而无需配置 `value_source` 和 `value`. 这些内置属性会自动从请求/响应中提取相应的值:
	// | 内置属性键 | 说明 | 适用场景 |
	// | --- | --- | --- |
	// | `question` | 用户提问内容 | 支持 OpenAI/Claude 消息格式 |
	// | `system` | 系统提示词 | 支持 Claude `/v1/messages` 的顶层 system 字段 |
	// | `answer` | AI 回答内容 | 支持 OpenAI/Claude 消息格式, 流式和非流式 |
	// | `tool_calls` | 工具调用信息 | OpenAI/Claude 工具调用 |
	// | `reasoning` | 推理过程 | OpenAI o1 等推理模型 |
	// | `reasoning_tokens` | 推理 token 数(如 o1 模型) | OpenAI Chat Completions, 从 `output_token_details.reasoning_tokens` 提取 |
	// | `cached_tokens` | 缓存命中的 token 数 | OpenAI Chat Completions, 从 `input_token_details.cached_tokens` 提取 |
	// | `input_token_details` | 输入 token 详细信息(完整对象) | OpenAI/Gemini/Anthropic, 包含缓存、工具使用等详情 |
	// | `output_token_details` | 输出 token 详细信息(完整对象) | OpenAI/Gemini/Anthropic, 包含推理 token、生成图片数等详情 |
	// 使用内置属性时, 只需设置 `key`、`apply_to_log` 等参数, 无需设置 `value_source` 和 `value`.
	// 注意:
	// - `reasoning_tokens` 和 `cached_tokens` 是从 token details 中提取的便捷字段, 适用于 OpenAI Chat Completions API
	// - `input_token_details` 和 `output_token_details` 会以 JSON 字符串形式记录完整的 token 详情对象
	BuiltinQuestionKey        = "question"
	BuiltinAnswerKey          = "answer"
	BuiltinToolCallsKey       = "tool_calls"
	BuiltinReasoningKey       = "reasoning"
	BuiltinSystemKey          = "system"
	BuiltinReasoningTokens    = "reasoning_tokens"
	BuiltinCachedTokens       = "cached_tokens"
	BuiltinInputTokenDetails  = "input_token_details"
	BuiltinOutputTokenDetails = "output_token_details"

	// Built-in attribute paths
	// Question paths (from request body)
	QuestionPathOpenAI = "messages.@reverse.0.content"
	// QuestionPathClaude = "messages.@reverse.0.content" // Claude uses same format

	// System prompt paths (from request body)
	SystemPathClaude = "system" // Claude /v1/messages has system as a top-level field

	// Answer paths (from response body - non-streaming)
	AnswerPathOpenAINonStreaming = "choices.0.message.content"
	AnswerPathClaudeNonStreaming = "content.0.text"

	// Answer paths (from response streaming body)
	AnswerPathOpenAIStreaming = "choices.0.delta.content"
	AnswerPathClaudeStreaming = "delta.text"

	// Tool calls paths (OpenAI format)
	ToolCallsPathNonStreaming = "choices.0.message.tool_calls"
	ToolCallsPathStreaming    = "choices.0.delta.tool_calls"

	// Claude/Anthropic tool calls paths (streaming)
	ClaudeEventType         = "type"
	ClaudeContentBlockType  = "content_block.type"
	ClaudeContentBlockID    = "content_block.id"
	ClaudeContentBlockName  = "content_block.name"
	ClaudeContentBlockInput = "content_block.input"
	ClaudeDeltaPartialJSON  = "delta.partial_json"
	ClaudeIndex             = "index"

	// Reasoning paths
	ReasoningPathNonStreaming = "choices.0.message.reasoning_content"
	ReasoningPathStreaming    = "choices.0.delta.reasoning_content"

	// Context key for streaming tool calls buffer
	CtxStreamingToolCallsBuffer = "streamingToolCallsBuffer"
)

// getDefaultAttributes returns the default attributes configuration for empty config
// This includes all attributes but may consume significant memory for large conversations
func getDefaultAttributes() []Attribute {
	return []Attribute{
		// Extract complete conversation history from request body
		{
			Key:         "messages",
			ValueSource: RequestBody,
			Value:       "messages",
			ApplyToLog:  true,
		},
		// Built-in attributes (no value_source needed, will be auto-extracted)
		{
			Key:        BuiltinQuestionKey,
			ApplyToLog: true,
		},
		{
			Key:        BuiltinSystemKey,
			ApplyToLog: true,
		},
		{
			Key:        BuiltinAnswerKey,
			ApplyToLog: true,
			Rule:       RuleAppend, // Streaming responses need to append content from all chunks
		},
		{
			Key:        BuiltinReasoningKey,
			ApplyToLog: true,
			Rule:       RuleAppend, // Streaming responses need to append content from all chunks
		},
		{
			Key:        BuiltinToolCallsKey,
			ApplyToLog: true,
		},
		// Token statistics (auto-extracted from response)
		{
			Key:        BuiltinReasoningTokens,
			ApplyToLog: true,
		},
		{
			Key:        BuiltinCachedTokens,
			ApplyToLog: true,
		},
		// Detailed token information
		{
			Key:        BuiltinInputTokenDetails,
			ApplyToLog: true,
		},
		{
			Key:        BuiltinOutputTokenDetails,
			ApplyToLog: true,
		},
	}
}

// getDefaultResponseAttributes returns a lightweight default attributes configuration
// for production environments with high concurrency and high latency.
// - Buffers request body for model extraction (small, essential field)
// - Does NOT extract large fields like question, system, messages
// - Does NOT buffer streaming response body (no answer, reasoning, tool_calls)
// - Only extracts token statistics from response context
func getDefaultResponseAttributes() []Attribute {
	return []Attribute{
		// Token statistics (extracted from context, no body buffering needed)
		{
			Key:        BuiltinReasoningTokens,
			ApplyToLog: true,
		},
		{
			Key:        BuiltinCachedTokens,
			ApplyToLog: true,
		},
		{
			Key:        BuiltinInputTokenDetails,
			ApplyToLog: true,
		},
		{
			Key:        BuiltinOutputTokenDetails,
			ApplyToLog: true,
		},
	}
}

// Default session ID headers in priority order
var defaultSessionHeaders = []string{
	"x-openclaw-session-key",
	"x-clawdbot-session-key",
	"x-moltbot-session-key",
	"x-agent-session",
}

// extractSessionId extracts session ID from request headers
// If customHeader is configured, it takes priority; otherwise falls back to default headers
func extractSessionId(customHeader string) string {
	log.Infof(traceLogPrefix+"[extractSessionId] Attempting to extract session ID. customHeader: '%s', defaultCandidates: %v", customHeader, defaultSessionHeaders)
	// [AI-STATISTICS-TRACE] [extractSessionId] Attempting to extract session ID. customHeader: '', defaultCandidates: [x-openclaw-session-key x-clawdbot-session-key x-moltbot-session-key x-agent-session]

	// If custom header is configured, try it first
	if customHeader != "" {
		if sessionId, _ := proxywasm.GetHttpRequestHeader(customHeader); sessionId != "" {
			log.Infof(traceLogPrefix+"[extractSessionId] Successfully matched session ID from custom header '%s': %s", customHeader, sessionId)
			return sessionId
		}
		log.Infof(traceLogPrefix+"[extractSessionId] Custom header '%s' not present or empty, falling back to defaults", customHeader)
	}
	// Fall back to default session headers in priority order
	for _, header := range defaultSessionHeaders {
		if sessionId, _ := proxywasm.GetHttpRequestHeader(header); sessionId != "" {
			log.Infof(traceLogPrefix+"[extractSessionId] Successfully matched session ID from default header candidate '%s': %s", header, sessionId)
			return sessionId
		}
	}
	log.Infof(traceLogPrefix + "[extractSessionId] No session ID header found in request.")
	return ""
}

// ToolCall represents a single tool call in the response
type ToolCall struct {
	Index    int              `json:"index,omitempty"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function ToolCallFunction `json:"function,omitempty"`
}

// ToolCallFunction represents the function details in a tool call
type ToolCallFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// StreamingToolCallsBuffer holds the state for assembling streaming tool calls
type StreamingToolCallsBuffer struct {
	ToolCalls       map[int]*ToolCall // keyed by index (OpenAI format)
	InToolBlock     map[int]bool      // tracks which indices are in tool_use blocks (Claude format)
	ArgumentsBuffer map[int]string    // buffers partial JSON arguments (Claude format)
}

// extractStreamingToolCalls extracts and assembles tool calls from streaming response chunks (OpenAI format)
func extractStreamingToolCalls(data []byte, buffer *StreamingToolCallsBuffer) *StreamingToolCallsBuffer {
	if buffer == nil {
		buffer = &StreamingToolCallsBuffer{
			ToolCalls:       make(map[int]*ToolCall),
			InToolBlock:     make(map[int]bool),
			ArgumentsBuffer: make(map[int]string),
		}
	}

	chunks := bytes.Split(bytes.TrimSpace(wrapper.UnifySSEChunk(data)), []byte("\n\n"))
	for _, chunk := range chunks {
		toolCallsResult := gjson.GetBytes(chunk, ToolCallsPathStreaming)
		if !toolCallsResult.Exists() || !toolCallsResult.IsArray() {
			continue
		}

		for _, tcResult := range toolCallsResult.Array() {
			index := int(tcResult.Get("index").Int())

			// Get or create tool call entry
			tc, exists := buffer.ToolCalls[index]
			if !exists {
				tc = &ToolCall{Index: index}
				buffer.ToolCalls[index] = tc
				log.Infof(traceLogPrefix+"[extractStreamingToolCalls] Discovered new OpenAI tool call at index: %d", index)
			}

			// Update fields if present
			if id := tcResult.Get("id").String(); id != "" {
				tc.ID = id
			}
			if tcType := tcResult.Get("type").String(); tcType != "" {
				tc.Type = tcType
			}
			if funcName := tcResult.Get("function.name").String(); funcName != "" {
				tc.Function.Name = funcName
				log.Infof(traceLogPrefix+"[extractStreamingToolCalls] Tool call index %d function name updated to: %s", index, funcName)
			}
			// Append arguments (they come in chunks)
			if args := tcResult.Get("function.arguments").String(); args != "" {
				tc.Function.Arguments += args
				log.Infof(traceLogPrefix+"[extractStreamingToolCalls] Tool call index %d arguments appended chunk: %s, total length now: %d", index, args, len(tc.Function.Arguments))
			}
		}
	}

	return buffer
}

// extractClaudeStreamingToolCalls extracts and assembles tool calls from Claude/Anthropic streaming response chunks
// Claude format uses events: content_block_start, content_block_delta, content_block_stop
func extractClaudeStreamingToolCalls(data []byte, buffer *StreamingToolCallsBuffer) *StreamingToolCallsBuffer {
	if buffer == nil {
		buffer = &StreamingToolCallsBuffer{
			ToolCalls:       make(map[int]*ToolCall),
			InToolBlock:     make(map[int]bool),
			ArgumentsBuffer: make(map[int]string),
		}
	}

	chunks := bytes.Split(bytes.TrimSpace(wrapper.UnifySSEChunk(data)), []byte("\n\n"))
	for _, chunk := range chunks {
		// Get event type
		eventType := gjson.GetBytes(chunk, ClaudeEventType)
		if !eventType.Exists() {
			continue
		}

		switch eventType.String() {
		case "content_block_start":
			// Check if this is a tool_use block
			contentBlockType := gjson.GetBytes(chunk, ClaudeContentBlockType)
			if contentBlockType.Exists() && contentBlockType.String() == "tool_use" {
				index := int(gjson.GetBytes(chunk, ClaudeIndex).Int())

				// Create tool call entry
				tc := &ToolCall{Index: index}

				// Extract id and name
				if id := gjson.GetBytes(chunk, ClaudeContentBlockID).String(); id != "" {
					tc.ID = id
				}
				if name := gjson.GetBytes(chunk, ClaudeContentBlockName).String(); name != "" {
					tc.Function.Name = name
				}
				tc.Type = "tool_use"

				buffer.ToolCalls[index] = tc
				buffer.InToolBlock[index] = true
				buffer.ArgumentsBuffer[index] = ""

				log.Infof(traceLogPrefix+"[extractClaudeStreamingToolCalls] Claude tool_use started. Index: %d, ID: %s, Name: %s", index, tc.ID, tc.Function.Name)

				// Try to extract initial input if present
				if input := gjson.GetBytes(chunk, ClaudeContentBlockInput); input.Exists() {
					if inputMap, ok := input.Value().(map[string]interface{}); ok {
						if jsonBytes, err := json.Marshal(inputMap); err == nil {
							buffer.ArgumentsBuffer[index] = string(jsonBytes)
							log.Infof(traceLogPrefix+"[extractClaudeStreamingToolCalls] Claude tool_use initial input parsed: %s", buffer.ArgumentsBuffer[index])
						}
					}
				}
			}

		case "content_block_delta":
			// Check if we're in a tool block
			index := int(gjson.GetBytes(chunk, ClaudeIndex).Int())
			if buffer.InToolBlock[index] {
				// Accumulate partial JSON arguments
				partialJSON := gjson.GetBytes(chunk, ClaudeDeltaPartialJSON)
				if partialJSON.Exists() {
					buffer.ArgumentsBuffer[index] += partialJSON.String()
					log.Infof(traceLogPrefix+"[extractClaudeStreamingToolCalls] Claude tool_use delta input appended. Index: %d, partial: %s", index, partialJSON.String())
				}
			}

		case "content_block_stop":
			// Finalize the tool call if we were in a tool block
			index := int(gjson.GetBytes(chunk, ClaudeIndex).Int())
			if buffer.InToolBlock[index] {
				buffer.InToolBlock[index] = false

				// Parse accumulated arguments and set them
				if tc, exists := buffer.ToolCalls[index]; exists {
					tc.Function.Arguments = buffer.ArgumentsBuffer[index]
					log.Infof(traceLogPrefix+"[extractClaudeStreamingToolCalls] Claude tool_use finalized. Index: %d, Final Arguments: %s", index, tc.Function.Arguments)
				}
			}
		}
	}

	return buffer
}

// getToolCallsFromBuffer converts the buffer to a sorted slice of tool calls
func getToolCallsFromBuffer(buffer *StreamingToolCallsBuffer) []ToolCall {
	if buffer == nil || len(buffer.ToolCalls) == 0 {
		return nil
	}

	// Find max index to create properly sized slice
	maxIndex := 0
	for idx := range buffer.ToolCalls {
		if idx > maxIndex {
			maxIndex = idx
		}
	}

	result := make([]ToolCall, 0, len(buffer.ToolCalls))
	for i := 0; i <= maxIndex; i++ {
		if tc, exists := buffer.ToolCalls[i]; exists {
			result = append(result, *tc)
		}
	}
	log.Infof(traceLogPrefix+"[getToolCallsFromBuffer] Assembled %d tool calls from buffer.", len(result))
	return result
}

// TracingSpan is the tracing span configuration.
// Attribute 配置说明:

// | 名称 | 数据类型 | 填写要求 | 默认值 | 描述 |
// | --- | --- | --- | --- | --- |
// | `key` | string | 必填 | - | attribute 名称 |
// | `value_source` | string | 必填 | - | attribute 取值来源, 可选值为 `fixed_value`, `request_header`, `request_body`, `response_header`, `response_body`, `response_streaming_body` |
// | `value` | string | 必填 | - | attribute 取值 key value/path |
// | `default_value` | string | 非必填 | - | attribute 默认值 |
// | `rule` | string | 非必填 | - | 从流式响应中提取 attribute 的规则, 可选值为 `first`, `replace`, `append` |
// | `apply_to_log` | bool | 非必填 | false | 是否将提取的信息记录在日志中 |
// | `apply_to_span` | bool | 非必填 | false | 是否将提取的信息记录在链路追踪 span 中 |
// | `trace_span_key` | string | 非必填 | - | 链路追踪 attribute key, 默认会使用 `key` 的设置 |
// | `as_separate_log_field` | bool | 非必填 | false | 记录日志时是否作为单独的字段, 日志字段名使用 `key` 的设置 |

// `value_source` 的各种取值含义如下:

// - `fixed_value`: 固定值
// - `request_header`: attribute 值通过 http 请求头获取, value 配置为 header key
// - `request_body`: attribute 值通过请求 body 获取, value 配置格式为 gjson 的 jsonpath
// - `response_header`: attribute 值通过 http 响应头获取, value 配置为 header key
// - `response_body`: attribute 值通过响应 body 获取, value 配置格式为 gjson 的 jsonpath
// - `response_streaming_body`: attribute 值通过流式响应 body 获取, value 配置格式为 gjson 的 jsonpath

// 当 `value_source` 为 `response_streaming_body` 时, 应当配置 `rule`, 用于指定如何从流式 body 中获取指定值, 取值含义如下:

// - `first`: 多个 chunk 中取第一个有效 chunk 的值
// - `replace`: 多个 chunk 中取最后一个有效 chunk 的值
// - `append`: 拼接多个有效 chunk 中的值, 可用于获取回答内容
type Attribute struct {
	Key                string `json:"key"`
	ValueSource        string `json:"value_source"`
	Value              string `json:"value"`
	DefaultValue       string `json:"default_value,omitempty"`
	Rule               string `json:"rule,omitempty"`
	ApplyToLog         bool   `json:"apply_to_log,omitempty"`
	ApplyToSpan        bool   `json:"apply_to_span,omitempty"`
	TraceSpanKey       string `json:"trace_span_key,omitempty"`
	AsSeparateLogField bool   `json:"as_separate_log_field,omitempty"`
}

type AIStatisticsConfig struct {
	valueLengthLimit int
	// Attributes to be recorded in log & span
	attributes []Attribute
	// If there exist attributes extracted from request body, request body should be buffered
	shouldBufferRequestBody bool
	// If there exist attributes extracted from streaming body, chunks should be buffered
	shouldBufferStreamingBody bool
	// Metrics
	// TODO: add more metrics in Gauge and Histogram format
	counterMetrics map[string]proxywasm.MetricCounter
	// If disableOpenaiUsage is true, model/input_token/output_token logs will be skipped
	disableOpenaiUsage bool
	// Path suffixes to enable the plugin on
	enablePathSuffixes []string
	// Content types to enable response body buffering
	enableContentTypes []string
	// Session ID header name (if configured, takes priority over default headers)
	sessionIdHeader string
}

func generateMetricName(route, cluster, model, consumer, metricName string) string {
	return fmt.Sprintf("route.%s.upstream.%s.model.%s.consumer.%s.metric.%s", route, cluster, model, consumer, metricName)
}

func getRouteName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err != nil {
		log.Infof(traceLogPrefix+"[getRouteName] Failed to get route_name property from host: %v", err)
		return "-", err
	} else {
		log.Infof(traceLogPrefix+"[getRouteName] Successfully fetched route_name: %s", string(raw))
		// [AI-STATISTICS-TRACE] [getRouteName] Successfully fetched route_name: ai-route-115.internal
		return string(raw), nil
	}
}

func getAPIName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err != nil {
		log.Infof(traceLogPrefix+"[getAPIName] Failed to get route_name property from host: %v", err)
		return "-", err
	} else {
		parts := strings.Split(string(raw), "@")
		if len(parts) < 3 {
			log.Infof(traceLogPrefix+"[getAPIName] route_name '%s' does not match API format (parts < 3)", string(raw))
			// [AI-STATISTICS-TRACE] [getAPIName] route_name 'ai-route-115.internal' does not match API format (parts < 3)
			return "-", errors.New("not api type")
		} else {
			apiName := strings.Join(parts[:3], "@")
			log.Infof(traceLogPrefix+"[getAPIName] Parsed API name: %s", apiName)
			return apiName, nil
		}
	}
}

func getClusterName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err != nil {
		log.Infof(traceLogPrefix+"[getClusterName] Failed to get cluster_name property from host: %v", err)
		return "-", err
	} else {
		log.Infof(traceLogPrefix+"[getClusterName] Successfully fetched cluster_name: %s", string(raw))
		// [AI-STATISTICS-TRACE] [getClusterName] Successfully fetched cluster_name: outbound|443||llm-provider-1.internal.dns
		return string(raw), nil
	}
}

func (config *AIStatisticsConfig) incrementCounter(metricName string, inc uint64) {
	if inc == 0 {
		log.Infof(traceLogPrefix+"[incrementCounter] Metric increment is 0 for '%s', skipping", metricName)
		return
	}
	counter, ok := config.counterMetrics[metricName]
	if !ok {
		log.Infof(traceLogPrefix+"[incrementCounter] Defining new counter metric on host: %s", metricName)
		counter = proxywasm.DefineCounterMetric(metricName)
		config.counterMetrics[metricName] = counter
	}
	log.Infof(traceLogPrefix+"[incrementCounter] Incrementing metric '%s' by %d", metricName, inc)
	counter.Increment(inc)
}

// isPathEnabled checks if the request path matches any of the enabled path suffixes
func isPathEnabled(requestPath string, enabledSuffixes []string) bool {
	if len(enabledSuffixes) == 0 {
		log.Infof(traceLogPrefix+"[isPathEnabled] No path suffixes configured (empty slice). Allowing all paths: %s", requestPath)
		return true // If no path suffixes configured, enable for all
	}

	// Remove query parameters from path
	pathWithoutQuery := requestPath
	if queryPos := strings.Index(requestPath, "?"); queryPos != -1 {
		pathWithoutQuery = requestPath[:queryPos]
	}

	// Check if path ends with any enabled suffix
	for _, suffix := range enabledSuffixes {
		if strings.HasSuffix(pathWithoutQuery, suffix) {
			log.Infof(traceLogPrefix+"[isPathEnabled] Path '%s' matched configured suffix '%s'", pathWithoutQuery, suffix)
			return true
		}
	}
	log.Infof(traceLogPrefix+"[isPathEnabled] Path '%s' did not match any enabled suffixes %v", pathWithoutQuery, enabledSuffixes)
	return false
}

// isContentTypeEnabled checks if the content type matches any of the enabled content types
func isContentTypeEnabled(contentType string, enabledContentTypes []string) bool {
	if len(enabledContentTypes) == 0 {
		log.Infof(traceLogPrefix+"[isContentTypeEnabled] No content types configured (empty slice). Allowing all content types: %s", contentType)
		return true // If no content types configured, enable for all
	}

	for _, enabledType := range enabledContentTypes {
		if strings.Contains(contentType, enabledType) {
			log.Infof(traceLogPrefix+"[isContentTypeEnabled] Content-Type '%s' matched configured rule '%s'", contentType, enabledType)
			return true
		}
	}
	log.Infof(traceLogPrefix+"[isContentTypeEnabled] Content-Type '%s' did not match any configured types %v", contentType, enabledContentTypes)
	return false
}

// parseConfig 是插件的配置解析入口函数.
// 当网关启动或控制面(Higress Console / K8s CRD)下发/更新配置时由 wrapper 框架调用.
// configJson: 通过 tidwall/gjson 高性能解析的原始 JSON 配置, 无需全量反序列化即可快速按路径检索.
// config: 绑定的强类型 Go 结构体, 用于存储解析后的运行态策略, 伴随整个插件的生命周期.
func parseConfig(configJson gjson.Result, config *AIStatisticsConfig) error {
	log.Infof(traceLogPrefix+"[parseConfig] Starting configuration parse. Raw JSON: %s", configJson.Raw)
	// [AI-STATISTICS-TRACE] [parseConfig] Starting configuration parse. Raw JSON: {"use_default_attributes":true}

	// Check if use_default_attributes is enabled
	// 1. 获取预设模式标志(开箱即用模式)
	// use_default_attributes: 是否启用全量预设属性采集(采集 Prompt、Answer、Tokens、模型名、耗时等完整 Tracing 数据)
	// use_default_attributes | bool | 非必填 | false | 是否使用默认完整属性配置, 包含 messages、answer、question 等所有字段. 适用于调试、审计场景
	useDefaultAttributes := configJson.Get("use_default_attributes").Bool()

	// Check if use_default_response_attributes is enabled (lightweight mode)
	// use_default_response_attributes: 是否启用轻量级模式(只采集响应侧的元数据与 Token, 忽略庞大的请求 Prompt)
	// use_default_response_attributes | bool | 非必填 | false | 是否使用轻量级默认属性配置(推荐), 包含 model 和 token 统计, 不缓冲流式响应体. 适用于高并发生产环境
	useDefaultResponseAttributes := configJson.Get("use_default_response_attributes").Bool()

	log.Infof(traceLogPrefix+"[parseConfig] Mode flags - use_default_attributes: %v, use_default_response_attributes: %v", useDefaultAttributes, useDefaultResponseAttributes)
	// [AI-STATISTICS-TRACE] [parseConfig] Mode flags - use_default_attributes: true, use_default_response_attributes: false

	// Parse tracing span attributes setting.
	// 2. 读取用户自定义配置的 OpenTelemetry/Tracing Span 属性列表
	// attributes | []Attribute | 非必填 | - | 用户希望记录在 log/span 中的信息
	attributeConfigs := configJson.Get("attributes").Array()

	// Set value_length_limit
	// 3. 设置属性值的最大长度限制(内存安全截断阈值, 防止超大 Prompt/Completion 撑爆 Wasm 内存)
	// 如果用户显式传了则取用户的配置, 否则先给一个通用的兜底默认值 32000 字符 (~32KB)
	// value_length_limit | int | 非必填 | 4000 | 记录的单个 value 的长度限制 |
	if configJson.Get("value_length_limit").Exists() {
		config.valueLengthLimit = int(configJson.Get("value_length_limit").Int())
		log.Infof(traceLogPrefix+"[parseConfig] Explicitly configured value_length_limit: %d", config.valueLengthLimit)
	} else {
		config.valueLengthLimit = 32000
		log.Infof(traceLogPrefix+"[parseConfig] Using default value_length_limit: %d", config.valueLengthLimit)
		// [AI-STATISTICS-TRACE] [parseConfig] Using default value_length_limit: 32000
	}

	// Parse attributes or use defaults
	// 4. 解析属性收集策略: 采用三态互斥优先级(全量预设 > 轻量预设 > 完全自定义)
	if useDefaultAttributes {
		// [模式 A] 全量预设模式: 加载默认全量属性列表(包括用户原始输入、模型回复、调用元数据)
		config.attributes = getDefaultAttributes()

		// Update value_length_limit to default when using default attributes
		// 全量模式下若用户未显式指定阈值, 将上限放宽至 10MB, 以容纳长文本上下文(如 128k prompt、长论文解读)
		if !configJson.Get("value_length_limit").Exists() {
			config.valueLengthLimit = 10485760 // 10MB
			log.Infof(traceLogPrefix+"[parseConfig] Full default mode: value_length_limit adjusted to 10MB (%d bytes)", config.valueLengthLimit)
			// [AI-STATISTICS-TRACE] [parseConfig] Full default mode: value_length_limit adjusted to 10MB (10485760 bytes)
		}
		log.Infof("Using default attributes configuration")
		log.Infof(traceLogPrefix+"[parseConfig] Loaded %d default attributes into config.attributes", len(config.attributes))
		// [AI-STATISTICS-TRACE] [parseConfig] Loaded 10 default attributes into config.attributes

	} else if useDefaultResponseAttributes {
		// [模式 B] 轻量响应模式: 只加载响应属性列表(如 answer 摘要、首字时延 TTFT、token 统计)
		config.attributes = getDefaultResponseAttributes()

		// Use a reasonable default for lightweight mode
		// 轻量模式下将长度阈值收紧为 4000 字符 (~4KB), 最大程度节省 Wasm 堆内存开销
		if !configJson.Get("value_length_limit").Exists() {
			config.valueLengthLimit = 4000
			log.Infof(traceLogPrefix+"[parseConfig] Lightweight mode: value_length_limit adjusted to %d bytes", config.valueLengthLimit)
		}
		log.Infof("Using default response attributes configuration (lightweight mode)")
		log.Infof(traceLogPrefix+"[parseConfig] Loaded %d default response attributes into config.attributes", len(config.attributes))

	} else {
		// [模式 C] 用户完全自定义模式: 解析用户在 attributes 中逐条声明的采集规则
		log.Infof(traceLogPrefix+"[parseConfig] Custom mode: Parsing %d custom attributes...", len(attributeConfigs))
		config.attributes = make([]Attribute, len(attributeConfigs))
		for i, attributeConfig := range attributeConfigs {
			attribute := Attribute{}
			// 使用标准 json 反序列化将单条属性配置转为结构体
			err := json.Unmarshal([]byte(attributeConfig.Raw), &attribute)
			if err != nil {
				log.Errorf("parse config failed, %v", err)
				log.Infof(traceLogPrefix+"[parseConfig] Error unmarshaling attribute at index %d: %v", i, err)
				return err
			}
			// 严格校验属性合并冲突规则(Fail-Fast 机制): 必须是合法的策略之一
			// 空("")代表默认覆盖, RuleFirst 保留首次值, RuleReplace 强制替换, RuleAppend 累加字符串
			if attribute.Rule != "" && attribute.Rule != RuleFirst && attribute.Rule != RuleReplace && attribute.Rule != RuleAppend {
				log.Infof(traceLogPrefix+"[parseConfig] Invalid rule '%s' for attribute '%s'", attribute.Rule, attribute.Key)
				return errors.New("value of rule must be one of [nil, first, replace, append]")
			}
			config.attributes[i] = attribute
			log.Infof(traceLogPrefix+"[parseConfig] Attribute[%d] loaded: Key=%s, ValueSource=%s, Value=%s, Rule=%s, ApplyToLog=%v, ApplyToSpan=%v",
				i, attribute.Key, attribute.ValueSource, attribute.Value, attribute.Rule, attribute.ApplyToLog, attribute.ApplyToSpan)
		}
	}

	log.Infof(traceLogPrefix+"[parseConfig] ====== Effective Attributes Details (%d total) ======", len(config.attributes))
	for idx, attr := range config.attributes {
		log.Infof(traceLogPrefix+"[parseConfig] Attribute[%d] -> Key: '%s', ValueSource: '%s', Value: '%s', DefaultValue: '%s', Rule: '%s', ApplyToLog: %v, ApplyToSpan: %v, TraceSpanKey: '%s', AsSeparateLogField: %v",
			idx, attr.Key, attr.ValueSource, attr.Value, attr.DefaultValue, attr.Rule, attr.ApplyToLog, attr.ApplyToSpan, attr.TraceSpanKey, attr.AsSeparateLogField)
	}
	// [AI-STATISTICS-TRACE] [parseConfig] Attribute[0] -> Key: 'messages', ValueSource: 'request_body', Value: 'messages', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
	// [AI-STATISTICS-TRACE] [parseConfig] Attribute[1] -> Key: 'question', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
	// [AI-STATISTICS-TRACE] [parseConfig] Attribute[2] -> Key: 'system', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
	// [AI-STATISTICS-TRACE] [parseConfig] Attribute[3] -> Key: 'answer', ValueSource: '', Value: '', DefaultValue: '', Rule: 'append', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
	// [AI-STATISTICS-TRACE] [parseConfig] Attribute[4] -> Key: 'reasoning', ValueSource: '', Value: '', DefaultValue: '', Rule: 'append', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
	// [AI-STATISTICS-TRACE] [parseConfig] Attribute[5] -> Key: 'tool_calls', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
	// [AI-STATISTICS-TRACE] [parseConfig] Attribute[6] -> Key: 'reasoning_tokens', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
	// [AI-STATISTICS-TRACE] [parseConfig] Attribute[7] -> Key: 'cached_tokens', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
	// [AI-STATISTICS-TRACE] [parseConfig] Attribute[8] -> Key: 'input_token_details', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
	// [AI-STATISTICS-TRACE] [parseConfig] Attribute[9] -> Key: 'output_token_details', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false

	if attrJSON, err := json.Marshal(config.attributes); err == nil {
		log.Infof(traceLogPrefix+"[parseConfig] Attributes JSON dump: %s", string(attrJSON))
		// [AI-STATISTICS-TRACE] [parseConfig] Attributes JSON dump:
		// [
		// 	{
		// 		"key": "messages",
		// 		"value_source": "request_body",
		// 		"value": "messages",
		// 		"apply_to_log": true
		// 	},
		// 	{
		// 		"key": "question",
		// 		"value_source": "",
		// 		"value": "",
		// 		"apply_to_log": true
		// 	},
		// 	{
		// 		"key": "system",
		// 		"value_source": "",
		// 		"value": "",
		// 		"apply_to_log": true
		// 	},
		// 	{
		// 		"key": "answer",
		// 		"value_source": "",
		// 		"value": "",
		// 		"rule": "append",
		// 		"apply_to_log": true
		// 	},
		// 	{
		// 		"key": "reasoning",
		// 		"value_source": "",
		// 		"value": "",
		// 		"rule": "append",
		// 		"apply_to_log": true
		// 	},
		// 	{
		// 		"key": "tool_calls",
		// 		"value_source": "",
		// 		"value": "",
		// 		"apply_to_log": true
		// 	},
		// 	{
		// 		"key": "reasoning_tokens",
		// 		"value_source": "",
		// 		"value": "",
		// 		"apply_to_log": true
		// 	},
		// 	{
		// 		"key": "cached_tokens",
		// 		"value_source": "",
		// 		"value": "",
		// 		"apply_to_log": true
		// 	},
		// 	{
		// 		"key": "input_token_details",
		// 		"value_source": "",
		// 		"value": "",
		// 		"apply_to_log": true
		// 	},
		// 	{
		// 		"key": "output_token_details",
		// 		"value_source": "",
		// 		"value": "",
		// 		"apply_to_log": true
		// 	}
		// ]
	} else {
		log.Infof(traceLogPrefix+"[parseConfig] Failed to serialize config.attributes to JSON: %v", err)
	}
	log.Infof(traceLogPrefix + "[parseConfig] ===================================================================================")

	// Check if any attribute needs request body or streaming body buffering
	// 5. 核心性能优化: 预计算数据流的"缓冲需求"(Lazy Buffering / 按需缓冲分析)
	// 在 Envoy 中, 缓冲 HTTP Body(尤其是 SSE 流)是非常沉重的性能开销.
	// 这里遍历所有待采集的 attributes, 反向推导本次请求生命周期中是否"必须"拦截并缓冲 Body.
	for idx, attribute := range config.attributes {
		log.Infof(traceLogPrefix+"[parseConfig] Check if any attribute needs request body or streaming body buffering Attribute[%d] -> Key: '%s', ValueSource: '%s', Value: '%s', DefaultValue: '%s', Rule: '%s', ApplyToLog: %v, ApplyToSpan: %v, TraceSpanKey: '%s', AsSeparateLogField: %v",
			idx, attribute.Key, attribute.ValueSource, attribute.Value, attribute.DefaultValue, attribute.Rule, attribute.ApplyToLog, attribute.ApplyToSpan, attribute.TraceSpanKey, attribute.AsSeparateLogField)
		// [AI-STATISTICS-TRACE] [parseConfig] Check if any attribute needs request body or streaming body buffering Attribute[0] -> Key: 'messages', ValueSource: 'request_body', Value: 'messages', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
		// [AI-STATISTICS-TRACE] [parseConfig] Check if any attribute needs request body or streaming body buffering Attribute[1] -> Key: 'question', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
		// [AI-STATISTICS-TRACE] [parseConfig] Check if any attribute needs request body or streaming body buffering Attribute[2] -> Key: 'system', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
		// [AI-STATISTICS-TRACE] [parseConfig] Check if any attribute needs request body or streaming body buffering Attribute[3] -> Key: 'answer', ValueSource: '', Value: '', DefaultValue: '', Rule: 'append', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
		// [AI-STATISTICS-TRACE] [parseConfig] Check if any attribute needs request body or streaming body buffering Attribute[4] -> Key: 'reasoning', ValueSource: '', Value: '', DefaultValue: '', Rule: 'append', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
		// [AI-STATISTICS-TRACE] [parseConfig] Check if any attribute needs request body or streaming body buffering Attribute[5] -> Key: 'tool_calls', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
		// [AI-STATISTICS-TRACE] [parseConfig] Check if any attribute needs request body or streaming body buffering Attribute[6] -> Key: 'reasoning_tokens', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
		// [AI-STATISTICS-TRACE] [parseConfig] Check if any attribute needs request body or streaming body buffering Attribute[7] -> Key: 'cached_tokens', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
		// [AI-STATISTICS-TRACE] [parseConfig] Check if any attribute needs request body or streaming body buffering Attribute[8] -> Key: 'input_token_details', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false
		// [AI-STATISTICS-TRACE] [parseConfig] Check if any attribute needs request body or streaming body buffering Attribute[9] -> Key: 'output_token_details', ValueSource: '', Value: '', DefaultValue: '', Rule: '', ApplyToLog: true, ApplyToSpan: false, TraceSpanKey: '', AsSeparateLogField: false

		// Check for request body buffering
		// 如果有属性需要从完整请求体解析(例如从 prompt JSON 里提取特定字段), 则开启请求体缓冲
		if attribute.ValueSource == RequestBody {
			config.shouldBufferRequestBody = true
			log.Infof(traceLogPrefix+"[parseConfig] Attribute '%s' requires request body buffering", attribute.Key)
		}

		// Check for streaming body buffering (explicitly configured)
		// 如果显式指定要从流式响应体提取数据, 则开启流式 Body 缓冲
		if attribute.ValueSource == ResponseStreamingBody {
			config.shouldBufferStreamingBody = true
			log.Infof(traceLogPrefix+"[parseConfig] Attribute '%s' requires streaming response body buffering", attribute.Key)
		}

		// For built-in attributes without explicit ValueSource, check default sources
		// 处理内置属性(开发者未显式指定 ValueSource, 但该属性属于内置已知字段)
		if attribute.ValueSource == "" && isBuiltinAttribute(attribute.Key) {
			defaultSources := getBuiltinAttributeDefaultSources(attribute.Key)
			for _, src := range defaultSources {
				if src == RequestBody {
					config.shouldBufferRequestBody = true
					log.Infof(traceLogPrefix+"[parseConfig] Built-in attribute '%s' requires request body buffering via default source %s", attribute.Key, src)
				}
				// Only answer/reasoning/tool_calls need actual body buffering
				// Token-related attributes are extracted from context, not from body
				// 极致优化分支: 判断是否真正需要缓冲流式响应体.
				// 像 prompt_tokens、total_tokens 很多时候从上下文状态机或最终的 usage 字段即可获得, 无需缓冲整个 SSE 文本块;
				// 只有当明确需要捕获 answer、reasoning(深度思考过程)、tool_calls 时, 才需要缓冲 chunk 组装完整文本.
				if src == ResponseStreamingBody && needsBodyBuffering(attribute.Key) {
					config.shouldBufferStreamingBody = true
					log.Infof(traceLogPrefix+"[parseConfig] Built-in attribute '%s' requires streaming body buffering via default source %s", attribute.Key, src)
				}
			}
		}
	}
	log.Infof(traceLogPrefix+"[parseConfig] Buffering decision - shouldBufferRequestBody: %v, shouldBufferStreamingBody: %v", config.shouldBufferRequestBody, config.shouldBufferStreamingBody)
	// [AI-STATISTICS-TRACE] [parseConfig] Buffering decision - shouldBufferRequestBody: true, shouldBufferStreamingBody: true

	// Metric settings
	// 6. 初始化 Envoy 统计度量指标的缓存 Map
	// 用于缓存 proxywasm.DefineMetric 返回的计数器 ID, 避免运行时频繁向 Host 调用定义指标
	config.counterMetrics = make(map[string]proxywasm.MetricCounter)

	// Parse openai usage config setting.
	// 7. 解析是否禁用 OpenAI 官方 Usage 兼容解析
	// disable_openai_usage | bool | 非必填 | false | 非 openai 兼容协议时, model、token 的支持非标, 配置为 true 时可以避免报错 |
	config.disableOpenaiUsage = configJson.Get("disable_openai_usage").Bool()
	log.Infof(traceLogPrefix+"[parseConfig] disable_openai_usage: %v", config.disableOpenaiUsage)
	// [AI-STATISTICS-TRACE] [parseConfig] disable_openai_usage: false

	// Parse path suffix configuration
	// 8. 解析匹配的 URL 路径后缀(Path Suffix Filtering)
	// enable_path_suffixes | []string | 非必填 | [] | 只对这些特定路径后缀的请求生效, 可以配置为 "\*" 以匹配所有路径(通配符检查会优先进行以提高性能). 如果为空数组, 则对所有路径生效 |
	pathSuffixes := configJson.Get("enable_path_suffixes").Array()
	config.enablePathSuffixes = make([]string, 0, len(pathSuffixes))

	// If use_default_attributes or use_default_response_attributes is enabled and enable_path_suffixes is not configured, use default path suffixes
	// 若启用了预设模式且用户未显式配后缀, 自动装配主流大模型标准路由: OpenAI 的 /completions 与 Anthropic 的 /messages
	if (useDefaultAttributes || useDefaultResponseAttributes) && !configJson.Get("enable_path_suffixes").Exists() {
		config.enablePathSuffixes = []string{"/completions", "/messages"}
		log.Infof("Using default path suffixes: /completions, /messages")
	} else {
		// Process manually configured path suffixes
		// 解析用户手动配置的后缀列表
		for _, suffix := range pathSuffixes {
			suffixStr := suffix.String()
			// 关键优化: 如果配置包含通配符 "*", 表示匹配全路径.
			// 清空切片代表"无限制过滤", 后续在请求处理热路径(Fast Path)只需判断 len == 0 即可 O(1) 放行, 无需逐个字符串匹配.
			if suffixStr == "*" {
				// Clear the suffixes list since * means all paths are enabled
				config.enablePathSuffixes = make([]string, 0)
				log.Infof(traceLogPrefix + "[parseConfig] Wildcard '*' detected in enable_path_suffixes. Collapsed into empty slice for O(1) bypass.")
				break
			}
			config.enablePathSuffixes = append(config.enablePathSuffixes, suffixStr)
		}
	}
	log.Infof(traceLogPrefix+"[parseConfig] Final effective enable_path_suffixes: %v", config.enablePathSuffixes)
	// [AI-STATISTICS-TRACE] [parseConfig] Final effective enable_path_suffixes: [/completions /messages]

	// Parse content type configuration
	// 9. 解析 Content-Type 内容类型过滤器
	// enable_content_types | []string | 非必填 | [] | 只对这些内容类型的响应进行缓冲处理. 如果为空数组, 则对所有内容类型生效 |
	contentTypes := configJson.Get("enable_content_types").Array()
	config.enableContentTypes = make([]string, 0, len(contentTypes))

	for _, contentType := range contentTypes {
		contentTypeStr := contentType.String()
		// 同理: 遇到通配符 "*" 则置空切片, 运行态享受 O(1) 无条件放行优化
		if contentTypeStr == "*" {
			// Clear the content types list since * means all content types are enabled
			config.enableContentTypes = make([]string, 0)
			log.Infof(traceLogPrefix + "[parseConfig] Wildcard '*' detected in enable_content_types. Collapsed into empty slice for O(1) bypass.")
			break
		}
		config.enableContentTypes = append(config.enableContentTypes, contentTypeStr)
	}
	log.Infof(traceLogPrefix+"[parseConfig] Final effective enable_content_types: %v", config.enableContentTypes)
	// [AI-STATISTICS-TRACE] [parseConfig] Final effective enable_content_types: []

	// Parse session ID header configuration
	// 10. 解析会话追踪 ID 对应的 HTTP Header 名(如 x-session-id)
	// 支持将网关日志、链路追踪与上游业务的会话上下文(Session ID)关联对齐
	// session_id_header | string | 非必填 | - | 指定读取 session ID 的 header 名称. 如果不配置, 将按以下优先级自动查找: `x-openclaw-session-key`、`x-clawdbot-session-key`、`x-moltbot-session-key`、`x-agent-session`. session ID 可用于追踪多轮 Agent 对话 |
	if sessionIdHeader := configJson.Get("session_id_header"); sessionIdHeader.Exists() {
		config.sessionIdHeader = sessionIdHeader.String()
		log.Infof(traceLogPrefix+"[parseConfig] Configured custom session_id_header: %s", config.sessionIdHeader)
	}

	log.Infof(traceLogPrefix + "[parseConfig] ==================== Effective AIStatisticsConfig Dump ====================")
	log.Infof(traceLogPrefix+"[parseConfig] config.valueLengthLimit: %d bytes", config.valueLengthLimit)
	log.Infof(traceLogPrefix+"[parseConfig] config.attributes: total %d items loaded", len(config.attributes))
	log.Infof(traceLogPrefix+"[parseConfig] config.shouldBufferRequestBody: %t", config.shouldBufferRequestBody)
	log.Infof(traceLogPrefix+"[parseConfig] config.shouldBufferStreamingBody: %t", config.shouldBufferStreamingBody)
	log.Infof(traceLogPrefix+"[parseConfig] config.disableOpenaiUsage: %t", config.disableOpenaiUsage)
	log.Infof(traceLogPrefix+"[parseConfig] config.enablePathSuffixes: %v (len: %d)", config.enablePathSuffixes, len(config.enablePathSuffixes))
	log.Infof(traceLogPrefix+"[parseConfig] config.enableContentTypes: %v (len: %d)", config.enableContentTypes, len(config.enableContentTypes))
	log.Infof(traceLogPrefix+"[parseConfig] config.sessionIdHeader: '%s'", config.sessionIdHeader)
	log.Infof(traceLogPrefix+"[parseConfig] config.counterMetrics: initialized map (capacity/length: %d)", len(config.counterMetrics))
	log.Infof(traceLogPrefix + "[parseConfig] ===========================================================================")
	// [AI-STATISTICS-TRACE] [parseConfig] config.valueLengthLimit: 10485760 bytes
	// [AI-STATISTICS-TRACE] [parseConfig] config.attributes: total 10 items loaded
	// [AI-STATISTICS-TRACE] [parseConfig] config.shouldBufferRequestBody: true
	// [AI-STATISTICS-TRACE] [parseConfig] config.shouldBufferStreamingBody: true
	// [AI-STATISTICS-TRACE] [parseConfig] config.disableOpenaiUsage: false
	// [AI-STATISTICS-TRACE] [parseConfig] config.enablePathSuffixes: [/completions /messages] (len: 2)
	// [AI-STATISTICS-TRACE] [parseConfig] config.enableContentTypes: [] (len: 0)
	// [AI-STATISTICS-TRACE] [parseConfig] config.sessionIdHeader: ''
	// [AI-STATISTICS-TRACE] [parseConfig] config.counterMetrics: initialized map (capacity/length: 0)

	log.Infof(traceLogPrefix + "[parseConfig] Plugin configuration parsing completed successfully.")
	return nil
}

// onHttpRequestHeaders 是 HTTP 请求头处理阶段的核心回调函数.
// 对应 Envoy Proxy-Wasm 的 proxy_on_http_request_headers 阶段.
// 此时网关已完成 TLS 握手、HTTP 报文头解析以及路由匹配, 正在准备向下游或过滤器链传递.
func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig) types.Action {
	// 详细输出请求头信息
	allReqHeaders, _ := proxywasm.GetHttpRequestHeaders()
	log.Infof(traceLogPrefix+"[onHttpRequestHeaders] >>> Received HTTP Request Headers: [%s]", formatHeaders(allReqHeaders))
	// [AI-STATISTICS-TRACE] [onHttpRequestHeaders] >>> Received HTTP Request Headers:
	// [
	// 	:authority: higress-gateway.higress-system.svc.cluster.local,
	// 	:path: /v1/chat/completions,
	// 	:method: POST,
	// 	:scheme: http,
	// 	user-agent: Go-http-client/1.1,
	// 	accept: text/event-stream,
	// 	authorization: Bearer iabjwhwdaryy32nyux6bmv0h,
	// 	content-type: application/json,
	// 	accept-encoding: gzip,
	// 	x-forwarded-for: 10.42.1.148,
	// 	x-forwarded-proto: http,
	// 	x-envoy-original-host: higress-gateway.higress-system.svc.cluster.local,
	// 	x-envoy-external-address: 10.42.1.148,
	// 	x-request-id: 7da249a6-0601-479d-a55f-54533106d90f,
	// 	x-envoy-decorator-operation: istio-autogenerated-k8s-ingress-higress-system-ai-route-111.internal-61af79b04c9c8e70:80/*,
	// 	x-higress-original-model: DEEPSEEK,
	// 	x-higress-llm-model: DEEPSEEK,
	// 	x-higress-llm-model-final: Vendor3/DeepSeek-V4-Flash,
	// 	x-mse-consumer: 1-403238158,
	// 	x-mse-cuser: 1-1
	// ]

	// Check if request path matches enabled suffixes
	// 1. 获取 HTTP 伪头 :path (Path + Query), 进行白名单/后缀路由匹配
	requestPath, _ := proxywasm.GetHttpRequestHeader(":path")
	log.Infof(traceLogPrefix+"[onHttpRequestHeaders] Checking path '%s' against enabled suffixes %v", requestPath, config.enablePathSuffixes)
	// [AI-STATISTICS-TRACE] [onHttpRequestHeaders] Checking path '/v1/chat/completions' against enabled suffixes [/completions /messages]

	if !isPathEnabled(requestPath, config.enablePathSuffixes) {
		// [快速旁路优化 / Fast Bypass Path]
		// 如果当前请求路径不符合配置的 AI API 后缀(例如访问了 /healthz 或普通 REST 接口)
		log.Debugf("ai-statistics: skipping request for path %s (not in enabled suffixes)", requestPath)
		log.Infof(traceLogPrefix+"[onHttpRequestHeaders] Path '%s' is not enabled. Setting SkipProcessing=true and calling DontReadRequestBody / DontReadResponseBody", requestPath)
		// Set skip processing flag and avoid reading request/response body

		// 标记当前请求上下文为"跳过后续处理"
		ctx.SetContext(SkipProcessing, true)
		// 核心性能保护: 显式告知 Envoy 框架"完全不要拦截/缓冲后续的请求体和响应体"
		// 避免非 AI 流量因 Wasm 插件介入而造成不必要的 CPU 消耗与流式阻塞
		ctx.DontReadRequestBody()
		ctx.DontReadResponseBody()

		// 立即放行请求, 让 Envoy 继续下一个 Filter 或转发至 Upstream
		return types.ActionContinue
	}

	// 2. 冻结 Envoy 路由重匹配(Disable Reroute)
	// 禁止后续的 Header 改动引发 Envoy 重新计算路由表, 消除不必要的路由树遍历开销, 保证路由结果的确定性
	ctx.DisableReroute()
	log.Infof(traceLogPrefix + "[onHttpRequestHeaders] Called DisableReroute() to lock route decisions")

	// 3. 提取网关内部路由与集群元数据, 用于多维度的监控度量 (Metrics Aggregation)
	route, _ := getRouteName()     // ai-route-115.internal
	cluster, _ := getClusterName() // outbound|443||llm-provider-1.internal.dns
	api, apiError := getAPIName()
	// 如果 Higress 上配置了面向用户的更高维度的 API Name, 则优先覆盖内部低阶的 route 标识
	if apiError == nil {
		log.Infof(traceLogPrefix+"[onHttpRequestHeaders] API Name '%s' found, overriding route '%s'", api, route)
		route = api
	} else {
		log.Infof(traceLogPrefix+"[onHttpRequestHeaders] API Name error/not found: %v, keeping route: '%s'", apiError, route)
		// [AI-STATISTICS-TRACE] [onHttpRequestHeaders] API Name error/not found: not api type, keeping route: 'ai-route-115.internal'
	}

	// 将提取的拓扑元数据暂存到请求生命周期 Context 中, 供后续 Body 阶段及统计上报时复用
	ctx.SetContext(RouteName, route)
	ctx.SetContext(ClusterName, cluster)
	// 同时作为用户自定义属性写入链路追踪(Tracing / Span Attribute)与访问日志(Access Log)
	ctx.SetUserAttribute(APIName, api)
	log.Infof(traceLogPrefix+"[onHttpRequestHeaders] Topology context saved - Route: '%s', Cluster: '%s', API: '%s'", route, cluster, api)
	// [AI-STATISTICS-TRACE] [onHttpRequestHeaders] Topology context saved - Route: 'ai-route-115.internal', Cluster: 'outbound|443||llm-provider-1.internal.dns', API: '-'

	// 4. 记录请求到达网关的物理起始时间戳(毫秒级)
	// 该时间是计算 TTFT (Time To First Token 首字耗时) 和全局网关端到端延迟(Latency)的绝对基准
	startTime := time.Now().UnixMilli()
	ctx.SetContext(StatisticsRequestStartTime, startTime)
	log.Infof(traceLogPrefix+"[onHttpRequestHeaders] Stored StatisticsRequestStartTime: %d ms", startTime)
	// [AI-STATISTICS-TRACE] [onHttpRequestHeaders] Stored StatisticsRequestStartTime: 1790133804691 ms

	// 5. 保存请求路径与调用方凭据(Consumer)
	if requestPath, _ := proxywasm.GetHttpRequestHeader(":path"); requestPath != "" {
		ctx.SetContext(RequestPath, requestPath)
		log.Infof(traceLogPrefix+"[onHttpRequestHeaders] Saved RequestPath in context: %s", requestPath)
		// [AI-STATISTICS-TRACE] [onHttpRequestHeaders] Saved RequestPath in context: /v1/chat/completions
	}
	// ConsumerKey 是 Higress 认证鉴权插件(如 key-auth、jwt-auth)注入的调用方身份标识
	// 记录该标识以实现多租户维度(按 Tenant/User/App)的 Token 计量与限额统计
	if consumer, _ := proxywasm.GetHttpRequestHeader(ConsumerKey); consumer != "" {
		ctx.SetContext(ConsumerKey, consumer)
		log.Infof(traceLogPrefix+"[onHttpRequestHeaders] Found consumer header '%s': %s", ConsumerKey, consumer)
		// [AI-STATISTICS-TRACE] [onHttpRequestHeaders] Found consumer header 'x-mse-consumer': 1-403238158
	} else {
		log.Infof(traceLogPrefix+"[onHttpRequestHeaders] No consumer header '%s' present", ConsumerKey)
	}

	// Always buffer request body to extract model field
	// This is essential for metrics and logging
	// 6. 核心动作: 强制要求 Envoy 缓冲并向插件暴露请求体(Request Body)
	// 在 AI 网关中, 决定计费与调用逻辑的 model(如 gpt-4o、deepseek-r1)深藏在 JSON 请求体中.
	// 这里预先设定缓冲上限(防止巨型 Payload 引发 OOM), 告知后续阶段必须读取 Body.
	ctx.SetRequestBodyBufferLimit(defaultMaxBodyBytes)
	log.Infof(traceLogPrefix+"[onHttpRequestHeaders] SetRequestBodyBufferLimit set to %d bytes", defaultMaxBodyBytes)
	// [AI-STATISTICS-TRACE] [onHttpRequestHeaders] SetRequestBodyBufferLimit set to 104857600 bytes

	// Extract session ID from headers
	// 7. 会话 ID(Session Tracking)透传与提取
	// 支持按多轮会话(Conversation / Session)聚合上下文与统计对话轮次
	sessionId := extractSessionId(config.sessionIdHeader)
	if sessionId != "" {
		ctx.SetUserAttribute(SessionID, sessionId)
		log.Infof(traceLogPrefix+"[onHttpRequestHeaders] Session ID set to UserAttribute: %s", sessionId)
	}

	// Set span attributes for ARMS.
	// 8. OpenTelemetry / 阿里 ARMS 链路追踪属性染色
	// 将当前 Span 明确标记为 "LLM" 类型, 使 APM 系统渲染专有的 AI 大模型监控看板
	setSpanAttribute(ArmsSpanKind, "LLM")
	log.Infof(traceLogPrefix+"[onHttpRequestHeaders] Set Span attribute %s = 'LLM'", ArmsSpanKind)
	// [AI-STATISTICS-TRACE] [onHttpRequestHeaders] Set Span attribute gen_ai.span.kind = 'LLM'
	// Set user defined log & span attributes which type is fixed_value

	// 9. 尽早绑定静态与头部属性(Fail-Safe 策略)
	// 在请求头阶段先处理 FixedValue 和 RequestHeader 属性并注入 Span
	// 确保即使后续上游服务超时、连接重置导致流中断, 当前的 Trace 中依然包含了关键的用户身份与元数据
	log.Infof(traceLogPrefix + "[onHttpRequestHeaders] Processing attributes from FixedValue...")
	setAttributeBySource(ctx, config, FixedValue, nil)
	// Set user defined log & span attributes which type is request_header
	log.Infof(traceLogPrefix + "[onHttpRequestHeaders] Processing attributes from RequestHeader...")
	setAttributeBySource(ctx, config, RequestHeader, nil)

	// 返回 Continue, 指示 Envoy 当前 Header 阶段处理完成, 继续向下流转
	log.Infof(traceLogPrefix + "[onHttpRequestHeaders] <<< Finished onHttpRequestHeaders, returning ActionContinue")
	return types.ActionContinue
}

// onHttpRequestBody 是 HTTP 请求体处理阶段的核心回调函数.
// 对应 Envoy Proxy-Wasm 的 proxy_on_http_request_body 阶段.
// 此时网关已完成对下游客户端请求体(Body)的接收与拼装, 数据以字节切片 body 的形式注入.
func onHttpRequestBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte) types.Action {
	log.Infof(traceLogPrefix+"[onHttpRequestBody] >>> Received HTTP Request Body. Length: %d bytes, Content: %s", len(body), string(body))
	// [AI-STATISTICS-TRACE] [onHttpRequestBody] >>> Received HTTP Request Body. Length: 334 bytes, Content: {"infer_url":"http://higress-gateway.higress-system.svc.cluster.local/v1/chat/completions","max_tokens":0,"messages":[{"role":"user","content":"😀 你可以提供哪些服务？"}],"model":"Vendor3/DeepSeek-V4-Flash","model_type":"text_generation","question_id":"","repetition_penalty":1,"stream":true,"temperature":0.8,"top_p":0.9}

	// Check if processing should be skipped
	// 1. 快速旁路检查(Fast-Bypass Guard)
	// 如果前置的 onHttpRequestHeaders 已将该请求标记为 SkipProcessing(如非 AI 路由、静态资源、健康检查),
	// 则直接放行, 避免对无关请求进行昂贵的 JSON 反序列化与文本解析.
	if ctx.GetBoolContext(SkipProcessing, false) {
		log.Infof(traceLogPrefix + "[onHttpRequestBody] SkipProcessing is true. Bypassing request body processing.")
		return types.ActionContinue
	}

	// Only process request body if we need to extract attributes from it
	// 2. 按需提取用户自定义请求体属性(Lazy/Conditional Extraction)
	// 仅当用户在配置中显式声明了需要从 RequestBody 提取字段(shouldBufferRequestBody 为 true),
	// 且 Body 非空时, 才执行正则或 JSONPath 属性提取, 写入 Tracing Span 与日志.
	if config.shouldBufferRequestBody && len(body) > 0 {
		log.Infof(traceLogPrefix + "[onHttpRequestBody] Extracting user-defined attributes from RequestBody...")
		// Set user defined log & span attributes.
		setAttributeBySource(ctx, config, RequestBody, body)
	} else {
		log.Infof(traceLogPrefix+"[onHttpRequestBody] Skip RequestBody attribute extraction (shouldBufferRequestBody=%v, bodyLen=%d)", config.shouldBufferRequestBody, len(body))
	}

	// Extract model from request body if available, otherwise try path
	// 3. 核心诉求: 跨厂商多态提取模型名称(Model Extraction)
	// 模型名称(如 gpt-4o、claude-3-5-sonnet、gemini-1.5-pro)是 AI 网关计量、计费与限流的第一要素.
	requestModel := "UNKNOWN"

	// [厂商分支 A] 针对主流行业规范(OpenAI、Anthropic/Claude、DeepSeek、Ollama 等)
	// 这些厂商将模型名称作为 JSON 顶层字段存放于请求体中: {"model": "xxx", "messages": [...]}
	// 使用 gjson.GetBytes 直接在 byte slice 上进行指针扫描, 实现零内存分配(Zero-Allocation)的高性能检索.
	if len(body) > 0 {
		if model := gjson.GetBytes(body, "model"); model.Exists() {
			requestModel = model.String()
			log.Infof(traceLogPrefix+"[onHttpRequestBody] Successfully extracted model from JSON body: %s", requestModel)
			// [AI-STATISTICS-TRACE] [onHttpRequestBody] Successfully extracted model from JSON body: Vendor3/DeepSeek-V4-Flash
		} else {
			log.Infof(traceLogPrefix + "[onHttpRequestBody] 'model' field not found in JSON body, will attempt path extraction")
		}
	}
	// If model not found in body, try to extract from path (Gemini style)

	// [厂商分支 B] 针对 Google Gemini API(非 OpenAI 协议兼容格式)
	// Gemini 原生 REST 规范未在 Body 中存放 model 字段, 而是将模型名嵌入在 URI 路径中:
	// 例如: POST /v1beta/models/gemini-1.5-pro:generateContent
	if requestModel == "UNKNOWN" {
		requestPath := ctx.GetStringContext(RequestPath, "")
		log.Infof(traceLogPrefix+"[onHttpRequestBody] Attempting Gemini model extraction from RequestPath: %s", requestPath)
		// 快速判断是否为 Gemini 的调用动词, 避免不必要的正则计算
		if strings.Contains(requestPath, "generateContent") || strings.Contains(requestPath, "streamGenerateContent") { // Google Gemini GenerateContent
			// 通过命名捕获组正则匹配, 从路径提取 model 字段
			reg := regexp.MustCompile(`^.*/(?P<api_version>[^/]+)/models/(?P<model>[^:]+):\w+Content$`)
			matches := reg.FindStringSubmatch(requestPath)
			if len(matches) == 3 {
				requestModel = matches[2]
				log.Infof(traceLogPrefix+"[onHttpRequestBody] Successfully extracted Gemini model from path: %s", requestModel)
			} else {
				log.Infof(traceLogPrefix+"[onHttpRequestBody] Path regex did not match expected Gemini pattern. Matches: %v", matches)
			}
		}
	}

	// 将统一提取出的模型名称存入内部 Context, 供后续计算 Token 消耗、Prometheus 标签绑定复用
	ctx.SetContext(tokenusage.CtxKeyRequestModel, requestModel)
	// 同时将模型名称写入链路追踪(Tracing Span), 使可观测系统可按模型聚合调用链
	setSpanAttribute(ArmsRequestModel, requestModel)
	log.Infof(traceLogPrefix+"[onHttpRequestBody] Final determined request model: '%s' (saved to context and span)", requestModel)

	// Set the number of conversation rounds (only if body is available)
	// 4. 业务洞察: 计算多轮对话轮次(Chat Rounds / User Prompt Count)
	// 统计历史上下文中用户实际发起的提问次数, 用于识别单轮问答、长会话深度以及评估 Prompt 上下文膨胀情况.
	userPromptCount := 0
	if len(body) > 0 {
		// [格式 1] OpenAI 与 Claude 规范: 使用 "messages" 数组, 且消息对象含有 "role" 字段
		if messages := gjson.GetBytes(body, "messages"); messages.Exists() && messages.IsArray() {
			msgArray := messages.Array()
			log.Infof(traceLogPrefix+"[onHttpRequestBody] Found OpenAI/Claude 'messages' array with %d elements", len(msgArray))
			// OpenAI and Claude/Anthropic format - both use "messages" array with "role" field
			for _, msg := range msgArray {
				// 仅统计 role 为 user 的输入, 排除 system、assistant 以及 tool/function 的输出
				if msg.Get("role").String() == "user" {
					userPromptCount += 1
				}
			}
			log.Infof(traceLogPrefix+"[onHttpRequestBody] Counted %d user prompts in 'messages'", userPromptCount)
			// [格式 2] Google Gemini 规范: 使用 "contents" 数组
		} else if contents := gjson.GetBytes(body, "contents"); contents.Exists() && contents.IsArray() {
			contentArray := contents.Array()
			log.Infof(traceLogPrefix+"[onHttpRequestBody] Found Gemini 'contents' array with %d elements", len(contentArray))
			// Google Gemini GenerateContent
			for _, content := range contentArray {
				// Gemini 协议规定: role 字段是可选的, 当不存在 role 时隐式等价于 "user"
				if !content.Get("role").Exists() || content.Get("role").String() == "user" {
					userPromptCount += 1
				}
			}
			log.Infof(traceLogPrefix+"[onHttpRequestBody] Counted %d user prompts in 'contents'", userPromptCount)
		} else {
			log.Infof(traceLogPrefix + "[onHttpRequestBody] Neither 'messages' nor 'contents' array found in body")
		}
	}
	// 将会话轮次作为用户属性注册, 最终沉淀到访问日志(Access Log)和 OpenTelemetry Span 中
	ctx.SetUserAttribute(ChatRound, userPromptCount)
	log.Infof(traceLogPrefix+"[onHttpRequestBody] ChatRound set to %d", userPromptCount)

	// Write log
	// 5. 结构化日志持久化与放行
	// debugLogAiLog 输出调试日志(便于开发者本地排错)
	debugLogAiLog(ctx)
	// WriteUserAttributeToLogWithKey 是 Higress SDK 提供的核心能力:
	// 将当前请求生命周期内累积的所有 UserAttribute 以统一的 JSON 格式写入 Envoy 访问日志中的指定 Key(如 ai_log)
	_ = ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
	log.Infof(traceLogPrefix + "[onHttpRequestBody] Flushed UserAttributes to access log with key: " + wrapper.AILogKey)

	// 告知 Envoy 放行请求体, 继续向下游 upstream 发送
	log.Infof(traceLogPrefix + "[onHttpRequestBody] <<< Finished onHttpRequestBody, returning ActionContinue")
	return types.ActionContinue
}

// onHttpResponseHeaders 是 HTTP 响应头处理阶段的核心回调函数.
// 对应 Envoy Proxy-Wasm ABI 的 proxy_on_http_response_headers 阶段.
// 此时上游(LLM 供应商)已完成首字节推理并回传了 HTTP 状态码与响应头, 网关正准备处理响应数据流.
func onHttpResponseHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig) types.Action {
	allRespHeaders, _ := proxywasm.GetHttpResponseHeaders()
	log.Infof(traceLogPrefix+"[onHttpResponseHeaders] >>> Received HTTP Response Headers: [%s]", formatHeaders(allRespHeaders))
	// [AI-STATISTICS-TRACE] [onHttpResponseHeaders] >>> Received HTTP Response Headers:
	// [
	// 	:status: 200,
	// 	cache-control: no-cache,
	// 	trace-id: c54c77d1d3cf454f8e70b6b10dd1b7ca,
	// 	x-cds-request-id: c54c77d1-d3cf-454f-8e70-b6b10dd1b7ca,
	// 	x-envoy-upstream-service-time: 3497,
	// 	access-control-allow-origin: *,
	// 	req-cost-time: 3620,
	// 	resp-start-time: 1790133808273,
	// 	content-type: text/event-stream,
	// 	server: istio-envoy,
	// 	date: Wed, 23 Sep 2026 03:23:28 GMT,
	// 	transfer-encoding: chunked,
	// 	req-arrive-time: 1790133804652
	// ]

	// 1. 获取上游返回的 Content-Type 响应头
	// Envoy 会将 Header Key 规范化为小写, 因此这里直接检索 "content-type"
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	log.Infof(traceLogPrefix+"[onHttpResponseHeaders] Content-Type received: '%s'", contentType)
	// [AI-STATISTICS-TRACE] [onHttpResponseHeaders] Content-Type received: 'text/event-stream'

	// 2. 响应类型白名单门禁(Content-Type 旁路校验)
	// 判断上游返回的媒体类型是否在配置允许的统计名单中(如 application/json, text/event-stream)
	if !isContentTypeEnabled(contentType, config.enableContentTypes) {
		// [旁路裁剪路径 / Response Bypass Path]
		// 如果上游返回的不是合法的 AI 响应类型(例如 502/504 网关报错返回的 text/html, 或非 AI 业务的二进制文件)
		log.Debugf("ai-statistics: skipping response for content type %s (not in enabled content types)", contentType)
		log.Infof(traceLogPrefix+"[onHttpResponseHeaders] Content-Type '%s' not enabled. Setting SkipProcessing=true and calling DontReadResponseBody", contentType)
		// Set skip processing flag and avoid reading response body

		// 标记当前上下文状态为"跳过处理", 通知后续可能注册的响应体钩子不要执行分析
		ctx.SetContext(SkipProcessing, true)

		// 核心性能保护: 显式告知 Envoy 运行时"不要对该响应的 Response Body 进行任何内存缓冲或 Wasm 拦截"
		// 数据包将在 Envoy 核心层以零拷贝流式透传给客户端, 完全卸载当前插件对非目标流量的计算负担
		ctx.DontReadResponseBody()

		// 立即放行响应头, 继续后续处理
		return types.ActionContinue
	}

	// 3. 核心调度逻辑: 流式(SSE)与非流式(Blocking JSON)的分流决策
	// 检查 Content-Type 是否包含 "text/event-stream"(大模型标准的 Server-Sent Events 流式输出)
	if !strings.Contains(contentType, "text/event-stream") {
		// [分支 A: 非流式响应 (stream: false)]
		// 对应标准的 application/json 返回. 显式指示 Envoy 开启响应体缓冲(BufferResponseBody).
		// Envoy 会在内部暂存所有上游响应 chunks, 直到收到 EOF (End-of-Stream) 组装成完整的 HTTP Body 后,
		// 才会触发后置的 ProcessResponseBody(onHttpResponseBody) 回调, 以便插件解析完整的 usage.total_tokens.
		log.Infof(traceLogPrefix + "[onHttpResponseHeaders] Non-streaming response detected (no 'text/event-stream'). Calling BufferResponseBody() to buffer full payload.")
		ctx.BufferResponseBody()
	} else {
		// [分支 B: 流式响应 (stream: true)]
		// 若包含 "text/event-stream", 严禁调用 ctx.BufferResponseBody()!
		// 此时 Envoy 会直接以 Chunk 为单位逐片流式触发 ProcessStreamingResponseBody(onHttpStreamingBody),
		// 确保打字机效果(TTFT)不受任何缓冲阻滞, 毫秒级透传给下游终端.
		log.Infof(traceLogPrefix + "[onHttpResponseHeaders] Streaming SSE response detected ('text/event-stream'). Allowing direct streaming to downstream.")
	}

	// Set user defined log & span attributes.
	// 4. 提取上游响应头属性并注入链路追踪(Tracing Span)
	// 上游模型往往会在响应头中返回关键元数据(如速率限制 x-ratelimit-remaining-*、上游请求 ID x-request-id 等).
	// 在 Header 阶段第一时间抓取这些指标, 并沉淀为 OpenTelemetry Span 属性.
	log.Infof(traceLogPrefix + "[onHttpResponseHeaders] Extracting attributes from ResponseHeader...")
	setAttributeBySource(ctx, config, ResponseHeader, nil)

	// 5. 放行响应头, 指示 Envoy 继续向下游客户端推送 Header
	log.Infof(traceLogPrefix + "[onHttpResponseHeaders] <<< Finished onHttpResponseHeaders, returning ActionContinue")
	return types.ActionContinue
}

// onHttpStreamingBody 是 HTTP 流式响应体处理阶段的核心回调函数.
// 对应 Envoy Proxy-Wasm 的 proxy_on_http_response_body 流式切片阶段.
// 针对 text/event-stream (SSE) 流量, 该函数在一个 HTTP 事务中会被触发多次(每个 chunk 触发一次).
// data: 当前接收到的上游数据切片(通常为类似 "data: {...}\n\n" 的 SSE 报文分块).
// endOfStream: 布尔标志位, true 表示当前数据块是该响应流的最后一个分片(流结束).
// 返回值 []byte: 最终发送给下游客户端的数据块. 若无篡改需求, 直接原样返回 data 即可.
func onHttpStreamingBody(ctx wrapper.HttpContext, config AIStatisticsConfig, data []byte, endOfStream bool) []byte {
	log.Infof(traceLogPrefix+"[onHttpStreamingBody] >>> Received SSE Chunk. Length: %d bytes, endOfStream: %v, Chunk Data: %s", len(data), endOfStream, string(data))

	// Check if processing should be skipped
	// 1. 快速旁路守卫(Fast Bypass Guard)
	// 流式回调频率极高(一次对话可能触发数十至上百次).
	// 若前置阶段标记了 SkipProcessing(非 AI 流量或非目标 Content-Type), 必须首行极速返回, 杜绝无效计算.
	if ctx.GetBoolContext(SkipProcessing, false) {
		log.Infof(traceLogPrefix + "[onHttpStreamingBody] SkipProcessing is true. Transparently forwarding chunk.")
		return data
	}

	// Buffer stream body for record log & span attributes
	// 2. 内存安全防御: 按需缓冲流式响应体(Conditional Stream Buffering)
	// 只有当配置解析阶段(parseConfig)明确推导出用户需要 answer、reasoning 等完整文本内容时,
	// shouldBufferStreamingBody 才会为 true. 此时将当前 chunk 拼接到上下文缓冲区中.
	// 若用户仅仅做 Token 统计和指标度量, 此分支完全跳过, 避免在 Wasm 堆中积攒巨型字符串造成 OOM.
	if config.shouldBufferStreamingBody {
		streamingBodyBuffer, ok := ctx.GetContext(CtxStreamingBodyBuffer).([]byte)
		if !ok {
			streamingBodyBuffer = data
			log.Infof(traceLogPrefix+"[onHttpStreamingBody] Initialized CtxStreamingBodyBuffer with %d bytes", len(data))
		} else {
			oldLen := len(streamingBodyBuffer)
			streamingBodyBuffer = append(streamingBodyBuffer, data...)
			log.Infof(traceLogPrefix+"[onHttpStreamingBody] Appended %d bytes to CtxStreamingBodyBuffer. Total length: %d -> %d bytes", len(data), oldLen, len(streamingBodyBuffer))
		}
		ctx.SetContext(CtxStreamingBodyBuffer, streamingBodyBuffer)
	}

	// 3. 标记链路类型为流式调用
	ctx.SetUserAttribute(ResponseType, "stream")

	// 4. 多厂商协议归一化: 提取多轮会话 ID(ChatID / MessageID)
	// 使用 Higress 提供的 GetValueFromBody 在当前 chunk 中检索 ID 字段.
	// 抹平不同厂商规范:
	// - OpenAI/DeepSeek 规范: "id" 或 "response.id"
	// - Google Gemini 规范: "responseId"
	// - Anthropic Claude 规范: "message.id"
	if chatID := wrapper.GetValueFromBody(data, []string{
		"id",
		"response.id",
		"responseId", // Gemini generateContent
		"message.id", // anthropic/claude messages
	}); chatID != nil {
		log.Infof(traceLogPrefix+"[onHttpStreamingBody] Extracted ChatID from chunk: %s", chatID.String())
		// [AI-STATISTICS-TRACE] [onHttpStreamingBody] Extracted ChatID from chunk: cds_c54c77d1-d3cf-454f-8e70-b6b10dd1b7ca
		ctx.SetUserAttribute(ChatID, chatID.String())
	}

	// Get requestStartTime from http context
	// 5. 校验请求起始基准时间戳(在 onHttpRequestHeaders 阶段存入)
	requestStartTime, ok := ctx.GetContext(StatisticsRequestStartTime).(int64)
	if !ok {
		log.Error("failed to get requestStartTime from http context")
		log.Infof(traceLogPrefix + "[onHttpStreamingBody] Error: StatisticsRequestStartTime not found in context!")
		return data
	}

	// If this is the first chunk, record first token duration metric and span attribute
	// 6. 状态锁机制: 计算首字时延(TTFT - Time To First Token)
	// 通过检查 StatisticsFirstTokenTime 是否为空作为单向锁(Latch).
	// 保证只有在收到[首个分块]时, 才触发当前系统时间的记录, 计算出从客户端建连发起到网关吐出第一个 Token 的真实耗时.
	if ctx.GetContext(StatisticsFirstTokenTime) == nil {
		firstTokenTime := time.Now().UnixMilli()
		ctx.SetContext(StatisticsFirstTokenTime, firstTokenTime)
		ttft := firstTokenTime - requestStartTime
		ctx.SetUserAttribute(LLMFirstTokenDuration, ttft)
		log.Infof(traceLogPrefix+"[onHttpStreamingBody] FIRST TOKEN DETECTED! TTFT: %d ms (startTime: %d, firstTokenTime: %d)", ttft, requestStartTime, firstTokenTime)
	}

	// Set information about this request
	// 7. 流中 Token 消耗实时拦截(In-Flight Token Usage Interception)
	// 在 OpenAI 及兼容协议中, 开启 stream_options: {"include_usage": true} 时,
	// 模型会在倒数第 1 或第 2 个 chunk 携带 usage 信息.
	if !config.disableOpenaiUsage {
		// tokenusage.GetTokenUsage 封装了解析 SSE 数据包中 usage 字段的逻辑
		if usage := tokenusage.GetTokenUsage(ctx, data); usage.TotalToken > 0 {
			log.Infof(traceLogPrefix+"[onHttpStreamingBody] In-flight token usage intercepted! Model: '%s', TotalToken: %d, InputToken: %d, OutputToken: %d",
				usage.Model, usage.TotalToken, usage.InputToken, usage.OutputToken)
			// [AI-STATISTICS-TRACE] [onHttpStreamingBody] In-flight token usage intercepted! Model: 'Vendor3/DeepSeek-V4-Flash', TotalToken: 961, InputToken: 91, OutputToken: 870

			// Set span attributes for ARMS.
			// 捕获到 Token 数据后, 立即将其写入分布式链路追踪(ARMS / OpenTelemetry Span)
			setSpanAttribute(ArmsTotalToken, usage.TotalToken)
			setSpanAttribute(ArmsModelName, usage.Model)
			setSpanAttribute(ArmsInputToken, usage.InputToken)
			setSpanAttribute(ArmsOutputToken, usage.OutputToken)

			// Set token details to context for later use in attributes
			// 缓存 Token 细分指标(如 Prompt Cache 命中量、思考链推理 Token 量), 供下游自定义属性提取
			if len(usage.InputTokenDetails) > 0 {
				log.Infof(traceLogPrefix+"[onHttpStreamingBody] Setting InputTokenDetails in context: %+v", usage.InputTokenDetails)
				// [AI-STATISTICS-TRACE] [onHttpStreamingBody] Setting InputTokenDetails in context: map[cache_creation:0]
				ctx.SetContext(tokenusage.CtxKeyInputTokenDetails, usage.InputTokenDetails)
			}
			if len(usage.OutputTokenDetails) > 0 {
				log.Infof(traceLogPrefix+"[onHttpStreamingBody] Setting OutputTokenDetails in context: %+v", usage.OutputTokenDetails)
				// [AI-STATISTICS-TRACE] [onHttpStreamingBody] Setting OutputTokenDetails in context: map[reasoning_tokens:468]
				ctx.SetContext(tokenusage.CtxKeyOutputTokenDetails, usage.OutputTokenDetails)
			}

			// Write once
			// 提前刷盘写入日志(确保即使后续流意外断连, Token 计费数据也不丢失)
			_ = ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
			log.Infof(traceLogPrefix + "[onHttpStreamingBody] Flushed in-flight UserAttributes to access log")
		}
	}
	// If the end of the stream is reached, record metrics/logs/spans.

	// 8. 终态收敛逻辑: 流结束阶段处理(End-of-Stream Finalization)
	// 当且仅当接收到上游发出的最后一个数据块(endOfStream == true)时触发
	if endOfStream {
		responseEndTime := time.Now().UnixMilli()
		totalDuration := responseEndTime - requestStartTime
		// 计算大模型整体端到端服务耗时(Service Duration)
		ctx.SetUserAttribute(LLMServiceDuration, totalDuration)
		log.Infof(traceLogPrefix+"[onHttpStreamingBody] === END OF STREAM REACHED === Total service duration: %d ms", totalDuration)

		// Set user defined log & span attributes from streaming body.
		// Always call setAttributeBySource even if shouldBufferStreamingBody is false,
		// because token-related attributes are extracted from context (not buffered body).
		// 提取从流式 Body 中衍生出的用户自定义 Span/Log 属性(如完整的生成文本、Tool Calls)
		// 注意: 即使 shouldBufferStreamingBody 为 false, 也必须调用,
		// 因为部分 Token 属性是直接从上面的 Context 中抽取的, 无需 Body 参与
		var streamingBodyBuffer []byte
		if config.shouldBufferStreamingBody {
			streamingBodyBuffer, _ = ctx.GetContext(CtxStreamingBodyBuffer).([]byte)
			log.Infof(traceLogPrefix+"[onHttpStreamingBody] Final buffered streaming body length: %d bytes", len(streamingBodyBuffer))
		}
		setAttributeBySource(ctx, config, ResponseStreamingBody, streamingBodyBuffer)

		// Write log
		// 调试日志输出并持久化最终的用户属性到 Envoy Access Log
		debugLogAiLog(ctx)
		_ = ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
		log.Infof(traceLogPrefix + "[onHttpStreamingBody] Final access log written for stream")

		// Write metrics
		// 触发 Prometheus / Envoy 指标上报(QPS、Total Tokens、TTFT Histogram、总耗时等)
		// 指标必须且仅能在 endOfStream 提交一次, 防止计数重复膨胀
		log.Infof(traceLogPrefix + "[onHttpStreamingBody] Triggering writeMetric...")
		writeMetric(ctx, config)
	}

	// 9. 零延迟透传: 将上游 chunk 原样返回给下游客户端
	// 保证终端用户的流式打字机视觉效果不受任何代理层时延影响
	return data
}

// onHttpResponseBody 是完整 HTTP 响应体处理阶段的核心回调函数.
// 对应 Envoy Proxy-Wasm ABI 的 proxy_on_http_response_body 完整报文阶段.
// 专用于处理非流式调用(如前端传了 stream: false 的普通 JSON 响应).
// 此时网关已通过 BufferResponseBody() 将上游完整的 HTTP 报文暂存完毕, 作为 body 字节切片传入.
// 该函数在整个请求生命周期中[只会被触发执行一次].
func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte) types.Action {
	log.Infof(traceLogPrefix+"[onHttpResponseBody] >>> Received Full HTTP Response Body. Length: %d bytes, Content: %s", len(body), string(body))

	// Check if processing should be skipped
	// 1. 快速旁路守卫(Fast Bypass Guard)
	// 如果前置阶段(请求头匹配失败、白名单不通过等)标记了 SkipProcessing, 直接放行, 避免无效计算.
	if ctx.GetBoolContext(SkipProcessing, false) {
		log.Infof(traceLogPrefix + "[onHttpResponseBody] SkipProcessing is true. Bypassing non-streaming response body.")
		return types.ActionContinue
	}

	// Get requestStartTime from http context
	// 2. 提取时间基准, 计算端到端总服务耗时(LLM Service Duration)
	// 从上下文中取出在 onHttpRequestHeaders 阶段记录的绝对起始时间戳
	requestStartTime, _ := ctx.GetContext(StatisticsRequestStartTime).(int64)

	// 获取当前响应完成的毫秒级时间戳
	responseEndTime := time.Now().UnixMilli()
	serviceDuration := responseEndTime - requestStartTime
	// 计算总耗时(包括网络往返、网关处理、上游排队以及大模型全量生成的时间), 记录到可观测属性中
	ctx.SetUserAttribute(LLMServiceDuration, serviceDuration)
	log.Infof(traceLogPrefix+"[onHttpResponseBody] Non-streaming Service Duration: %d ms (start: %d, end: %d)", serviceDuration, requestStartTime, responseEndTime)

	// 3. 标记调用形态与提取全局会话 ID
	// 将响应类型标记为 "normal"(常规非流式), 与流式调用的 "stream" 形成鲜明区分, 便于监控大盘分组下钻
	ctx.SetUserAttribute(ResponseType, "normal")

	// 多厂商协议兼容提取 ChatID / RequestID:
	// 从完整 JSON 响应中提取唯一标识符, 兼容 OpenAI/DeepSeek ("id" / "response.id")、
	// Google Gemini ("responseId")、Anthropic Claude ("message.id")
	if chatID := wrapper.GetValueFromBody(body, []string{
		"id",
		"response.id",
		"responseId", // Gemini generateContent
		"message.id", // anthropic/claude messages
	}); chatID != nil {
		ctx.SetUserAttribute(ChatID, chatID.String())
		log.Infof(traceLogPrefix+"[onHttpResponseBody] Extracted ChatID: %s", chatID.String())
	} else {
		log.Infof(traceLogPrefix + "[onHttpResponseBody] ChatID not found in response body")
	}

	// Set information about this request
	// 4. 解析大模型 Token 消耗(Token Usage Extraction)
	// 在非流式响应中, 各大模型厂商规范均会在根节点返回完整的 usage JSON 结构体
	if !config.disableOpenaiUsage {
		// tokenusage.GetTokenUsage 会直接在 body 字节切片上检索 usage 字段
		if usage := tokenusage.GetTokenUsage(ctx, body); usage.TotalToken > 0 {
			log.Infof(traceLogPrefix+"[onHttpResponseBody] Token usage parsed: Model='%s', TotalToken=%d, InputToken=%d, OutputToken=%d",
				usage.Model, usage.TotalToken, usage.InputToken, usage.OutputToken)

			// Set span attributes for ARMS.
			// 将消耗数据注入分布式链路追踪(ARMS / OpenTelemetry Span 属性)
			setSpanAttribute(ArmsModelName, usage.Model)
			setSpanAttribute(ArmsInputToken, usage.InputToken)
			setSpanAttribute(ArmsOutputToken, usage.OutputToken)
			setSpanAttribute(ArmsTotalToken, usage.TotalToken)

			// Set token details to context for later use in attributes
			// 缓存 Token 细分明细(如 Prompt Cache 命中的 token 数、思考链推理 reasoning_tokens), 供自定义规则提取
			if len(usage.InputTokenDetails) > 0 {
				log.Infof(traceLogPrefix+"[onHttpResponseBody] Setting InputTokenDetails in context: %+v", usage.InputTokenDetails)
				ctx.SetContext(tokenusage.CtxKeyInputTokenDetails, usage.InputTokenDetails)
			}
			if len(usage.OutputTokenDetails) > 0 {
				log.Infof(traceLogPrefix+"[onHttpResponseBody] Setting OutputTokenDetails in context: %+v", usage.OutputTokenDetails)
				ctx.SetContext(tokenusage.CtxKeyOutputTokenDetails, usage.OutputTokenDetails)
			}
		} else {
			log.Infof(traceLogPrefix + "[onHttpResponseBody] No token usage found in response body or TotalToken == 0")
		}
	}

	// Set user defined log & span attributes.
	// 5. 提取用户自定义属性(从完整响应体中按规则提取)
	// 例如用户在配置中声明需要提取 choices[0].message.content(完整回复)或 finish_reason
	log.Infof(traceLogPrefix + "[onHttpResponseBody] Extracting user-defined attributes from ResponseBody...")
	setAttributeBySource(ctx, config, ResponseBody, body)

	// Write log
	// 6. 访问日志(Access Log)落盘与持久化
	debugLogAiLog(ctx)
	// 一键将当前请求中收集的所有 UserAttribute(模型名、Token量、耗时、ChatID等)序列化输出到网关访问日志
	_ = ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
	log.Infof(traceLogPrefix + "[onHttpResponseBody] Flushed all UserAttributes to access log with key: " + wrapper.AILogKey)

	// Write metrics
	// 7. 提交 Prometheus / Envoy 指标度量
	// 递增请求计数器、上报延迟直方图(Histogram)以及 Token 累加器
	log.Infof(traceLogPrefix + "[onHttpResponseBody] Triggering writeMetric...")
	writeMetric(ctx, config)

	// 8. 放行响应体, 指示 Envoy 将缓冲的数据完整发送给下游客户端
	log.Infof(traceLogPrefix + "[onHttpResponseBody] <<< Finished onHttpResponseBody, returning ActionContinue")
	return types.ActionContinue
}

// fetches the tracing span value from the specified source.

// setAttributeBySource 是一个通用的、数据驱动的属性提取与分发引擎.
// 它在请求的不同阶段被反复调用, 根据传入的 source 类型(请求头、请求体、流式响应体等),
// 从相应的数据载荷(body / header)中提取属性, 并格式化输出到日志、链路追踪(Trace)与度量指标(Metrics).
// ctx: 当前请求的上下文管理器
// config: 插件配置(包含所有属性的提取规则、截断长度等)
// source: 当前所处的触发源阶段(如 FixedValue, RequestHeader, RequestBody, ResponseStreamingBody, ResponseBody)
// body: 当前阶段对应的数据载荷(若是 Body/Streaming 阶段则传入 byte 切片, Header 阶段传入 nil)
func setAttributeBySource(ctx wrapper.HttpContext, config AIStatisticsConfig, source string, body []byte) {
	log.Infof(traceLogPrefix+"[setAttributeBySource] Running extraction for source: '%s', payload length: %d bytes, configured attributes: %d", source, len(body), len(config.attributes))
	// [AI-STATISTICS-TRACE] [setAttributeBySource] Running extraction for source: 'fixed_value', payload length: 0 bytes, configured attributes: 10
	// [AI-STATISTICS-TRACE] [setAttributeBySource] Running extraction for source: 'request_header', payload length: 0 bytes, configured attributes: 10
	// [AI-STATISTICS-TRACE] [setAttributeBySource] Running extraction for source: 'request_body', payload length: 334 bytes, configured attributes: 10
	// [AI-STATISTICS-TRACE] [setAttributeBySource] Running extraction for source: 'response_header', payload length: 0 bytes, configured attributes: 10
	// [AI-STATISTICS-TRACE] [setAttributeBySource] Running extraction for source: 'response_streaming_body', payload length: 134492 bytes, configured attributes: 10

	// 遍历用户在配置中声明的所有待采集属性规则(config.attributes)
	for _, attribute := range config.attributes {
		var key string
		var value interface{}
		key = attribute.Key

		// Check if this attribute should be processed for the current source
		// For built-in attributes without value_source configured, use default source matching
		// 1. 触发阶段与来源的精准匹配校验(Phase Guard)
		// 检查当前属性是否应当在当前的 source 阶段被处理.
		// 如果该属性配置了明确的 ValueSource(如 RequestBody), 但当前处于 RequestHeader 阶段, 直接跳过;
		// 若为未显式配置来源的内置属性, 则依赖 shouldProcessBuiltinAttribute 判断是否属于当前阶段的默认来源.
		if !shouldProcessBuiltinAttribute(key, attribute.ValueSource, source) {
			continue
		}
		log.Infof(traceLogPrefix+"[setAttributeBySource] Target attribute matched for source '%s': Key='%s', ValueSource='%s', ConfiguredPath='%s', Rule='%s'",
			source, key, attribute.ValueSource, attribute.Value, attribute.Rule)
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Target attribute matched for source 'request_body': Key='messages', ValueSource='request_body', ConfiguredPath='messages', Rule=''
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Target attribute matched for source 'request_body': Key='question', ValueSource='', ConfiguredPath='', Rule=''
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Target attribute matched for source 'request_body': Key='system', ValueSource='', ConfiguredPath='', Rule=''
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Target attribute matched for source 'response_streaming_body': Key='answer', ValueSource='', ConfiguredPath='', Rule='append'
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Target attribute matched for source 'response_streaming_body': Key='reasoning', ValueSource='', ConfiguredPath='', Rule='append'
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Target attribute matched for source 'response_streaming_body': Key='tool_calls', ValueSource='', ConfiguredPath='', Rule=''
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Target attribute matched for source 'response_streaming_body': Key='reasoning_tokens', ValueSource='', ConfiguredPath='', Rule=''
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Target attribute matched for source 'response_streaming_body': Key='cached_tokens', ValueSource='', ConfiguredPath='', Rule=''
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Target attribute matched for source 'response_streaming_body': Key='input_token_details', ValueSource='', ConfiguredPath='', Rule=''
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Target attribute matched for source 'response_streaming_body': Key='output_token_details', ValueSource='', ConfiguredPath='', Rule=''

		// If value is configured, try to extract using the configured path
		// 2. 第一级提取: 基于显式配置的 Path/Key 进行提取
		if attribute.Value != "" {
			switch source {
			case FixedValue:
				// 静态固定值(如固定环境标识 env: "prod")
				value = attribute.Value
				log.Infof(traceLogPrefix+"[setAttributeBySource] Key '%s' extracted from FixedValue: '%+v'", key, value)
			case RequestHeader:
				// 从客户端 HTTP 请求头提取(如 Authorization、X-User-Id)
				value, _ = proxywasm.GetHttpRequestHeader(attribute.Value)
				log.Infof(traceLogPrefix+"[setAttributeBySource] Key '%s' extracted from RequestHeader ('%s'): '%+v'", key, attribute.Value, value)
			case RequestBody:
				// 使用 GJSON 高性能 JSONPath 语法, 直接从请求体 JSON 检索目标字段
				value = gjson.GetBytes(body, attribute.Value).Value()
				log.Infof(traceLogPrefix+"[setAttributeBySource] Key '%s' extracted from RequestBody (path '%s'): '%+v'", key, attribute.Value, value)
				// [AI-STATISTICS-TRACE] [setAttributeBySource] Key 'messages' extracted from RequestBody (path 'messages'): '[map[content:😀 你可以提供哪些服务？ role:user]]'

			case ResponseHeader:
				// 从上游返回的 HTTP 响应头提取(如 X-RateLimit-Reset)
				value, _ = proxywasm.GetHttpResponseHeader(attribute.Value)
				log.Infof(traceLogPrefix+"[setAttributeBySource] Key '%s' extracted from ResponseHeader ('%s'): '%+v'", key, attribute.Value, value)
			case ResponseStreamingBody:
				// 专用于流式 SSE 场景: 根据合并规则(First/Replace/Append)提取流式分块中的目标字段并拼接聚合
				value = extractStreamingBodyByJsonPath(body, attribute.Value, attribute.Rule)
				log.Infof(traceLogPrefix+"[setAttributeBySource] Key '%s' extracted from ResponseStreamingBody (path '%s', rule '%s'): '%+v'", key, attribute.Value, attribute.Rule, value)
			case ResponseBody:
				// 从完整非流式响应体 JSON 检索目标字段(如 choices.0.message.content)
				value = gjson.GetBytes(body, attribute.Value).Value()
				log.Infof(traceLogPrefix+"[setAttributeBySource] Key '%s' extracted from ResponseBody (path '%s'): '%+v'", key, attribute.Value, value)
			default:
				log.Infof(traceLogPrefix+"[setAttributeBySource] Key '%s' encountered unknown source: '%s'", key, source)
			}
		}

		// Handle built-in attributes: use fallback if value is empty or not configured
		// 3. 第二级提取(兜底降级): 内置属性的自适应提取(Fallback to Built-in Logic)
		// 如果用户未显式配置 Value 路径, 或者显式配置的路径未提取到有效值,
		// 且当前 Key 属于网关内置已知属性(如 model、tokens 等),
		// 则触发插件内置的智能抽取机制(自动按 OpenAI/Claude/Gemini 等协议硬编码规范提取)
		if (value == nil || value == "") && isBuiltinAttribute(key) {
			log.Infof(traceLogPrefix+"[setAttributeBySource] Value empty for built-in attribute '%s'. Triggering getBuiltinAttributeFallback...", key)
			// [AI-STATISTICS-TRACE] [setAttributeBySource] Value empty for built-in attribute 'question'. Triggering getBuiltinAttributeFallback...
			// [AI-STATISTICS-TRACE] [setAttributeBySource] Value empty for built-in attribute 'system'. Triggering getBuiltinAttributeFallback...
			// [AI-STATISTICS-TRACE] [setAttributeBySource] Value empty for built-in attribute 'answer'. Triggering getBuiltinAttributeFallback...
			// [AI-STATISTICS-TRACE] [setAttributeBySource] Value empty for built-in attribute 'reasoning'. Triggering getBuiltinAttributeFallback...
			// [AI-STATISTICS-TRACE] [setAttributeBySource] Value empty for built-in attribute 'tool_calls'. Triggering getBuiltinAttributeFallback...
			// [AI-STATISTICS-TRACE] [setAttributeBySource] Value empty for built-in attribute 'reasoning_tokens'. Triggering getBuiltinAttributeFallback...
			// [AI-STATISTICS-TRACE] [setAttributeBySource] Value empty for built-in attribute 'cached_tokens'. Triggering getBuiltinAttributeFallback...
			// [AI-STATISTICS-TRACE] [setAttributeBySource] Value empty for built-in attribute 'input_token_details'. Triggering getBuiltinAttributeFallback...
			// [AI-STATISTICS-TRACE] [setAttributeBySource] Value empty for built-in attribute 'output_token_details'. Triggering getBuiltinAttributeFallback...

			value = getBuiltinAttributeFallback(ctx, config, key, source, body, attribute.Rule)
			if value != nil && value != "" {
				log.Debugf("[attribute] Used built-in extraction for %s: %+v", key, value)
				log.Infof(traceLogPrefix+"[setAttributeBySource] Built-in extraction succeeded for Key '%s': '%+v'", key, value)
				// [AI-STATISTICS-TRACE] [setAttributeBySource] Built-in extraction succeeded for Key 'question': '😀 你可以提供哪些服务？'
				// [AI-STATISTICS-TRACE] [setAttributeBySource] Built-in extraction succeeded for Key 'answer': '你好呀
				// [AI-STATISTICS-TRACE] [setAttributeBySource] Built-in extraction succeeded for Key 'reasoning': '我们需要理解用户的问题
				// [AI-STATISTICS-TRACE] [setAttributeBySource] Built-in extraction succeeded for Key 'reasoning_tokens': '468'
				// [AI-STATISTICS-TRACE] [setAttributeBySource] Built-in extraction succeeded for Key 'input_token_details': 'map[cache_creation:0]'
				// [AI-STATISTICS-TRACE] [setAttributeBySource] Built-in extraction succeeded for Key 'output_token_details': 'map[reasoning_tokens:468]'

			} else {
				log.Infof(traceLogPrefix+"[setAttributeBySource] Built-in extraction returned nil/empty for Key '%s'", key)
				// [AI-STATISTICS-TRACE] [setAttributeBySource] Built-in extraction returned nil/empty for Key 'system'
				// [AI-STATISTICS-TRACE] [setAttributeBySource] Built-in extraction returned nil/empty for Key 'tool_calls'
				// [AI-STATISTICS-TRACE] [setAttributeBySource] Built-in extraction returned nil/empty for Key 'cached_tokens'

			}
		}

		// 4. 第三级提取(终极兜底): 填充预设的默认值
		// 如果前面所有提取手段均告失效, 且用户配置了 DefaultValue, 则填入默认值
		if (value == nil || value == "") && attribute.DefaultValue != "" {
			value = attribute.DefaultValue
			log.Infof(traceLogPrefix+"[setAttributeBySource] Applying DefaultValue for Key '%s': '%s'", key, attribute.DefaultValue)
		}

		// Format value for logging/span
		// 5. 格式化与内存安全截断保护(Format & Truncation Guard)
		var formattedValue interface{}
		switch v := value.(type) {
		case map[string]int64:
			// For token details maps, convert to JSON string
			// 针对复杂的嵌套对象(如 Token 明细 details: {"cached_tokens": 100, "reasoning_tokens": 50}),
			// 序列化为规范的 JSON 文本字符串, 以便于后续打日志或打 Span
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				log.Warnf("failed to marshal token details: %v", err)
				log.Infof(traceLogPrefix+"[setAttributeBySource] Failed to marshal map[string]int64 for Key '%s': %v", key, err)
				formattedValue = fmt.Sprint(v)
			} else {
				formattedValue = string(jsonBytes)
			}
		default:
			formattedValue = value
			valStr := fmt.Sprint(value)
			// 核心内存与带宽防护: 首尾保留截断(Head-Tail Truncation)
			// 当文本长度超过配置的 valueLengthLimit 时, 严禁全量输出(防止 Wasm 堆膨胀或撑爆日志采集器).
			// 算法保留前半部分(看 Prompt/开头)和后半部分(看结论/回复尾部), 中间插入 " [truncated] ".
			if len(valStr) > config.valueLengthLimit {
				half := config.valueLengthLimit / 2
				formattedValue = valStr[:half] + " [truncated] " + valStr[len(valStr)-half:]
				log.Infof(traceLogPrefix+"[setAttributeBySource] Value for Key '%s' exceeded valueLengthLimit (%d > %d). Truncated preserving head & tail.", key, len(valStr), config.valueLengthLimit)
			}
		}

		log.Debugf("[attribute] source type: %s, key: %s, value: %+v", source, key, formattedValue)
		log.Infof(traceLogPrefix+"[setAttributeBySource] Final formatted value for Key '%s': %+v", key, formattedValue)
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Final formatted value for Key 'messages': [map[content:😀 你可以提供哪些服务？ role:user]]
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Final formatted value for Key 'question': 😀 你可以提供哪些服务？
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Final formatted value for Key 'system': <nil>
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Final formatted value for Key 'answer': 你好呀
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Final formatted value for Key 'reasoning': 我们需要理解用户的问题
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Final formatted value for Key 'tool_calls': <nil>
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Final formatted value for Key 'reasoning_tokens': 468
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Final formatted value for Key 'cached_tokens': <nil>
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Final formatted value for Key 'input_token_details': {"cache_creation":0}
		// [AI-STATISTICS-TRACE] [setAttributeBySource] Final formatted value for Key 'output_token_details': {"reasoning_tokens":468}

		// 6. 分发出口 A: 沉淀至访问日志(Access Log)
		if attribute.ApplyToLog {
			if attribute.AsSeparateLogField {
				// 分支 1: 作为 Envoy 顶层独立日志字段暴露
				// 通过 SetProperty 写入 FilterState, 使 Envoy 的 access_log_format 可以通过
				// %FILTER_STATE(key)% 独立引用此字段, 而无需嵌套在整体的 JSON 字典中
				var marshalledJsonStr string
				if _, ok := value.(map[string]int64); ok {
					// Already marshaled in formattedValue
					marshalledJsonStr = fmt.Sprint(formattedValue)
				} else {
					marshalledJsonStr = wrapper.MarshalStr(fmt.Sprint(formattedValue))
				}
				log.Infof(traceLogPrefix+"[setAttributeBySource] Saving Key '%s' to FilterState as separate log field: %s", key, marshalledJsonStr)
				if err := proxywasm.SetProperty([]string{key}, []byte(marshalledJsonStr)); err != nil {
					log.Warnf("failed to set %s in filter state, raw is %s, err is %v", key, marshalledJsonStr, err)
					log.Infof(traceLogPrefix+"[setAttributeBySource] SetProperty failed for separate log field '%s': %v", key, err)
				}
			} else {
				// 分支 2: 作为通用的自定义用户属性
				// 聚合存入 ctx.userAttribute, 后续随 wrapper.AILogKey 一并以统一的 JSON 块输出
				ctx.SetUserAttribute(key, formattedValue)
				log.Infof(traceLogPrefix+"[setAttributeBySource] Attached Key '%s' to UserAttribute", key)
			}
		}
		// for metrics

		// 7. 分发出口 B: 同步至度量指标缓存(Metrics Sync)
		// 如果当前提取的字段属于核心 Token/模型指标, 立即存入内部 Context(ctx.userContext),
		// 确保在请求结束调用 writeMetric 时, 能够无损拿到这些值更新 Prometheus 计数器
		if key == tokenusage.CtxKeyModel || key == tokenusage.CtxKeyInputToken || key == tokenusage.CtxKeyOutputToken || key == tokenusage.CtxKeyTotalToken {
			ctx.SetContext(key, value)
			log.Infof(traceLogPrefix+"[setAttributeBySource] Synced metric context key '%s' = '%+v'", key, value)
		}

		// 8. 分发出口 C: 注入分布式链路追踪(Distributed Tracing Span)
		if attribute.ApplyToSpan {
			spanKey := key
			// 支持别名重映射(例如内部叫 input_token, 但在 Trace 中按照 OpenTelemetry 规范映射为 gen_ai.usage.prompt_tokens)
			if attribute.TraceSpanKey != "" {
				spanKey = attribute.TraceSpanKey
			}
			log.Infof(traceLogPrefix+"[setAttributeBySource] Injecting Span Tag: '%s' = '%+v'", spanKey, value)
			// 通过之前解析过的 setSpanAttribute 写入 FilterState 的 trace_span_tag.* 前缀中
			setSpanAttribute(spanKey, value)
		}
	}
}

// isBuiltinAttribute checks if the given key is a built-in attribute
func isBuiltinAttribute(key string) bool {
	return key == BuiltinQuestionKey || key == BuiltinAnswerKey || key == BuiltinToolCallsKey || key == BuiltinReasoningKey || key == BuiltinSystemKey ||
		key == BuiltinReasoningTokens || key == BuiltinCachedTokens ||
		key == BuiltinInputTokenDetails || key == BuiltinOutputTokenDetails
}

// needsBodyBuffering checks if a built-in attribute needs body buffering
// Token-related attributes are extracted from context (set by tokenusage.GetTokenUsage),
// so they don't require buffering the response body.
func needsBodyBuffering(key string) bool {
	return key == BuiltinAnswerKey || key == BuiltinToolCallsKey || key == BuiltinReasoningKey
}

// getBuiltinAttributeDefaultSources returns the default value_source(s) for a built-in attribute
// Returns nil if the key is not a built-in attribute
// Note: Token-related attributes are extracted from context (set by tokenusage.GetTokenUsage),
// so they don't require body buffering even though they're processed during response phase.
func getBuiltinAttributeDefaultSources(key string) []string {
	switch key {
	case BuiltinQuestionKey, BuiltinSystemKey:
		return []string{RequestBody}
	case BuiltinAnswerKey, BuiltinToolCallsKey, BuiltinReasoningKey:
		return []string{ResponseStreamingBody, ResponseBody}
	case BuiltinReasoningTokens, BuiltinCachedTokens, BuiltinInputTokenDetails, BuiltinOutputTokenDetails:
		// Token details are extracted from context (set by tokenusage.GetTokenUsage),
		// not from body parsing. We use ResponseStreamingBody/ResponseBody to indicate
		// they should be processed during response phase, but they don't require body buffering.
		return []string{ResponseStreamingBody, ResponseBody}
	default:
		return nil
	}
}

// shouldProcessBuiltinAttribute checks if a built-in attribute should be processed for the given source
func shouldProcessBuiltinAttribute(key, configuredSource, currentSource string) bool {
	// If value_source is configured, use exact match
	if configuredSource != "" {
		return configuredSource == currentSource
	}
	// If value_source is not configured and it's a built-in attribute, check default sources
	defaultSources := getBuiltinAttributeDefaultSources(key)
	for _, src := range defaultSources {
		if src == currentSource {
			return true
		}
	}
	return false
}

// getBuiltinAttributeFallback provides protocol compatibility fallback for built-in attributes
func getBuiltinAttributeFallback(ctx wrapper.HttpContext, config AIStatisticsConfig, key, source string, body []byte, rule string) interface{} {
	log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Resolving fallback for Key: '%s', Source: '%s', Rule: '%s'", key, source, rule)
	// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Resolving fallback for Key: 'question', Source: 'request_body', Rule: ''
	// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Resolving fallback for Key: 'system', Source: 'request_body', Rule: ''
	// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Resolving fallback for Key: 'answer', Source: 'response_streaming_body', Rule: 'append'
	// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Resolving fallback for Key: 'reasoning', Source: 'response_streaming_body', Rule: 'append'
	// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Resolving fallback for Key: 'tool_calls', Source: 'response_streaming_body', Rule: ''
	// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Resolving fallback for Key: 'reasoning_tokens', Source: 'response_streaming_body', Rule: ''
	// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Resolving fallback for Key: 'cached_tokens', Source: 'response_streaming_body', Rule: ''
	// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Resolving fallback for Key: 'input_token_details', Source: 'response_streaming_body', Rule: ''
	// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Resolving fallback for Key: 'output_token_details', Source: 'response_streaming_body', Rule: ''

	switch key {
	case BuiltinQuestionKey:
		if source == RequestBody {
			// Try OpenAI/Claude format (both use same messages structure)
			if value := gjson.GetBytes(body, QuestionPathOpenAI).Value(); value != nil && value != "" {
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found 'question' using QuestionPathOpenAI: %+v", value)
				// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Found 'question' using QuestionPathOpenAI: 😀 你可以提供哪些服务？
				return value
			}
		}
	case BuiltinSystemKey:
		if source == RequestBody {
			// Try Claude /v1/messages format (system is a top-level field)
			if value := gjson.GetBytes(body, SystemPathClaude).Value(); value != nil && value != "" {
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found 'system' using SystemPathClaude: %+v", value)
				return value
			}
		}
	case BuiltinAnswerKey:
		if source == ResponseStreamingBody {
			// Try OpenAI format first
			if value := extractStreamingBodyByJsonPath(body, AnswerPathOpenAIStreaming, rule); value != nil && value != "" {
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found streaming 'answer' using AnswerPathOpenAIStreaming: %+v", value)
				// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Found streaming 'answer' using AnswerPathOpenAIStreaming: 你好呀
				return value
			}
			// Try Claude format
			if value := extractStreamingBodyByJsonPath(body, AnswerPathClaudeStreaming, rule); value != nil && value != "" {
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found streaming 'answer' using AnswerPathClaudeStreaming: %+v", value)
				return value
			}
		} else if source == ResponseBody {
			// Try OpenAI format first
			if value := gjson.GetBytes(body, AnswerPathOpenAINonStreaming).Value(); value != nil && value != "" {
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found non-streaming 'answer' using AnswerPathOpenAINonStreaming: %+v", value)
				return value
			}
			// Try Claude format
			if value := gjson.GetBytes(body, AnswerPathClaudeNonStreaming).Value(); value != nil && value != "" {
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found non-streaming 'answer' using AnswerPathClaudeNonStreaming: %+v", value)
				return value
			}
		}
	case BuiltinToolCallsKey:
		if source == ResponseStreamingBody {
			// Get or create buffer from context
			var buffer *StreamingToolCallsBuffer
			if existingBuffer, ok := ctx.GetContext(CtxStreamingToolCallsBuffer).(*StreamingToolCallsBuffer); ok {
				buffer = existingBuffer
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Reusing existing StreamingToolCallsBuffer with %d calls", len(buffer.ToolCalls))
			} else {
				log.Infof(traceLogPrefix + "[getBuiltinAttributeFallback] Creating fresh StreamingToolCallsBuffer")
				// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Creating fresh StreamingToolCallsBuffer
			}
			// Try OpenAI format first
			buffer = extractStreamingToolCalls(body, buffer)
			// Also try Claude format (both formats can be checked)
			buffer = extractClaudeStreamingToolCalls(body, buffer)
			ctx.SetContext(CtxStreamingToolCallsBuffer, buffer)

			// Also set tool_calls to user attributes so they appear in ai_log
			toolCalls := getToolCallsFromBuffer(buffer)
			if len(toolCalls) > 0 {
				ctx.SetUserAttribute(BuiltinToolCallsKey, toolCalls)
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Set %d tool_calls to UserAttribute", len(toolCalls))
				return toolCalls
			}
		} else if source == ResponseBody {
			if value := gjson.GetBytes(body, ToolCallsPathNonStreaming).Value(); value != nil {
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found non-streaming tool_calls: %+v", value)
				return value
			}
		}
	case BuiltinReasoningKey:
		if source == ResponseStreamingBody {
			if value := extractStreamingBodyByJsonPath(body, ReasoningPathStreaming, RuleAppend); value != nil && value != "" {
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found streaming 'reasoning': %+v", value)
				// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Found streaming 'reasoning': 我们需要理解用户的问题
				return value
			}
		} else if source == ResponseBody {
			if value := gjson.GetBytes(body, ReasoningPathNonStreaming).Value(); value != nil && value != "" {
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found non-streaming 'reasoning': %+v", value)
				return value
			}
		}
	case BuiltinReasoningTokens:
		// Extract reasoning_tokens from output_token_details (only available after response)
		if source == ResponseBody || source == ResponseStreamingBody {
			if outputTokenDetails, ok := ctx.GetContext(tokenusage.CtxKeyOutputTokenDetails).(map[string]int64); ok {
				if reasoningTokens, exists := outputTokenDetails["reasoning_tokens"]; exists {
					log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found 'reasoning_tokens' from outputTokenDetails: %d", reasoningTokens)
					// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Found 'reasoning_tokens' from outputTokenDetails: 468
					return reasoningTokens
				}
			}
		}
	case BuiltinCachedTokens:
		// Extract cached_tokens from input_token_details (only available after response)
		if source == ResponseBody || source == ResponseStreamingBody {
			if inputTokenDetails, ok := ctx.GetContext(tokenusage.CtxKeyInputTokenDetails).(map[string]int64); ok {
				if cachedTokens, exists := inputTokenDetails["cached_tokens"]; exists {
					log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found 'cached_tokens' from inputTokenDetails: %d", cachedTokens)
					return cachedTokens
				}
			}
		}
	case BuiltinInputTokenDetails:
		// Return the entire input_token_details map (only available after response)
		if source == ResponseBody || source == ResponseStreamingBody {
			if inputTokenDetails, ok := ctx.GetContext(tokenusage.CtxKeyInputTokenDetails).(map[string]int64); ok {
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found 'input_token_details': %+v", inputTokenDetails)
				// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Found 'input_token_details': map[cache_creation:0]
				return inputTokenDetails
			}
		}
	case BuiltinOutputTokenDetails:
		// Return the entire output_token_details map (only available after response)
		if source == ResponseBody || source == ResponseStreamingBody {
			if outputTokenDetails, ok := ctx.GetContext(tokenusage.CtxKeyOutputTokenDetails).(map[string]int64); ok {
				log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Found 'output_token_details': %+v", outputTokenDetails)
				// [AI-STATISTICS-TRACE] [getBuiltinAttributeFallback] Found 'output_token_details': map[reasoning_tokens:468]
				return outputTokenDetails
			}
		}
	}
	log.Infof(traceLogPrefix+"[getBuiltinAttributeFallback] Fallback extraction yielded nothing for Key: '%s', Source: '%s'", key, source)
	return nil
}

func extractStreamingBodyByJsonPath(data []byte, jsonPath string, rule string) interface{} {
	chunks := bytes.Split(bytes.TrimSpace(wrapper.UnifySSEChunk(data)), []byte("\n\n"))
	var value interface{}
	log.Infof(traceLogPrefix+"[extractStreamingBodyByJsonPath] Parsing %d SSE chunks for jsonPath: '%s', rule: '%s'", len(chunks), jsonPath, rule)
	if rule == RuleFirst {
		for _, chunk := range chunks {
			jsonObj := gjson.GetBytes(chunk, jsonPath)
			if jsonObj.Exists() {
				value = jsonObj.Value()
				log.Infof(traceLogPrefix+"[extractStreamingBodyByJsonPath] RuleFirst matched value: %+v", value)
				break
			}
		}
	} else if rule == RuleReplace {
		for _, chunk := range chunks {
			jsonObj := gjson.GetBytes(chunk, jsonPath)
			if jsonObj.Exists() {
				value = jsonObj.Value()
			}
		}
		log.Infof(traceLogPrefix+"[extractStreamingBodyByJsonPath] RuleReplace final replaced value: %+v", value)
	} else if rule == RuleAppend {
		// extract llm response
		var strValue string
		for _, chunk := range chunks {
			jsonObj := gjson.GetBytes(chunk, jsonPath)
			if jsonObj.Exists() {
				strValue += jsonObj.String()
			}
		}
		value = strValue
		log.Infof(traceLogPrefix+"[extractStreamingBodyByJsonPath] RuleAppend accumulated string length: %d", len(strValue))
	} else {
		log.Errorf("unsupported rule type: %s", rule)
		log.Infof(traceLogPrefix+"[extractStreamingBodyByJsonPath] Error: unsupported rule type '%s'", rule)
	}
	return value
}

// shouldLogDebug returns true if the log level is debug or trace
func shouldLogDebug() bool {
	value, err := proxywasm.CallForeignFunction("get_log_level", nil)
	if err != nil {
		// If we can't get log level, default to not logging debug info
		return false
	}
	if len(value) < 4 {
		// Invalid log level value length
		return false
	}
	envoyLogLevel := binary.LittleEndian.Uint32(value[:4])
	return envoyLogLevel == LogLevelTrace || envoyLogLevel == LogLevelDebug
}

// debugLogAiLog logs the current user attributes that will be written to ai_log
func debugLogAiLog(ctx wrapper.HttpContext) {
	// Only log in debug/trace mode
	if !shouldLogDebug() {
		return
	}

	// Get all user attributes as a map
	userAttrs := make(map[string]interface{})

	// Try to reconstruct from GetUserAttribute (note: this is best-effort)
	// The actual attributes are stored internally, we log what we know
	if question := ctx.GetUserAttribute("question"); question != nil {
		userAttrs["question"] = question
	}
	if system := ctx.GetUserAttribute("system"); system != nil {
		userAttrs["system"] = system
	}
	if answer := ctx.GetUserAttribute("answer"); answer != nil {
		userAttrs["answer"] = answer
	}
	if reasoning := ctx.GetUserAttribute("reasoning"); reasoning != nil {
		userAttrs["reasoning"] = reasoning
	}
	if toolCalls := ctx.GetUserAttribute("tool_calls"); toolCalls != nil {
		userAttrs["tool_calls"] = toolCalls
	}
	if messages := ctx.GetUserAttribute("messages"); messages != nil {
		userAttrs["messages"] = messages
	}
	if sessionId := ctx.GetUserAttribute("session_id"); sessionId != nil {
		userAttrs["session_id"] = sessionId
	}
	if model := ctx.GetUserAttribute("model"); model != nil {
		userAttrs["model"] = model
	}
	if inputToken := ctx.GetUserAttribute("input_token"); inputToken != nil {
		userAttrs["input_token"] = inputToken
	}
	if outputToken := ctx.GetUserAttribute("output_token"); outputToken != nil {
		userAttrs["output_token"] = outputToken
	}
	if totalToken := ctx.GetUserAttribute("total_token"); totalToken != nil {
		userAttrs["total_token"] = totalToken
	}
	if chatId := ctx.GetUserAttribute("chat_id"); chatId != nil {
		userAttrs["chat_id"] = chatId
	}
	if responseType := ctx.GetUserAttribute("response_type"); responseType != nil {
		userAttrs["response_type"] = responseType
	}
	if llmFirstTokenDuration := ctx.GetUserAttribute("llm_first_token_duration"); llmFirstTokenDuration != nil {
		userAttrs["llm_first_token_duration"] = llmFirstTokenDuration
	}
	if llmServiceDuration := ctx.GetUserAttribute("llm_service_duration"); llmServiceDuration != nil {
		userAttrs["llm_service_duration"] = llmServiceDuration
	}
	if reasoningTokens := ctx.GetUserAttribute("reasoning_tokens"); reasoningTokens != nil {
		userAttrs["reasoning_tokens"] = reasoningTokens
	}
	if cachedTokens := ctx.GetUserAttribute("cached_tokens"); cachedTokens != nil {
		userAttrs["cached_tokens"] = cachedTokens
	}
	if inputTokenDetails := ctx.GetUserAttribute("input_token_details"); inputTokenDetails != nil {
		userAttrs["input_token_details"] = inputTokenDetails
	}
	if outputTokenDetails := ctx.GetUserAttribute("output_token_details"); outputTokenDetails != nil {
		userAttrs["output_token_details"] = outputTokenDetails
	}

	// Log the attributes as JSON
	logJson, _ := json.Marshal(userAttrs)
	log.Debugf("[ai_log] attributes to be written: %s", string(logJson))
}

// Set the tracing span with value.
// setSpanAttribute 将自定义的键值对以 Span Tag(链路标签/属性)的形式注入到当前请求的分布式追踪 Span 中.
// 适用于将 AI 模型的关键指标(如 model、tokens、TTFT、rag_docs、session_id 等)沉淀到链路拓扑中.
// key: Span 属性名(如 "llm_model"、"total_tokens")
// value: 属性值, 支持 string、int、int64、float 等任意可转换为文本的标量类型
func setSpanAttribute(key string, value interface{}) {
	log.Infof(traceLogPrefix+"[setSpanAttribute] Attempting to set Span Attribute: Key='%s', Value='%+v'", key, value)
	// [AI-STATISTICS-TRACE] [setSpanAttribute] Attempting to set Span Attribute: Key='gen_ai.span.kind', Value='LLM'

	// 1. 过滤空值属性(空值守卫与防噪)
	// 在链路追踪规范中, 空标签没有观测价值, 且部分 APM 后端对空 Tag 会告警或浪费索引存储
	if value != "" {
		// 2. 拼接 Higress 规范的 FilterState 属性前缀
		// wrapper.TraceSpanTagPrefix 的常量值为 "trace_span_tag."
		// Envoy 的 Tracing Filter 会在请求结束时扫描所有带有此特定前缀的 FilterState,
		// 并自动剥离前缀, 将其转换为真正的 Trace Span Tag(例如将 "trace_span_tag.model" 转为 Span 的 "model" 属性)
		traceSpanTag := wrapper.TraceSpanTagPrefix + key

		// 3. 跨 Wasm 沙箱 HostCall: 写入 Envoy FilterState
		// fmt.Sprint(value): 将任意 Go 类型的 value(如数字 1500、布尔值)泛化为标准的字符串表示
		// []byte(...): Proxy-Wasm ABI 只接受底层连续的字节切片指针(*char, size_t)
		// proxywasm.SetProperty: 触发 proxy_set_property 系统调用, 将属性下沉到 Envoy C++ 宿主侧的请求上下文中
		if e := proxywasm.SetProperty([]string{traceSpanTag}, []byte(fmt.Sprint(value))); e != nil {
			// 4. 容灾与告警: 如果宿主调用失败(如 Stream 已关闭或 Key 冲突), 记录 Warn 日志
			// 遵循"可观测性失败不得阻断主业务流量"的防御性原则, 仅记日志, 不向上抛出错误
			log.Warnf("failed to set %s in filter state: %v", traceSpanTag, e)
			log.Infof(traceLogPrefix+"[setSpanAttribute] Failed to set %s in filter state: %v", traceSpanTag, e)
		} else {
			log.Infof(traceLogPrefix+"[setSpanAttribute] Successfully set %s in filter state", traceSpanTag)
			// [AI-STATISTICS-TRACE] [setSpanAttribute] Successfully set trace_span_tag.gen_ai.span.kind in filter state
		}
	} else {
		// 5. 空值调试日志: 以 Debug 级别静默记录
		// 避免在生产环境高并发请求下因为大量空属性刷屏产生日志污染
		log.Debugf("failed to write span attribute [%s], because it's value is empty", key)
		log.Infof(traceLogPrefix+"[setSpanAttribute] Skipped writing span attribute [%s] because value is empty", key)
	}
}

func writeMetric(ctx wrapper.HttpContext, config AIStatisticsConfig) {
	log.Infof(traceLogPrefix + "[writeMetric] >>> Starting writeMetric...")
	// Generate usage metrics
	var ok bool
	var route, cluster, model string
	consumer := ctx.GetStringContext(ConsumerKey, "none")
	route, ok = ctx.GetContext(RouteName).(string)
	if !ok {
		log.Info("RouteName type assert failed, skip metric record")
		log.Infof(traceLogPrefix + "[writeMetric] RouteName type assertion failed, aborting writeMetric")
		return
	}
	cluster, ok = ctx.GetContext(ClusterName).(string)
	if !ok {
		log.Info("ClusterName type assert failed, skip metric record")
		log.Infof(traceLogPrefix + "[writeMetric] ClusterName type assertion failed, aborting writeMetric")
		return
	}

	log.Infof(traceLogPrefix+"[writeMetric] Target identifiers - Route: '%s', Cluster: '%s', Consumer: '%s', disableOpenaiUsage: %v",
		route, cluster, consumer, config.disableOpenaiUsage)

	if config.disableOpenaiUsage {
		log.Infof(traceLogPrefix + "[writeMetric] disableOpenaiUsage is true. Skipping token usage metrics.")
		return
	}

	if ctx.GetUserAttribute(tokenusage.CtxKeyModel) == nil || ctx.GetUserAttribute(tokenusage.CtxKeyInputToken) == nil || ctx.GetUserAttribute(tokenusage.CtxKeyOutputToken) == nil || ctx.GetUserAttribute(tokenusage.CtxKeyTotalToken) == nil {
		log.Info("get usage information failed, skip metric record")
		log.Infof(traceLogPrefix+"[writeMetric] Missing one of core usage attributes in UserAttribute. Model=%v, InputToken=%v, OutputToken=%v, TotalToken=%v",
			ctx.GetUserAttribute(tokenusage.CtxKeyModel), ctx.GetUserAttribute(tokenusage.CtxKeyInputToken), ctx.GetUserAttribute(tokenusage.CtxKeyOutputToken), ctx.GetUserAttribute(tokenusage.CtxKeyTotalToken))
		return
	}
	model, ok = ctx.GetUserAttribute(tokenusage.CtxKeyModel).(string)
	if !ok {
		log.Info("Model type assert failed, skip metric record")
		log.Infof(traceLogPrefix+"[writeMetric] Model assertion to string failed, value was: %+v", ctx.GetUserAttribute(tokenusage.CtxKeyModel))
		return
	}
	log.Infof(traceLogPrefix+"[writeMetric] Metric Model: '%s'", model)

	if inputToken, ok := convertToUInt(ctx.GetUserAttribute(tokenusage.CtxKeyInputToken)); ok {
		metricName := generateMetricName(route, cluster, model, consumer, tokenusage.CtxKeyInputToken)
		log.Infof(traceLogPrefix+"[writeMetric] Incrementing InputToken metric: '%s' by %d", metricName, inputToken)
		config.incrementCounter(metricName, inputToken)
	} else {
		log.Info("InputToken type assert failed, skip metric record")
		log.Infof(traceLogPrefix+"[writeMetric] InputToken convertToUInt failed for value: %+v", ctx.GetUserAttribute(tokenusage.CtxKeyInputToken))
	}

	if outputToken, ok := convertToUInt(ctx.GetUserAttribute(tokenusage.CtxKeyOutputToken)); ok {
		metricName := generateMetricName(route, cluster, model, consumer, tokenusage.CtxKeyOutputToken)
		log.Infof(traceLogPrefix+"[writeMetric] Incrementing OutputToken metric: '%s' by %d", metricName, outputToken)
		config.incrementCounter(metricName, outputToken)
	} else {
		log.Info("OutputToken type assert failed, skip metric record")
		log.Infof(traceLogPrefix+"[writeMetric] OutputToken convertToUInt failed for value: %+v", ctx.GetUserAttribute(tokenusage.CtxKeyOutputToken))
	}

	if totalToken, ok := convertToUInt(ctx.GetUserAttribute(tokenusage.CtxKeyTotalToken)); ok {
		metricName := generateMetricName(route, cluster, model, consumer, tokenusage.CtxKeyTotalToken)
		log.Infof(traceLogPrefix+"[writeMetric] Incrementing TotalToken metric: '%s' by %d", metricName, totalToken)
		config.incrementCounter(metricName, totalToken)
	} else {
		log.Info("TotalToken type assert failed, skip metric record")
		log.Infof(traceLogPrefix+"[writeMetric] TotalToken convertToUInt failed for value: %+v", ctx.GetUserAttribute(tokenusage.CtxKeyTotalToken))
	}

	// Generate duration metrics
	var llmFirstTokenDuration, llmServiceDuration uint64
	// Is stream response
	if ctx.GetUserAttribute(LLMFirstTokenDuration) != nil {
		llmFirstTokenDuration, ok = convertToUInt(ctx.GetUserAttribute(LLMFirstTokenDuration))
		if !ok {
			log.Info("LLMFirstTokenDuration type assert failed")
			log.Infof(traceLogPrefix+"[writeMetric] LLMFirstTokenDuration convertToUInt failed for value: %+v", ctx.GetUserAttribute(LLMFirstTokenDuration))
			return
		}
		metricNameDuration := generateMetricName(route, cluster, model, consumer, LLMFirstTokenDuration)
		metricNameCount := generateMetricName(route, cluster, model, consumer, LLMStreamDurationCount)
		log.Infof(traceLogPrefix+"[writeMetric] Incrementing TTFT metric '%s' by %d ms, and stream count '%s' by 1", metricNameDuration, llmFirstTokenDuration, metricNameCount)
		config.incrementCounter(metricNameDuration, llmFirstTokenDuration)
		config.incrementCounter(metricNameCount, 1)
	}
	if ctx.GetUserAttribute(LLMServiceDuration) != nil {
		llmServiceDuration, ok = convertToUInt(ctx.GetUserAttribute(LLMServiceDuration))
		if !ok {
			log.Warnf("LLMServiceDuration type assert failed")
			log.Infof(traceLogPrefix+"[writeMetric] LLMServiceDuration convertToUInt failed for value: %+v", ctx.GetUserAttribute(LLMServiceDuration))
			return
		}
		metricNameDuration := generateMetricName(route, cluster, model, consumer, LLMServiceDuration)
		metricNameCount := generateMetricName(route, cluster, model, consumer, LLMDurationCount)
		log.Infof(traceLogPrefix+"[writeMetric] Incrementing ServiceDuration metric '%s' by %d ms, and duration count '%s' by 1", metricNameDuration, llmServiceDuration, metricNameCount)
		config.incrementCounter(metricNameDuration, llmServiceDuration)
		config.incrementCounter(metricNameCount, 1)
	}
	log.Infof(traceLogPrefix + "[writeMetric] <<< Finished writeMetric successfully.")
}

func convertToUInt(val interface{}) (uint64, bool) {
	switch v := val.(type) {
	case float32:
		return uint64(v), true
	case float64:
		return uint64(v), true
	case int32:
		return uint64(v), true
	case int64:
		return uint64(v), true
	case uint32:
		return uint64(v), true
	case uint64:
		return v, true
	default:
		return 0, false
	}
}

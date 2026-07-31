package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid" // 引入 google uuid 库
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/tokenusage"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

const (
	pluginName = "ai-zdtc-token"

	headerXRequestID          = "x-request-id"
	headerXHigressLLMModel    = "x-higress-llm-model"
	headerXHigressLLMModelFin = "x-higress-llm-model-final"
	headerAuthorization       = "authorization"
	headerMseConsumer         = "x-mse-consumer"

	headerXResponseID = "x-response-id"

	ctxKeyRequestStartTime = "request_start_time"
	ctxKeyRequestUUID      = "request_uuid"
)

type TokenAuditLog struct {
	UUID           string `json:"uuid"`
	RequestID      string `json:"request_id"`
	LLMModel       string `json:"llm_model"`
	LLMModelFinal  string `json:"llm_model_final"`
	Authorization  string `json:"authorization"`
	MseConsumer    string `json:"mse_consumer"`
	ResponseID     string `json:"response_id"`
	StartTimeMilli int64  `json:"start_time_milli"`
	EndTimeMilli   int64  `json:"end_time_milli"`
	DurationMs     int64  `json:"duration_ms"`
	InputToken     int64  `json:"input_token"`
	OutputToken    int64  `json:"output_token"`
	TotalToken     int64  `json:"total_token"`
	Model          string `json:"model"`
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
	)
}

type PluginConfig struct {
	Debug       bool      `yaml:"debug" json:"debug"`
	RedisInfo   RedisInfo `yaml:"redis" json:"redis"`
	redisClient wrapper.RedisClient
}

type RedisInfo struct {
	ServiceName string `required:"true" yaml:"service_name" json:"service_name"`
	ServicePort int    `required:"false" yaml:"service_port" json:"service_port"`
	Username    string `required:"false" yaml:"username" json:"username"`
	Password    string `required:"false" yaml:"password" json:"password"`
	Timeout     int    `required:"false" yaml:"timeout" json:"timeout"`
	Database    int    `required:"false" yaml:"database" json:"database"`
}

// 1. 解析配置阶段
func parseConfig(json gjson.Result, config *PluginConfig, log log.Log) error {
	log.Infof("[ai-zdtc-token parseConfig] === [ParseConfig] 阶段开始 ===")
	log.Infof("[ai-zdtc-token parseConfig] 收到原始 JSON 配置: %s", json.Raw)
	// {"debug":true,"redis":{"service_name":"redis.dns","service_port":6379,"timeout":2000}}

	debugResult := json.Get("debug")
	if debugResult.Exists() {
		config.Debug = debugResult.Bool()
	} else {
		config.Debug = true
	}
	log.Infof("[ai-zdtc-token parseConfig] 解析后的 Debug: %t", config.Debug)

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
	log.Infof("[ai-zdtc-token parseConfig] === [ParseConfig] 阶段结束 ===")
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {
	log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段开始 ===")

	// 在请求头处理阶段生成一个 UUID 并写入 Context
	reqUUID := uuid.New().String()
	ctx.SetContext(ctxKeyRequestUUID, reqUUID)
	if config.Debug {
		log.Infof("[ai-zdtc-token onHttpRequestHeaders] 生成请求唯一 UUID: %s 并写入 Context", reqUUID)
	}

	// 将当前请求开始的时间戳(毫秒级)保存到上下文 ctx 中
	nowMilli := time.Now().UnixMilli()
	ctx.SetContext(ctxKeyRequestStartTime, nowMilli)
	if config.Debug {
		log.Infof("[ai-zdtc-token onHttpRequestHeaders]   [Context写入成功] -> Key: %s, Value: %d (毫秒级时间戳)", ctxKeyRequestStartTime, nowMilli)
	}

	path, _ := proxywasm.GetHttpRequestHeader(":path")
	method, _ := proxywasm.GetHttpRequestHeader(":method")
	log.Infof("[ai-zdtc-token onHttpRequestHeaders] 当前请求 - 方法: %s, 路径: %s", method, path)

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

			lowerKey := strings.ToLower(key)

			switch lowerKey {
			case headerXHigressLLMModel, headerXHigressLLMModelFin, headerAuthorization, headerMseConsumer:
				ctx.SetContext(lowerKey, val)
				log.Infof("[ai-zdtc-token onHttpRequestHeaders]   [Context写入成功] -> Key: %s, Value: %s", lowerKey, val)
			case "x-request-id":
				ctx.SetContext(headerXRequestID, val)
				log.Infof("[ai-zdtc-token onHttpRequestHeaders]   [Context写入成功] -> Key: %s, Value: %s", headerXRequestID, val)
			}
		}
	}

	log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段结束 ===")
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log log.Log) types.Action {
	log.Infof("[ai-zdtc-token onHttpRequestBody] === [OnHttpRequestBody] 阶段开始 ===")
	log.Infof("[ai-zdtc-token onHttpRequestBody] 请求体大小: %d 字节", len(body))

	if config.Debug {
		if len(body) > 0 {
			log.Infof("[ai-zdtc-token onHttpRequestBody] 请求体内容: %s", string(body))
		} else {
			log.Infof("[ai-zdtc-token onHttpRequestBody] 请求体为空")
		}
	}

	log.Infof("[ai-zdtc-token onHttpRequestBody] === [OnHttpRequestBody] 阶段结束 ===")
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {
	log.Infof("[ai-zdtc-token onHttpResponseHeaders] === [OnHttpResponseHeaders] 阶段开始 ===")

	headers, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		log.Errorf("[ai-zdtc-token onHttpResponseHeaders] 无法获取响应头: %#v", err)
	} else {
		log.Infof("[ai-zdtc-token onHttpResponseHeaders] 响应头总数: %d", len(headers))
		for _, h := range headers {
			key := h[0]
			val := h[1]
			if config.Debug {
				log.Infof("[ai-zdtc-token onHttpResponseHeaders]   Header -> %s: %s", key, val)
			}
			lowerKey := strings.ToLower(key)

			switch lowerKey {
			case "x-request-id":
				ctx.SetContext(headerXResponseID, val)
				log.Infof("[ai-zdtc-token onHttpResponseHeaders]   [Context写入成功] -> Key: %s, Value: %s", headerXResponseID, val)
			}
		}
	}

	log.Infof("[ai-zdtc-token onHttpResponseHeaders] === [OnHttpResponseHeaders] 阶段结束 ===")
	return types.ActionContinue
}

func onHttpStreamResponseBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isLastChunk bool, log log.Log) []byte {
	log.Infof("[ai-zdtc-token onHttpStreamResponseBody] === [OnHttpStreamResponseBody] 阶段触发 ===")
	log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 当前数据块大小: %d 字节, 是否为最后一个数据块: %t", len(chunk), isLastChunk)

	if config.Debug {
		if len(chunk) > 0 {
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 数据块内容: %s", string(chunk))
		}
	}
	if usage := tokenusage.GetTokenUsage(ctx, chunk); usage.TotalToken > 0 {
		ctx.SetContext(tokenusage.CtxKeyInputToken, usage.InputToken)
		ctx.SetContext(tokenusage.CtxKeyOutputToken, usage.OutputToken)
		ctx.SetContext(tokenusage.CtxKeyTotalToken, usage.TotalToken)
		ctx.SetContext(tokenusage.CtxKeyModel, usage.Model)
	}

	if isLastChunk {
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 收到最后一个数据块, 尝试解析 Token 消耗数据")

		var requestUUID, requestID, llmModel, llmModelFinal, authorization, mseConsumer, responseID string
		var startTimeMilli int64

		if v := ctx.GetContext(ctxKeyRequestUUID); v != nil {
			requestUUID, _ = v.(string)
		}
		if v := ctx.GetContext(headerXRequestID); v != nil {
			requestID, _ = v.(string)
		}
		if v := ctx.GetContext(headerXHigressLLMModel); v != nil {
			llmModel, _ = v.(string)
		}
		if v := ctx.GetContext(headerXHigressLLMModelFin); v != nil {
			llmModelFinal, _ = v.(string)
		}
		if v := ctx.GetContext(headerAuthorization); v != nil {
			authorization, _ = v.(string)
			// authorization: Bearer t07uhpqbo5zoj4z7cfarhuf0
			authorization = strings.TrimPrefix(authorization, "Bearer ")
			authorization = strings.TrimSpace(authorization)
		}
		if v := ctx.GetContext(headerMseConsumer); v != nil {
			mseConsumer, _ = v.(string)
		}
		if v := ctx.GetContext(headerXResponseID); v != nil {
			responseID, _ = v.(string)
		}
		if v := ctx.GetContext(ctxKeyRequestStartTime); v != nil {
			startTimeMilli, _ = v.(int64)
		}

		if config.Debug {
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 读取到暂存的 Context 字段值:")
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - UUID: %s", requestUUID)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXRequestID, requestID)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXHigressLLMModel, llmModel)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXHigressLLMModelFin, llmModelFinal)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerAuthorization, authorization)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerMseConsumer, mseConsumer)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXResponseID, responseID)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %d (毫秒级时间戳)", ctxKeyRequestStartTime, startTimeMilli)
		}

		var inputToken, outputToken, totalToken int64
		var modelStr string

		if v := ctx.GetContext(tokenusage.CtxKeyInputToken); v != nil {
			inputToken, _ = v.(int64)
		}
		if v := ctx.GetContext(tokenusage.CtxKeyOutputToken); v != nil {
			outputToken, _ = v.(int64)
		}
		if v := ctx.GetContext(tokenusage.CtxKeyTotalToken); v != nil {
			totalToken, _ = v.(int64)
		}
		if v := ctx.GetContext(tokenusage.CtxKeyModel); v != nil {
			modelStr, _ = v.(string)
		}

		if config.Debug {
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 模型名称 (Model): %s", modelStr)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 输入 Token 数量 (InputToken): %d", inputToken)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 输出 Token 数量 (OutputToken): %d", outputToken)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 总共 Token 数量 (TotalToken): %d", totalToken)
		}

		// 如果 input_token 或者 output_token 为零, 就不写入 Redis
		if inputToken == 0 || outputToken == 0 {
			log.Warnf("[ai-zdtc-token onHttpStreamResponseBody] 检测到 input_token (%d) 或 output_token (%d) 为 0, 跳过 Redis 写入", inputToken, outputToken)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody] === [OnHttpStreamResponseBody] 阶段结束 ===")
			return chunk
		}

		// 构造 Redis Key 规则: pluginName|mseConsumer|requestID|responseID
		redisKey := fmt.Sprintf("%s|%s|%s|%s", pluginName, mseConsumer, requestID, responseID)
		endTimeMilli := time.Now().UnixMilli()
		var durationMs int64
		if startTimeMilli > 0 {
			durationMs = endTimeMilli - startTimeMilli
		}
		auditLog := TokenAuditLog{
			UUID:           requestUUID, // 写入生成的唯一 UUID
			RequestID:      requestID,
			LLMModel:       llmModel,
			LLMModelFinal:  llmModelFinal,
			Authorization:  authorization,
			MseConsumer:    mseConsumer,
			ResponseID:     responseID,
			StartTimeMilli: startTimeMilli,
			EndTimeMilli:   endTimeMilli,
			DurationMs:     durationMs,
			InputToken:     inputToken,
			OutputToken:    outputToken,
			TotalToken:     totalToken,
			Model:          modelStr,
		}

		jsonBytes, err := json.Marshal(auditLog)
		if err != nil {
			log.Errorf("[ai-zdtc-token onHttpStreamResponseBody] 构造的 JSON 序列化失败: %#v", err)
		} else {
			jsonStr := string(jsonBytes)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 异步向 Redis 写入. Key: %s, Value: %s", redisKey, jsonStr)
			// Key: ai-zdtc-token|1-96323044|0ed4cf5c-8baf-43e4-9138-9bc863eca9d6|
			// Value: {"uuid":"4d8094c1-431c-46bf-afe4-54a8dada16f4","request_id":"0ed4cf5c-8baf-43e4-9138-9bc863eca9d6","llm_model":"capital","llm_model_final":"Vendor3/DeepSeek-V4-Flash","authorization":"pg589f6in04e5xqy6srrphk6","mse_consumer":"1-96323044","response_id":"","start_time_milli":1785477049103,"end_time_milli":1785477051608,"duration_ms":2505,"input_token":5,"output_token":111,"total_token":116,"model":"Vendor3/DeepSeek-V4-Flash"}

			err = config.redisClient.Set(redisKey, jsonStr, func(response resp.Value) {
				if response.Error() != nil {
					log.Errorf("[ai-zdtc-token onHttpStreamResponseBody] Redis 写入执行失败: %#v", response.Error())
				} else {
					log.Infof("[ai-zdtc-token onHttpStreamResponseBody] Redis 写入成功! 已保存完整大模型统计信息至 Redis")
				}
			})
			if err != nil {
				log.Errorf("[ai-zdtc-token onHttpStreamResponseBody] 异步调用 Redis 接口时发生内部错误: %#v", err)
			}
		}
	}

	log.Infof("[ai-zdtc-token onHttpStreamResponseBody] === [OnHttpStreamResponseBody] 阶段结束 ===")
	return chunk
}

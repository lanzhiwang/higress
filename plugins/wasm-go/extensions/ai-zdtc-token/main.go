package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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

	ctxKeyRequestUUID      = "request_uuid"
	ctxKeyRequestStartTime = "request_start_time"

	headerXHigressLLMModel    = "x-higress-llm-model"
	headerXHigressLLMModelFin = "x-higress-llm-model-final"
	headerAuthorization       = "authorization"
	headerXHiOriginalAuth     = "x-hi-original-auth"
	headerMseConsumer         = "x-mse-consumer"
	headerXRequestID          = "x-request-id"

	headerXResponseID = "x-response-id"

	ctxKeySkipPlugin = "skip_plugin"
)

type PluginConfig struct {
	Debug       bool      `yaml:"debug" json:"debug"`
	RedisInfo   RedisInfo `yaml:"redis" json:"redis"`
	redisClient wrapper.RedisClient
	PayInfo     PayInfo `yaml:"pay" json:"pay"`
	payClient   wrapper.HttpClient
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
	ResponseID     string `json:"response_id"`
	StartTimeMilli int64  `json:"start_time_milli"`
	EndTimeMilli   int64  `json:"end_time_milli"`
	DurationMs     int64  `json:"duration_ms"`
	InputToken     int64  `json:"input_token"`
	OutputToken    int64  `json:"output_token"`
	TotalToken     int64  `json:"total_token"`
	Model          string `json:"model"`
}

var (
	insufficientBalanceResponseHeaders = [][2]string{{"content-type", "application/json; charset=utf-8"}}
	insufficientBalanceResponseBody    = []byte(`{"message": "账户余额不足", "error": {"message": "账户余额不足"}}`)
)

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

	// 发起 Redis 连通性及 Authentication/DB 选择的探活校验
	pingKey := fmt.Sprintf("%s:__ping_test__", pluginName)
	err = config.redisClient.Get(pingKey, func(response resp.Value) {
		if response.Error() != nil {
			log.Errorf("[ai-zdtc-token parseConfig] Redis Ping/探活失败! 请检查服务名(%s)、端口(%d)、密码或数据库(%d)配置. 错误信息: %v",
				config.RedisInfo.ServiceName, config.RedisInfo.ServicePort, config.RedisInfo.Database, response.Error())
		} else {
			log.Infof("[ai-zdtc-token parseConfig] Redis 连通性及认证探活成功! (服务: %s:%d, DB: %d 响应正常)",
				config.RedisInfo.ServiceName, config.RedisInfo.ServicePort, config.RedisInfo.Database)
		}
	})

	if err != nil {
		log.Errorf("[ai-zdtc-token parseConfig] 发起 Redis 探活请求异常 (网关可能找不到 Redis 路由集群): %#v", err)
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

	var mseConsumerVal string
	var authHeaderVal string

	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Errorf("[ai-zdtc-token onHttpRequestHeaders] 无法获取请求头: %#v", err)
		return types.ActionContinue
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
			case headerXHigressLLMModel, headerXHigressLLMModelFin, headerAuthorization, headerXHiOriginalAuth, headerMseConsumer:
				ctx.SetContext(lowerKey, val)
				log.Infof("[ai-zdtc-token onHttpRequestHeaders]   [Context写入成功] -> Key: %s, Value: %s", lowerKey, val)
				if lowerKey == headerMseConsumer {
					mseConsumerVal = val
				}
				if lowerKey == headerXHiOriginalAuth {
					authHeaderVal = val
				}
			case "x-request-id":
				ctx.SetContext(headerXRequestID, val)
				log.Infof("[ai-zdtc-token onHttpRequestHeaders]   [Context写入成功] -> Key: %s, Value: %s", headerXRequestID, val)
			}
		}
	}

	// 如果 x-hi-original-auth 的值以 Bearer sp 开头, 那么就直接放行, 跳过后面的所有逻辑
	if authHeaderVal != "" {
		trimmedAuth := strings.TrimSpace(authHeaderVal)
		if strings.HasPrefix(trimmedAuth, "Bearer sp") {
			log.Infof("[ai-zdtc-token onHttpRequestHeaders] 校验到 x-hi-original-auth 以 'Bearer sp' 开头 (%s), 直接放行", trimmedAuth)
			ctx.SetContext(ctxKeySkipPlugin, true)
			log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段结束 ===")
			return types.ActionContinue
		}
	}

	// 从头信息中解析 mseConsumer 并进行余额校验
	if mseConsumerVal != "" {
		tenantID, _, ok := parseMseConsumer(mseConsumerVal)
		if ok && tenantID != "" {
			if config.payClient == nil {
				log.Warnf("[ai-zdtc-token onHttpRequestHeaders] 已解析出 tenantID: %s, 但 payClient 未初始化, 跳过余额校验", tenantID)
			} else {
				redisKey := fmt.Sprintf("%s:balance:%s", tenantID, pluginName)

				// 1. 优先查 Redis 缓存
				err := config.redisClient.Get(redisKey, func(response resp.Value) {
					if response.Error() == nil && !response.IsNull() {
						cachedData := response.String()
						log.Infof("[ai-zdtc-token onHttpRequestHeaders] Redis 命中租户 %s 余额缓存数据", tenantID)

						// 同时解析可用余额(cash_balance)与测试金(free_balance)
						cashBalance, freeBalance, parseErr := parseBalances(cachedData)
						if parseErr == nil {
							log.Infof("[ai-zdtc-token onHttpRequestHeaders] [Redis Hit] 租户 %s 资金明细 -> 账户可用余额(cash_balance): %.4f, 测试金(free_balance): %.4f",
								tenantID, cashBalance, freeBalance)

							// 资金校验规则: 只有当 cash_balance <= 0 并且 free_balance <= 0 时, 才判定为资金不足并拦截请求
							if cashBalance <= 0 && freeBalance <= 0 {
								log.Warnf("[ai-zdtc-token onHttpRequestHeaders] [Redis Hit] 租户 %s 账户可用余额 (%.4f) 与测试金 (%.4f) 均 <= 0, 资金不足, 拦截请求",
									tenantID, cashBalance, freeBalance)
								sendInsufficientBalanceResponse()
								// SendHttpResponse 触发 Envoy 拦截后, 必须 return 退出当前异步闭包, 否则程序会继续向下执行 ResumeHttpRequest 等逻辑导致状态冲突.
								return
							}

							log.Infof("[ai-zdtc-token onHttpRequestHeaders] [Redis Hit] 租户 %s 账户资金校验通过 (cash_balance: %.4f, free_balance: %.4f), 允许放行",
								tenantID, cashBalance, freeBalance)
							proxywasm.ResumeHttpRequest()
							// ResumeHttpRequest 已恢复 Envoy 过滤器链继续处理请求, 必须 return 退出闭包, 否则会向下执行 callPayService 重复请求远程接口.
							return
						}
						log.Errorf("[ai-zdtc-token onHttpRequestHeaders] 解析 Redis 缓存余额数据失败: %#v, 准备请求 Pay 服务", parseErr)
					}

					// 2. Redis 未命中或解析失败, 请求远程 Pay HTTP 服务
					callPayService(config, tenantID, redisKey, log)
				})

				if err != nil {
					log.Errorf("[ai-zdtc-token onHttpRequestHeaders] 调用 Redis Get 异常: %#v, 直接请求 Pay 服务", err)
					callPayService(config, tenantID, redisKey, log)
					// 请求 Pay 服务, 这里是不是应该直接 return
					// 不需要也不应该在这里 return. callPayService 内部是异步 HTTP 调用, 发起异步请求后, 主函数仍需继续运行到末尾返回 ActionPause, 通知 Envoy 暂停请求并等待回调.
				}

				log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段暂停(等待异步余额校验) ===")
				return types.ActionPause
			}
		} else {
			log.Warnf("[ai-zdtc-token onHttpRequestHeaders] 警告: x-mse-consumer 格式不正确 mseConsumerVal: %s", mseConsumerVal)
		}
	} else {
		log.Warnf("[ai-zdtc-token onHttpRequestHeaders] 警告: 未正常从头信息中获取到 x-mse-consumer mseConsumerVal: %s", mseConsumerVal)
	}

	log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段结束 ===")
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log log.Log) types.Action {
	log.Infof("[ai-zdtc-token onHttpRequestBody] === [OnHttpRequestBody] 阶段开始 ===")

	if skip, ok := ctx.GetContext(ctxKeySkipPlugin).(bool); ok && skip {
		log.Infof("[ai-zdtc-token onHttpRequestBody] 校验到 x-hi-original-auth 以 'Bearer sp' 开头, 直接放行")
		return types.ActionContinue
	}

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

	if skip, ok := ctx.GetContext(ctxKeySkipPlugin).(bool); ok && skip {
		log.Infof("[ai-zdtc-token onHttpResponseHeaders] 校验到 x-hi-original-auth 以 'Bearer sp' 开头, 直接放行")
		return types.ActionContinue
	}

	headers, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		log.Errorf("[ai-zdtc-token onHttpResponseHeaders] 无法获取响应头: %#v", err)
		// 获取响应头失败时跳过分析, 返回 ActionContinue 允许响应正常返回给客户端.
		return types.ActionContinue
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

	if skip, ok := ctx.GetContext(ctxKeySkipPlugin).(bool); ok && skip {
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 校验到 x-hi-original-auth 以 'Bearer sp' 开头, 直接放行")
		return chunk
	}

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

		var requestUUID, requestID, llmModel, llmModelFinal, authorization, originalAuth, mseConsumer, responseID string
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
			authorization = strings.TrimPrefix(authorization, "Bearer ")
			authorization = strings.TrimSpace(authorization)
		}
		if v := ctx.GetContext(headerXHiOriginalAuth); v != nil {
			originalAuth, _ = v.(string)
			originalAuth = strings.TrimPrefix(originalAuth, "Bearer ")
			originalAuth = strings.TrimSpace(originalAuth)
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
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXHiOriginalAuth, originalAuth)
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
		redisKey := fmt.Sprintf("{%s}-|%s|%s|%s", pluginName, mseConsumer, requestID, responseID)
		endTimeMilli := time.Now().UnixMilli()
		var durationMs int64
		if startTimeMilli > 0 {
			durationMs = endTimeMilli - startTimeMilli
		}
		auditLog := TokenAuditLog{
			UUID:           requestUUID,
			RequestID:      requestID,
			LLMModel:       llmModel,
			LLMModelFinal:  llmModelFinal,
			Authorization:  authorization,
			OriginalAuth:   originalAuth,
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

// 解析 mseConsumer 的规则函数
func parseMseConsumer(mseConsumer string) (tenantID, userID string, ok bool) {
	// 严格规则1: 必须且仅包含一个 "-"
	if strings.Count(mseConsumer, "-") != 1 {
		return "", "", false
	}

	parts := strings.Split(mseConsumer, "-")
	// 严格规则2: 切分后必须是 2 段, 且两段内容均不能为空
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	// 顺序: 前段是租户 ID, 后段是用户 ID
	return parts[0], parts[1], true
}

// 异步请求 Pay 服务的帮助方法
func callPayService(config PluginConfig, tenantID string, redisKey string, log log.Log) {
	path := "/v0/financial_center/get_user_balance_basic_info"

	var bodyStr string
	if tenantIDInt, err := strconv.ParseInt(tenantID, 10, 64); err == nil {
		bodyStr = fmt.Sprintf(`{"user_id": %d}`, tenantIDInt)
	} else {
		bodyStr = fmt.Sprintf(`{"user_id": "%s"}`, tenantID)
	}

	headers := [][2]string{
		{"Content-Type", "application/json"},
	}

	err := config.payClient.Post(path, headers, []byte(bodyStr), func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		log.Infof("[ai-zdtc-token callPayService] 查询余额接口返回状态码: %d, body: %s", statusCode, string(responseBody))

		if statusCode != http.StatusOK {
			log.Errorf("[ai-zdtc-token callPayService] 查询余额接口返回非 200 状态码: %d, body: %s, 降级放行请求", statusCode, string(responseBody))
			proxywasm.ResumeHttpRequest()
			// 接口返回非 200 触发降级放行, 必须 return 退出回调闭包, 防止代码向下继续解析数据.
			return
		}

		dataResult := gjson.GetBytes(responseBody, "data")
		if !dataResult.Exists() {
			log.Errorf("[ai-zdtc-token callPayService] 查询余额接口响应缺少 data 字段, body: %s, 降级放行请求", string(responseBody))
			proxywasm.ResumeHttpRequest()
			// 缺少 data 字段无法进行后续余额校验, 降级放行后必须 return 终止执行.
			return
		}

		dataRaw := dataResult.Raw
		log.Infof("[ai-zdtc-token callPayService] 查询余额接口返回 dataRaw: %s", dataRaw)

		// 从远程响应的 data 中解析账户可用余额(cash_balance)与测试金(free_balance)
		cashBalance, freeBalance, err := parseBalances(dataRaw)
		if err != nil {
			log.Errorf("[ai-zdtc-token callPayService] 解析余额字段 (cash_balance/free_balance) 失败: %#v, body: %s, 降级放行请求", err, string(responseBody))
			proxywasm.ResumeHttpRequest()
			// 余额字段解析异常进入降级逻辑, ResumeHttpRequest 恢复网关流后必须 return, 避免误入下方的拦截逻辑.
			return
		}

		log.Infof("[ai-zdtc-token callPayService] 查询 Pay 接口成功, 租户 %s 资金明细 -> 账户可用余额(cash_balance): %.4f, 测试金(free_balance): %.4f",
			tenantID, cashBalance, freeBalance)

		// 1. 将 data 的所有信息写入 Redis
		cacheTTL := config.PayInfo.BalanceCacheTTL
		log.Infof("[ai-zdtc-token callPayService] 将租户 %s 的余额 data 写入 Redis (TTL %ds), Key: %s, Value: %s", tenantID, cacheTTL, redisKey, dataRaw)
		// 使用 SetEx 保证原子性并避免嵌套回调导致的 Envoy 生命周期失效
		err = config.redisClient.SetEx(redisKey, dataRaw, cacheTTL, func(respVal resp.Value) {
			if respVal.Error() != nil {
				log.Errorf("[ai-zdtc-token callPayService] Redis 写入失败: %#v", respVal.Error())
			} else {
				log.Infof("[ai-zdtc-token callPayService] Redis 写入成功, TTL 成功设置为 %d 秒", cacheTTL)
			}
		})
		if err != nil {
			log.Errorf("[ai-zdtc-token callPayService] 调用 Redis SetEx 接口异常: %#v", err)
		}

		// 2. 资金拦截判定: 只有当 cash_balance <= 0 且 free_balance <= 0 时, 才拦截该请求
		if cashBalance <= 0 && freeBalance <= 0 {
			log.Warnf("[ai-zdtc-token callPayService] 租户 %s 账户可用余额 (%.4f) 和测试金 (%.4f) 均 <= 0, 资金不足, 拦截请求",
				tenantID, cashBalance, freeBalance)
			sendInsufficientBalanceResponse()
			// sendInsufficientBalanceResponse 发送 403 拦截并终止下游请求后, 必须 return 退出回调, 防止再调用下方的 ResumeHttpRequest.
			return
		}

		log.Infof("[ai-zdtc-token callPayService] 租户 %s 资金充裕 (cash_balance: %.4f, free_balance: %.4f), 余额校验通过",
			tenantID, cashBalance, freeBalance)
		proxywasm.ResumeHttpRequest()
	}, uint32(config.PayInfo.Timeout))

	if err != nil {
		log.Errorf("[ai-zdtc-token callPayService] 异步调用 Pay 服务接口失败: %#v, 降级放行请求", err)
		proxywasm.ResumeHttpRequest()
	}
}

// 从 data 的 JSON 字符串中解析 cash_balance (账户可用余额) 和 free_balance (测试金) 为 float64
func parseBalances(dataJson string) (cashBalance float64, freeBalance float64, err error) {
	// 1. 解析 cash_balance (账户可用余额)
	cashStr := gjson.Get(dataJson, "cash_balance").String()
	if cashStr == "" {
		return 0, 0, fmt.Errorf("cash_balance 字段为空或不存在")
	}
	cashVal, err := strconv.ParseFloat(cashStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("解析 cash_balance (%s) 为 float64 失败: %w", cashStr, err)
	}

	// 2. 解析 free_balance (测试金 = 免费账户收入 - 免费账户支出)
	freeStr := gjson.Get(dataJson, "free_balance").String()
	if freeStr == "" {
		return 0, 0, fmt.Errorf("free_balance 字段为空或不存在")
	}
	freeVal, err := strconv.ParseFloat(freeStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("解析 free_balance (%s) 为 float64 失败: %w", freeStr, err)
	}

	return cashVal, freeVal, nil
}

func sendInsufficientBalanceResponse() {
	proxywasm.SendHttpResponse(http.StatusForbidden, insufficientBalanceResponseHeaders, insufficientBalanceResponseBody, -1)
}

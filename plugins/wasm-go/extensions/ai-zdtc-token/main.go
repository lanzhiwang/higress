package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
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
	ctxKeyIsStream    = "is_stream"     // 标识当前请求是否为流式(SSE)请求
	ctxKeyHasSeenDone = "has_seen_done" // 标识流式请求是否已正常收到 [DONE] 结束标志
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

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {
	log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段开始 ===")

	// 在处理业务逻辑前, 安全触发一次探活(仅当前 Worker 线程首次请求会真实执行)
	probeRedisIfNeeded(config, log)

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

	var mseConsumerVal string    // 用于余额校验 (来自 x-mse-cuser: 1-1)
	var mseApiConsumerVal string // 用于查询用户来源 tenant_from (来自 x-mse-consumer: 1-87894137)
	var authHeaderVal string     // 用于免检放行逻辑 (来自 x-hi-original-auth)
	var decoratorOpVal string    // 用于提取供应商 ID (来自 x-envoy-decorator-operation)

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

			lowerKey := strings.ToLower(key)

			switch lowerKey {
			case headerXHigressLLMModel, headerXHigressLLMModelFin, headerAuthorization, headerXHiOriginalAuth, headerMseConsumer, headerMseApi, headerXRequestID, headerEnvoyDecoratorOperation:
				ctx.SetContext(lowerKey, val)
				log.Infof("[ai-zdtc-token onHttpRequestHeaders]   [Context写入成功] -> Key: %s, Value: %s", lowerKey, val)

				// 用于获取租户余额校验标识 (x-mse-cuser)
				if lowerKey == headerMseConsumer {
					mseConsumerVal = val
				}
				// 用于获取用户来源 tenant_from (x-mse-consumer)
				if lowerKey == headerMseApi {
					mseApiConsumerVal = val
				}
				// 如果 x-hi-original-auth 的值以 Bearer sp 开头, 那么就直接放行, 跳过后面的所有逻辑
				if lowerKey == headerXHiOriginalAuth {
					authHeaderVal = val
				}
				// 用于提取供应商 ID (x-envoy-decorator-operation)
				if lowerKey == headerEnvoyDecoratorOperation {
					decoratorOpVal = val
				}
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

	// 从 x-envoy-decorator-operation 中提取供应商 ID 并转换为 int64 记录至 Context
	providerIDStr := extractProviderID(decoratorOpVal)
	var providerID int64
	if providerIDStr != "" {
		if val, err := strconv.ParseInt(providerIDStr, 10, 64); err == nil {
			providerID = val
			log.Infof("[ai-zdtc-token onHttpRequestHeaders] 成功从 %s 提取到供应商 ID (ProviderID): %d 并存入 Context", decoratorOpVal, providerID)
		} else {
			log.Warnf("[ai-zdtc-token onHttpRequestHeaders] 提取的供应商 ID (%s) 转换为 int64 失败: %#v, ProviderID 默认为 0", providerIDStr, err)
		}
	} else {
		log.Infof("[ai-zdtc-token onHttpRequestHeaders] 未从 %s 中提取到有效的供应商 ID, ProviderID 默认为 0", decoratorOpVal)
	}
	ctx.SetContext(ctxKeyProviderID, providerID)

	// 余额校验流程: 异步查询 tenant_from 并串联执行余额校验
	return checkTenantFromAndBalance(ctx, config, mseApiConsumerVal, mseConsumerVal, log)
}

// 异步处理 tenant_from 查询与余额校验的主入口
func checkTenantFromAndBalance(ctx wrapper.HttpContext, config PluginConfig, mseApiConsumerVal string, mseConsumerVal string, log log.Log) types.Action {
	// 如果既没有 x-mse-consumer 也没有 x-mse-cuser, 直接放行, 无需异步暂停
	if mseApiConsumerVal == "" && mseConsumerVal == "" {
		log.Infof("[ai-zdtc-token onHttpRequestHeaders] 未检测到 x-mse-consumer 与 x-mse-cuser, 跳过来源查询与余额校验")
		ctx.SetContext(ctxKeyTenantFrom, int64(0))
		log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段结束 ===")
		return types.ActionContinue
	}

	// 情况 A: 存在 x-mse-consumer, 需要从 Redis Hash 获取 tenant_from 字段
	if mseApiConsumerVal != "" {
		consumerRedisKey := fmt.Sprintf("key-auth:consumer:%s", mseApiConsumerVal)
		log.Infof("[ai-zdtc-token onHttpRequestHeaders] 开始异步查询 Redis Hash 数据 [Key: %s, Field: tenant_from]...", consumerRedisKey)

		err := config.redisClient.HGet(consumerRedisKey, "tenant_from", func(response resp.Value) {
			var tenantFrom int64
			if response.Error() != nil {
				log.Errorf("[ai-zdtc-token onHttpRequestHeaders] 查询 Redis Hash [Key: %s, Field: tenant_from] 发生异常: %#v, tenant_from 将记录为 0", consumerRedisKey, response.Error())
			} else if response.IsNull() {
				log.Warnf("[ai-zdtc-token onHttpRequestHeaders] Redis Hash [Key: %s] 中未找到 tenant_from 字段(或值为 null), 记录为 0", consumerRedisKey)
			} else {
				rawStr := strings.TrimSpace(response.String())
				if val, err := strconv.ParseInt(rawStr, 10, 64); err == nil {
					tenantFrom = val
					log.Infof("[ai-zdtc-token onHttpRequestHeaders] 成功从 Redis Hash [Key: %s] 获取并解析 tenant_from: %d", consumerRedisKey, tenantFrom)
				} else {
					log.Errorf("[ai-zdtc-token onHttpRequestHeaders] 解析 Redis 返回的 tenant_from 字符串 (%s) 为 int64 失败: %#v, tenant_from 将记录为 0", rawStr, err)
				}
			}
			// 将获取到的 int64 tenant_from 写入 Context (失败或空值时写入 0)
			ctx.SetContext(ctxKeyTenantFrom, tenantFrom)

			// 在 HGet 异步回调中继续执行余额校验逻辑 (此时网关已处于 Pause 状态)
			proceedWithBalanceCheckInAsync(ctx, config, mseConsumerVal, log)
		})

		if err != nil {
			log.Errorf("[ai-zdtc-token onHttpRequestHeaders] 发起 Redis HGet 异步调用失败: %#v, tenant_from 记录为 0", err)
			ctx.SetContext(ctxKeyTenantFrom, int64(0))

			// 发起 HGet 失败时, 如果存在 mseConsumerVal 则同步尝试发起余额校验
			if mseConsumerVal != "" {
				return startBalanceCheckSync(ctx, config, mseConsumerVal, log)
			}
			log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段结束 ===")
			return types.ActionContinue
		}

		// Redis HGet 异步发起成功, 暂停当前请求等待回调链完成
		log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段暂停(等待异步 tenant_from 与余额校验) ===")
		return types.ActionPause
	}

	// 情况 B: 仅存在 x-mse-cuser (无需查询 tenant_from, 默认置 0)
	ctx.SetContext(ctxKeyTenantFrom, int64(0))
	return startBalanceCheckSync(ctx, config, mseConsumerVal, log)
}

// 同步发起余额校验流程 (用于尚未进入 ActionPause 的生命周期阶段)
func startBalanceCheckSync(ctx wrapper.HttpContext, config PluginConfig, mseConsumerVal string, log log.Log) types.Action {
	tenantID, _, ok := parseMseConsumer(mseConsumerVal)
	if !ok || tenantID == "" {
		log.Warnf("[ai-zdtc-token onHttpRequestHeaders] 警告: x-mse-cuser 格式不正确 (%s), 跳过余额校验", mseConsumerVal)
		log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段结束 ===")
		return types.ActionContinue
	}

	if config.payClient == nil {
		log.Warnf("[ai-zdtc-token onHttpRequestHeaders] 已解析出 tenantID: %s, 但 payClient 未初始化, 跳过余额校验", tenantID)
		log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段结束 ===")
		return types.ActionContinue
	}

	redisKey := fmt.Sprintf("%s:balance:%s", tenantID, pluginName)

	// 1. 优先查 Redis 缓存
	err := config.redisClient.Get(redisKey, func(response resp.Value) {
		if response.Error() == nil && !response.IsNull() {
			cachedData := response.String()
			log.Infof("[ai-zdtc-token onHttpRequestHeaders] Redis 命中租户 %s 余额缓存数据", tenantID)

			cashBalance, freeBalance, parseErr := parseBalances(cachedData)
			if parseErr == nil {
				log.Infof("[ai-zdtc-token onHttpRequestHeaders] [Redis Hit] 租户 %s 资金明细 -> 账户可用余额(cash_balance): %.4f, 测试金(free_balance): %.4f",
					tenantID, cashBalance, freeBalance)

				if cashBalance <= 0 && freeBalance <= 0 {
					log.Warnf("[ai-zdtc-token onHttpRequestHeaders] [Redis Hit] 租户 %s 账户可用余额 (%.4f) 与测试金 (%.4f) 均 <= 0, 资金不足, 拦截请求",
						tenantID, cashBalance, freeBalance)
					sendInsufficientBalanceResponse()
					return
				}

				log.Infof("[ai-zdtc-token onHttpRequestHeaders] [Redis Hit] 租户 %s 账户资金校验通过 (cash_balance: %.4f, free_balance: %.4f), 允许放行",
					tenantID, cashBalance, freeBalance)
				proxywasm.ResumeHttpRequest()
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
	}

	log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段暂停(等待异步余额校验) ===")
	return types.ActionPause
}

// 在异步回调上下文中执行余额校验 (此时网关已被 ActionPause 暂停, 结束时必须 Resume 或 SendHttpResponse)
func proceedWithBalanceCheckInAsync(ctx wrapper.HttpContext, config PluginConfig, mseConsumerVal string, log log.Log) {
	if mseConsumerVal == "" {
		log.Infof("[ai-zdtc-token onHttpRequestHeaders] 未配置 x-mse-cuser, 跳过余额校验, 恢复请求")
		proxywasm.ResumeHttpRequest()
		return
	}

	tenantID, _, ok := parseMseConsumer(mseConsumerVal)
	if !ok || tenantID == "" {
		log.Warnf("[ai-zdtc-token onHttpRequestHeaders] 警告: x-mse-cuser 格式不正确 (%s), 跳过余额校验, 恢复请求", mseConsumerVal)
		proxywasm.ResumeHttpRequest()
		return
	}

	if config.payClient == nil {
		log.Warnf("[ai-zdtc-token onHttpRequestHeaders] 已解析出 tenantID: %s, 但 payClient 未初始化, 跳过余额校验, 恢复请求", tenantID)
		proxywasm.ResumeHttpRequest()
		return
	}

	redisKey := fmt.Sprintf("%s:balance:%s", tenantID, pluginName)

	// 1. 优先查 Redis 缓存
	err := config.redisClient.Get(redisKey, func(response resp.Value) {
		if response.Error() == nil && !response.IsNull() {
			cachedData := response.String()
			log.Infof("[ai-zdtc-token onHttpRequestHeaders] Redis 命中租户 %s 余额缓存数据", tenantID)

			cashBalance, freeBalance, parseErr := parseBalances(cachedData)
			if parseErr == nil {
				log.Infof("[ai-zdtc-token onHttpRequestHeaders] [Redis Hit] 租户 %s 资金明细 -> 账户可用余额(cash_balance): %.4f, 测试金(free_balance): %.4f",
					tenantID, cashBalance, freeBalance)

				if cashBalance <= 0 && freeBalance <= 0 {
					log.Warnf("[ai-zdtc-token onHttpRequestHeaders] [Redis Hit] 租户 %s 账户可用余额 (%.4f) 与测试金 (%.4f) 均 <= 0, 资金不足, 拦截请求",
						tenantID, cashBalance, freeBalance)
					sendInsufficientBalanceResponse()
					return
				}

				log.Infof("[ai-zdtc-token onHttpRequestHeaders] [Redis Hit] 租户 %s 账户资金校验通过 (cash_balance: %.4f, free_balance: %.4f), 允许放行",
					tenantID, cashBalance, freeBalance)
				proxywasm.ResumeHttpRequest()
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
	}
}

func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log log.Log) types.Action {
	log.Infof("[ai-zdtc-token onHttpRequestBody] === [OnHttpRequestBody] 阶段开始 ===")

	if skip, ok := ctx.GetContext(ctxKeySkipPlugin).(bool); ok && skip {
		log.Infof("[ai-zdtc-token onHttpRequestBody] 校验到 x-hi-original-auth 以 'Bearer sp' 开头, 直接放行")
		return types.ActionContinue
	}

	log.Infof("[ai-zdtc-token onHttpRequestBody] 请求体大小: %d 字节", len(body))

	if len(body) > 0 {
		// 从请求体中探测 stream 字段
		// 如果客户端在请求体中显式传了 "stream": true, 则标记为流式请求, 否则默认为非流式
		if isStream := gjson.GetBytes(body, "stream").Bool(); isStream {
			ctx.SetContext(ctxKeyIsStream, true)
			log.Infof("[ai-zdtc-token onHttpRequestBody] 检测到请求体包含 stream=true, 标记为流式(Stream)请求")
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

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {
	log.Infof("[ai-zdtc-token onHttpResponseHeaders] === [OnHttpResponseHeaders] 阶段开始 ===")

	if skip, ok := ctx.GetContext(ctxKeySkipPlugin).(bool); ok && skip {
		log.Infof("[ai-zdtc-token onHttpResponseHeaders] 校验到 x-hi-original-auth 以 'Bearer sp' 开头, 直接放行")
		return types.ActionContinue
	}

	// 1: 获取 HTTP 响应状态码 :status
	// 获取状态码以记录客户端 4xx(如 400 参数错误、401 鉴权失效)或上游 5xx(如 504 超时、500 服务异常), 正常情况记录为 200
	statusStr, err := proxywasm.GetHttpResponseHeader(":status")
	if err == nil && statusStr != "" {
		if statusCode, convErr := strconv.Atoi(statusStr); convErr == nil {
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

	// 从响应头 Content-Type 探测 text/event-stream
	// 上游响应流式数据时 Content-Type 必为 text/event-stream; 非流式通常为 application/json
	if cType, err := proxywasm.GetHttpResponseHeader("content-type"); err == nil && cType != "" {
		if strings.Contains(strings.ToLower(cType), "text/event-stream") {
			ctx.SetContext(ctxKeyIsStream, true)
			log.Infof("[ai-zdtc-token onHttpResponseHeaders] 检测到响应 Content-Type 为 text/event-stream, 标记为流式响应")
		}
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

			lowerKey := strings.ToLower(key)

			switch lowerKey {
			case "x-cds-request-id":
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

	// 辅助兜底探测数据块前缀
	// 如果响应块中包含 "data:", 进一步确保 is_stream 被正确标记为 true
	if len(chunk) > 0 && strings.Contains(string(chunk), "data:") {
		ctx.SetContext(ctxKeyIsStream, true)
	}

	// 检查当前 chunk 是否包含 [DONE]
	// 标准 OpenAI/vLLM SSE 流正常结束时一定会输出包含 `data: [DONE]` 的帧
	if checkHasDoneMarker(chunk) {
		ctx.SetContext(ctxKeyHasSeenDone, true)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 检测到流式传输正常结束标志 [DONE], 成功标记 has_seen_done=true")
	}

	// 标准 Token 消耗解析 (提取 input_token, output_token, total_token, model)
	if usage := tokenusage.GetTokenUsage(ctx, chunk); usage.TotalToken > 0 {
		ctx.SetContext(tokenusage.CtxKeyInputToken, usage.InputToken)
		ctx.SetContext(tokenusage.CtxKeyOutputToken, usage.OutputToken)
		ctx.SetContext(tokenusage.CtxKeyTotalToken, usage.TotalToken)
		ctx.SetContext(tokenusage.CtxKeyModel, usage.Model)
	}

	// 提取上游服务返回的缓存 Token (cached_tokens)
	// 上游模型服务 (如 Qwen/vLLM, OpenAI 等) 在开启 Prompt 缓存时会在 usage.prompt_tokens_details.cached_tokens 中返回命中缓存的 token 数量
	if cachedTokens := extractCachedTokens(chunk); cachedTokens > 0 {
		ctx.SetContext(ctxKeyCachedToken, cachedTokens)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 成功提取到 Cached Tokens 数量: %d 并存入 Context", cachedTokens)
	}

	// 捕获响应体中的异常报错信息 (如客户端参数错误、令牌无效等)
	// 当上游发生报错(例如参数不合法、无效令牌或网关错误)时, 响应体通常包含 {"error": {"message": ...}} 结构
	if errMsg := extractErrorMessage(chunk); errMsg != "" {
		ctx.SetContext(ctxKeyErrorMessage, errMsg)
		log.Warnf("[ai-zdtc-token onHttpStreamResponseBody] 捕获到响应数据块中的异常报错信息: %s", errMsg)
	}

	if isLastChunk {
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 收到最后一个数据块, 尝试解析 Token 消耗与审计数据")

		var requestUUID, requestID, llmModel, llmModelFinal, authorization, originalAuth, mseConsumer, responseID string
		var tenantFrom, providerID int64
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

		// 读取暂存的 tenant_from (int64)
		if v := ctx.GetContext(ctxKeyTenantFrom); v != nil {
			if val, ok := v.(int64); ok {
				tenantFrom = val
			}
		}

		// 读取暂存的 provider_id (int64)
		if v := ctx.GetContext(ctxKeyProviderID); v != nil {
			if val, ok := v.(int64); ok {
				providerID = val
			}
		}

		if v := ctx.GetContext(headerXResponseID); v != nil {
			responseID, _ = v.(string)
		}
		if v := ctx.GetContext(ctxKeyRequestStartTime); v != nil {
			startTimeMilli, _ = v.(int64)
		}

		// 读取已暂存的 CachedToken
		var cachedToken int64
		if v := ctx.GetContext(ctxKeyCachedToken); v != nil {
			cachedToken, _ = v.(int64)
		}

		// 读取响应状态码, 默认为 200 OK (若未获取到则默认 200)
		statusCode := http.StatusOK
		if v := ctx.GetContext(ctxKeyResponseStatus); v != nil {
			if code, ok := v.(int); ok && code > 0 {
				statusCode = code
			}
		}

		// 读取错误信息, 正常 200 请求为 "" (空字符串)
		var errorMessage string
		if v := ctx.GetContext(ctxKeyErrorMessage); v != nil {
			errorMessage, _ = v.(string)
		}

		// 读取流式请求标识与 [DONE] 标志
		var isStream bool
		if v := ctx.GetContext(ctxKeyIsStream); v != nil {
			isStream, _ = v.(bool)
		}

		var hasSeenDone bool
		if v := ctx.GetContext(ctxKeyHasSeenDone); v != nil {
			hasSeenDone, _ = v.(bool)
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

		// 流式中途截断与非流式兼容处理核心逻辑
		// 1. 如果状态码为 504 (网关超时) 且未从 Body 解析到错误文本, 则赋予明确的超时说明
		if statusCode == http.StatusGatewayTimeout && errorMessage == "" {
			errorMessage = "上游响应超时 (Gateway Timeout 504)"
		} else if statusCode >= 400 && errorMessage == "" {
			errorMessage = fmt.Sprintf("HTTP 响应异常状态码: %d", statusCode)
		} else if statusCode == http.StatusOK && errorMessage == "" {
			// 2. 状态码为 200 正常时的完整性判定:
			if isStream {
				// 流式请求分支:
				// 标准 OpenAI/vLLM 流式正常结束时必定返回 [DONE] 标志和 usage 统计信息.
				// 若 isStream == true, 但 hasSeenDone == false 且 totalToken == 0, 说明流在生成中途被截断(客户端断开/上游崩溃等)
				if !hasSeenDone && totalToken == 0 {
					errorMessage = "流式传输异常中断(未收到完整的[DONE]与Usage统计信息)"
					log.Warnf("[ai-zdtc-token onHttpStreamResponseBody] 捕获到流式异常截断: 状态码 200, 属于流式请求, 但未检测到 [DONE] 标志且未统计到 Token, 标记为流中断")
				} else if !hasSeenDone && totalToken > 0 {
					log.Warnf("[ai-zdtc-token onHttpStreamResponseBody] 警告: 流式请求未检测到标准 [DONE] 标志, 但已成功解析到 Usage (totalToken: %d)", totalToken)
				}
			} else {
				// 非流式请求分支:
				// 非流式请求上游直接返回完整的 JSON 对象, 永远不会输出 [DONE] 标志.
				// 因此非流式请求完全不受 hasSeenDone 影响, 只要 totalToken > 0, errorMessage 保持为空字符串 "", 正常记录审计.
				if totalToken == 0 {
					log.Warnf("[ai-zdtc-token onHttpStreamResponseBody] 非流式请求响应 200 但未解析到 Token 统计数据 (totalToken=0)")
				} else {
					log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 非流式请求成功完成, Token 统计正常 (totalToken: %d)", totalToken)
				}
			}
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
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 租户来源 (TenantFrom): %d", tenantFrom)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 供应商ID (ProviderID): %d", providerID)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXResponseID, responseID)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %d (毫秒级时间戳)", ctxKeyRequestStartTime, startTimeMilli)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 响应状态码 (StatusCode): %d", statusCode)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 是否为流式 (IsStream): %t", isStream)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 是否检测到[DONE] (HasSeenDone): %t", hasSeenDone)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 异常错误信息 (ErrorMessage): %s", errorMessage)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 模型名称 (Model): %s", modelStr)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 输入 Token 数量 (InputToken): %d", inputToken)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 输出 Token 数量 (OutputToken): %d", outputToken)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 缓存 Token 数量 (CachedToken): %d", cachedToken)
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 总共 Token 数量 (TotalToken): %d", totalToken)
		}

		// 构造 Redis Key 规则: {pluginName}-|mseConsumer|requestID|responseID
		redisKey := fmt.Sprintf("{%s}-|%s|%s|%s", pluginName, mseConsumer, requestID, responseID)
		endTimeMilli := time.Now().UnixMilli()
		var durationMs int64
		if startTimeMilli > 0 {
			durationMs = endTimeMilli - startTimeMilli
		}

		// 构造审计结构体 (无论成功、失败、Token为0还是中途截断均 100% 完整记录)
		auditLog := TokenAuditLog{
			UUID:           requestUUID,
			RequestID:      requestID,
			LLMModel:       llmModel,
			LLMModelFinal:  llmModelFinal,
			Authorization:  authorization,
			OriginalAuth:   originalAuth,
			MseConsumer:    mseConsumer,
			TenantFrom:     tenantFrom,
			ProviderID:     providerID,
			ResponseID:     responseID,
			StartTimeMilli: startTimeMilli,
			EndTimeMilli:   endTimeMilli,
			DurationMs:     durationMs,
			InputToken:     inputToken,
			OutputToken:    outputToken,
			TotalToken:     totalToken,
			CachedToken:    cachedToken,
			Model:          modelStr,
			StatusCode:     statusCode,
			ErrorMessage:   errorMessage,
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
					log.Infof("[ai-zdtc-token onHttpStreamResponseBody] Redis 写入成功! 已保存完整大模型统计与审计信息至 Redis")
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

// 辅助函数: 从 x-envoy-decorator-operation 中提取供应商 ID
// 示例输入: "llm-provider-1.internal.dns:443/*" -> 提取 "1"
// 兼容 "llm-provider-123.internal.dns", "llm-provider-456:443" 等各类格式
func extractProviderID(decoratorOp string) string {
	if decoratorOp == "" {
		return ""
	}

	const prefix = "llm-provider-"
	idx := strings.Index(decoratorOp, prefix)
	if idx == -1 {
		return ""
	}

	// 截取前缀之后的内容
	sub := decoratorOp[idx+len(prefix):]
	if sub == "" {
		return ""
	}

	// 供应商 ID 截止到下一个 '.', ':', '/' 边界字符或字符串结尾
	endIdx := strings.IndexAny(sub, ".:/")
	if endIdx != -1 {
		return sub[:endIdx]
	}

	return sub
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

// 辅助函数: 执行 Redis 探活(仅在首个请求时被 sync.Once 触发)
func probeRedisIfNeeded(config PluginConfig, log log.Log) {
	if config.probeOnce == nil || config.redisClient == nil {
		return
	}

	config.probeOnce.Do(func() {
		pingKey := fmt.Sprintf("%s:__ping_test__", pluginName)
		log.Infof("[ai-zdtc-token probeRedisIfNeeded] 触发当前 WASM VM 线程的首次 Redis 连通性探活 (Key: %s)...", pingKey)

		// 发起异步探活
		err := config.redisClient.Get(pingKey, func(response resp.Value) {
			if response.Error() != nil {
				log.Errorf("[ai-zdtc-token probeRedisIfNeeded] Redis 探活校验失败! 错误信息: %v", response.Error())
			} else {
				log.Infof("[ai-zdtc-token probeRedisIfNeeded] Redis 探活成功! 连通性与鉴权正常.")
			}
		})

		if err != nil {
			log.Errorf("[ai-zdtc-token probeRedisIfNeeded] 发起 Redis 探活异步调用失败: %#v", err)
		}
	})
}

// 检测数据块中是否包含流式结束标志 [DONE]
// 理由: 标准 OpenAI/vLLM SSE 协议在流结束时会发送包含 `data: [DONE]` 的行
func checkHasDoneMarker(chunk []byte) bool {
	if len(chunk) == 0 {
		return false
	}
	chunkStr := string(chunk)
	if !strings.Contains(chunkStr, "[DONE]") {
		return false
	}
	lines := strings.Split(chunkStr, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "data: [DONE]" || trimmed == "data:[DONE]" || strings.HasPrefix(trimmed, "data: [DONE]") {
			return true
		}
	}
	return false
}

// 从响应数据块中提取缓存 Token (cached_tokens)
// 理由: 大模型上游返回 cached_tokens 字段通常在 `usage.prompt_tokens_details.cached_tokens` 中,
// 本函数兼容非流式 JSON 以及 SSE 流式输出(以 `data: ` 开头的行), 确保准确提取.
func extractCachedTokens(chunk []byte) int64 {
	if len(chunk) == 0 {
		return 0
	}

	// 1. 优先尝试直接从标准 JSON 块中提取
	if res := gjson.GetBytes(chunk, "usage.prompt_tokens_details.cached_tokens"); res.Exists() && res.Int() > 0 {
		return res.Int()
	}

	// 2. 兼容 SSE 流式块 (遍历每行并剥离 "data: " 前缀)
	chunkStr := string(chunk)
	if strings.Contains(chunkStr, "data:") {
		lines := strings.Split(chunkStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				dataContent := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if dataContent == "[DONE]" || dataContent == "" {
					continue
				}
				if res := gjson.Get(dataContent, "usage.prompt_tokens_details.cached_tokens"); res.Exists() && res.Int() > 0 {
					return res.Int()
				}
			}
		}
	}

	// 3. 兜底搜索 prompt_tokens_details.cached_tokens 结构
	if res := gjson.GetBytes(chunk, "..prompt_tokens_details.cached_tokens"); res.Exists() && res.Int() > 0 {
		return res.Int()
	}

	return 0
}

// 从响应数据块中提取业务异常错误信息
// 理由: 当发生客户端参数错误、令牌失效(如日志中的 new_api_error)、模型不存在或网关错误时,
// 提取 Body 中的 error 对象内容, 方便在审计日志中直观展现错误原因.
func extractErrorMessage(chunk []byte) string {
	if len(chunk) == 0 {
		return ""
	}

	// 1. 尝试直接解析标准 OpenAI 风格的 JSON 错误: {"error": {"message": "...", "code": "...", "type": "..."}}
	errObj := gjson.GetBytes(chunk, "error")
	if errObj.Exists() {
		if errObj.IsObject() {
			msg := errObj.Get("message").String()
			code := errObj.Get("code").String()
			errType := errObj.Get("type").String()

			var parts []string
			if msg != "" {
				parts = append(parts, msg)
			}
			if code != "" {
				parts = append(parts, fmt.Sprintf("code: %s", code))
			}
			if errType != "" {
				parts = append(parts, fmt.Sprintf("type: %s", errType))
			}
			if len(parts) > 0 {
				return strings.Join(parts, " | ")
			}
			return errObj.Raw
		}
		return errObj.String()
	}

	// 2. 兼容 SSE 格式中的 error 事件或 data 包含的 error 字段
	chunkStr := string(chunk)
	if strings.Contains(chunkStr, "data:") {
		lines := strings.Split(chunkStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				dataContent := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if dataContent == "[DONE]" || dataContent == "" {
					continue
				}
				if errRes := gjson.Get(dataContent, "error"); errRes.Exists() {
					if msg := errRes.Get("message").String(); msg != "" {
						return msg
					}
					return errRes.Raw
				}
			}
		}
	}

	return ""
}

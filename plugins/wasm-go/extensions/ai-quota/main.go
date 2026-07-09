package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/tokenusage"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-quota/util"
)

const (
	pluginName = "ai-quota"
)

type ChatMode string

const (
	ChatModeCompletion ChatMode = "completion"
	ChatModeAdmin      ChatMode = "admin"
	ChatModeNone       ChatMode = "none"
)

type AdminMode string

const (
	AdminModeRefresh AdminMode = "refresh"
	AdminModeQuery   AdminMode = "query"
	AdminModeDelta   AdminMode = "delta"
	AdminModeNone    AdminMode = "none"
)

func main() {}

// init 函数在 Go 包加载时首先执行.
// 当 Higress (基于 Envoy WASM 虚拟机) 加载该插件的 WASM 模块时, 会自动调用此 init 函数.
func init() {
	// wrapper.SetCtx 是 Higress Wasm-Go SDK 提供的主注册接口.
	// 它的核心作用是将底层复杂的 Proxy-WASM 接口(C++ 规范的 ABI), 封装映射为
	// 开发者友好、基于泛型的 Go 语言生命周期回调函数(Hooks).
	wrapper.SetCtx(
		// 1. 插件名称
		// [作用]: 在网关中标识该插件. Higress 控制台下发的插件配置会根据此名称进行匹配与绑定.
		pluginName,

		// 2. 注册配置解析函数
		// [作用]: 指定在网关配置下发时, 如何将 YAML/JSON 配置反序列化并预处理.
		// [执行时机]: 在 WASM 虚拟机启动、插件首次加载, 或网关管理员在控制台动态更新该插件的配置时执行.
		//            该阶段发生在处理具体 HTTP 请求之前. 它是"单次执行"的, 可以避免请求处理时重复解析配置, 从而极大提升网关性能.
		wrapper.ParseConfig(parseConfig),

		// 3. 注册请求头处理函数(对应 Envoy 过滤器的 DecodeHeaders 阶段)
		// [作用]: 拦截并处理 HTTP 请求头. 可用于读取客户端 Header(如 Token、API-Key 校验)、
		//          进行流控检查、黑白名单过滤, 或者直接拦截请求(例如发现 Quota 已耗尽则直接返回 429 响应).
		// [执行时机]: 当客户端发送的 HTTP 请求到达网关, 网关完全解析好请求头(Request Headers),
		//            但尚未将请求转发给上游 AI 服务之前.
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),

		// 4. 注册请求体处理函数(对应 Envoy 过滤器的 DecodeData 阶段)
		// [作用]: 拦截并处理 HTTP 请求体(Body). 在 AI 场景下, 通常在此处提取用户输入的 Prompt、
		//          分析输入 Token 长度或对 Body 里的敏感参数进行校验.
		// [执行时机]: 在请求头处理完后, 若客户端请求中含有 Body(如 POST 请求),
		//            随着请求体的数据块(Data Chunks)流经网关时触发.
		wrapper.ProcessRequestBody(onHttpRequestBody),

		// 5. 注册流式响应体处理函数(对应 Envoy 过滤器的 EncodeData 阶段的流式扩展)
		// [作用]: 专门用于拦截和处理 AI 场景下的流式(Streaming)响应体(如 SSE - Server-Sent Events).
		//          在 ai-quota 插件中, 此回调用于实时截获大模型流式吐出的每一个 Token, 以便精准计算大模型实际生成的输出 Token 数量,
		//          并在流式结束时扣减对应的配额.
		// [执行时机]: 当上游 AI 服务开始向网关流式返回响应数据, 网关每接收到一个流式数据块(Chunk)时触发.
		//            在此处处理完数据块后, 网关会将其流式地转发给终端用户.
		wrapper.ProcessStreamingResponseBody(onHttpStreamingResponseBody),
	)
}

type QuotaConfig struct {
	redisInfo          RedisInfo         `yaml:"redis"`
	RedisKeyPrefix     string            `yaml:"redis_key_prefix"`
	AdminConsumer      string            `yaml:"admin_consumer"`
	AdminPath          string            `yaml:"admin_path"`
	EnablePathSuffixes []string          `yaml:"enable_path_suffixes"`
	credential2Name    map[string]string `yaml:"-"`
	redisClient        wrapper.RedisClient
}

type Consumer struct {
	Name       string `yaml:"name"`
	Credential string `yaml:"credential"`
}

type RedisInfo struct {
	ServiceName string `required:"true" yaml:"service_name" json:"service_name"`
	ServicePort int    `required:"false" yaml:"service_port" json:"service_port"`
	Username    string `required:"false" yaml:"username" json:"username"`
	Password    string `required:"false" yaml:"password" json:"password"`
	Timeout     int    `required:"false" yaml:"timeout" json:"timeout"`
	Database    int    `required:"false" yaml:"database" json:"database"`
}

// parseConfig 是插件的配置解析回调函数.
// json: 网关控制台下发并自动转换为 JSON 的插件配置.
// config: 预先分配好的插件配置结构体指针, 解析后的数据会存入其中, 后续请求处理时可直接读取此结构体.
func parseConfig(json gjson.Result, config *QuotaConfig) error {
	log.Debugf("parse config()")

	// ==================== 1. 解析管理员/管理接口相关配置 ====================

	// 获取用于配额管理(例如查询、重置配额)的 API 路径, 默认为 "/quota"

	// admin
	config.AdminPath = json.Get("admin_path").String()
	config.AdminConsumer = json.Get("admin_consumer").String()
	if config.AdminPath == "" {
		config.AdminPath = "/quota"
	}

	// 解析生效的请求路径后缀. 插件仅会对匹配这些后缀的请求(如 /v1/chat/completions)进行配额校验与扣减.
	suffixResult := json.Get("enable_path_suffixes")
	if !suffixResult.Exists() {
		// 如果未配置, 默认支持 OpenAI 和 Anthropic 的典型聊天/消息接口
		config.EnablePathSuffixes = []string{"/v1/chat/completions", "/v1/messages"}
	} else if !suffixResult.IsArray() {
		// 校验配置格式是否合法
		return errors.New("enable_path_suffixes must be an array")
	} else {
		pathSuffixes := suffixResult.Array()
		config.EnablePathSuffixes = make([]string, 0, len(pathSuffixes))
		for _, suffix := range pathSuffixes {
			suffixStr := strings.TrimSpace(suffix.String())
			if suffixStr == "" {
				continue
			}
			config.EnablePathSuffixes = append(config.EnablePathSuffixes, suffixStr)
		}
	}

	// 基础合法性校验: 生效路径不能为空
	if len(config.EnablePathSuffixes) == 0 {
		return errors.New("enable_path_suffixes must not be empty")
	}

	// 基础合法性校验: 必须配置管理调用者(Admin Consumer), 用于管理接口的鉴权
	if config.AdminConsumer == "" {
		return errors.New("missing admin_consumer in config")
	}

	// ==================== 2. 解析 Redis 相关配置 ====================

	// 获取 Redis Key 的前缀, 默认值为 "chat_quota:", 用于区分其他插件或业务的 Key
	// Redis
	config.RedisKeyPrefix = json.Get("redis_key_prefix").String()
	if config.RedisKeyPrefix == "" {
		config.RedisKeyPrefix = "chat_quota:"
	}
	redisConfig := json.Get("redis")
	if !redisConfig.Exists() {
		return errors.New("missing redis in config")
	}

	// 获取 Redis 在网关中定义的服务名称(对应 Envoy/K8s Service)
	serviceName := redisConfig.Get("service_name").String()
	if serviceName == "" {
		return errors.New("redis service name must not be empty")
	}

	// 获取 Redis 服务端口
	servicePort := int(redisConfig.Get("service_port").Int())
	if servicePort == 0 {
		// Higress 特有逻辑: 若是 ".static" 结尾的静态服务, 默认端口为 80;
		// 否则, 使用标准的 Redis 默认端口 6379.
		if strings.HasSuffix(serviceName, ".static") {
			// use default logic port which is 80 for static service
			servicePort = 80
		} else {
			servicePort = 6379
		}
	}

	// 获取 Redis 鉴权及超时配置
	username := redisConfig.Get("username").String()
	password := redisConfig.Get("password").String()
	timeout := int(redisConfig.Get("timeout").Int())
	if timeout == 0 {
		// 默认超时时间 1000 毫秒
		timeout = 1000
	}
	database := int(redisConfig.Get("database").Int())

	// 将解析出的 Redis 参数保存到配置结构体中
	config.redisInfo.ServiceName = serviceName
	config.redisInfo.ServicePort = servicePort
	config.redisInfo.Username = username
	config.redisInfo.Password = password
	config.redisInfo.Timeout = timeout
	config.redisInfo.Database = database

	// ==================== 3. 初始化 Redis 客户端 ====================

	// 由于 WASM 沙箱的安全限制, WASM 插件无法像普通 Go 程序那样直接创建 TCP 原始连接.
	// 这里必须使用 Higress SDK 提供的 `wrapper.NewRedisClusterClient`,
	// 它通过网关宿主机(Envoy)提供的 Host Calls API 来代理执行 Redis 指令.
	config.redisClient = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: int64(servicePort),
	})

	// 初始化 Redis 客户端连接(传入账号、密码、超时以及选择的数据库 db 索引)
	return config.redisClient.Init(username, password, int64(timeout), wrapper.WithDataBase(database))
}

// onHttpRequestHeaders 是请求头处理阶段的回调函数.
// context: 当前请求的上下文, 可用于跨阶段共享数据或对请求体/响应体进行流控.
// config: 已解析的插件全局配置, 在此阶段为只读.
func onHttpRequestHeaders(context wrapper.HttpContext, config QuotaConfig) types.Action {
	// 1. 禁止自动重路由
	// 在 Envoy 过滤器中, 修改 Headers 有可能触发路由重新计算(Reroute).
	// 在此处显式禁用 Reroute 可以避免不必要的性能开销, 确保路由规则的一致性.
	context.DisableReroute()

	log.Debugf("onHttpRequestHeaders()")

	// 2. 提取消费者(Consumer)身份标识
	// "x-mse-consumer" 是 Higress 在前置认证插件(如 key-auth / jwt-auth)鉴权成功后自动注入的 Header,
	// 代表当前的调用者身份. 如果提取失败或为空, 说明请求未经过认证, 直接返回未授权错误.
	// get tokens
	consumer, err := proxywasm.GetHttpRequestHeader("x-mse-consumer")
	if err != nil {
		// 辅助函数: 构造并直接向客户端返回 401 响应
		return deniedNoKeyAuthData()
	}
	if consumer == "" {
		// 辅助函数: 构造并直接向客户端返回 403/401 响应
		return deniedUnauthorizedConsumer()
	}

	// 3. 提取请求 Path 并分析操作模式
	// 区分当前请求是普通大模型对话(Chat 模式), 还是配额的后台管理操作(Admin 模式).
	rawPath := context.Path()
	path, _ := url.Parse(rawPath)
	chatMode, adminMode := getOperationMode(path.Path, config.AdminPath, config.EnablePathSuffixes)
	// ChatModeAdmin      ChatMode = "admin"
	// AdminModeRefresh AdminMode = "refresh"

	// ChatModeAdmin      ChatMode = "admin"
	// AdminModeDelta   AdminMode = "delta"

	// ChatModeAdmin      ChatMode = "admin"
	// AdminModeQuery   AdminMode = "query"

	// ChatModeCompletion ChatMode = "completion"
	// AdminModeNone    AdminMode = "none"

	// ChatModeNone       ChatMode = "none"
	// AdminModeNone    AdminMode = "none"

	// 将这些业务字段注入到 HttpContext 中, 以便后续的 onHttpRequestBody 或 onHttpStreamingResponseBody 钩子可以直接获取.
	context.SetContext("chatMode", chatMode)
	context.SetContext("adminMode", adminMode)
	context.SetContext("consumer", consumer)
	log.Debugf("chatMode:%s, adminMode:%s, consumer:%s", chatMode, adminMode, consumer)

	// 4. 分流处理: 场景 A - 路径不属于该插件的管理和生效范围
	if chatMode == ChatModeNone {
		// 无需处理, 将请求放行传递给下一个过滤器
		return types.ActionContinue
	}

	// 5. 分流处理: 场景 B - 管理员操作模式(例如管理员通过 API 查询或充值额度)
	if chatMode == ChatModeAdmin {
		// 场景 B-1: 管理员仅查询配额
		// query quota
		if adminMode == AdminModeQuery {
			// 直接返回查询结果, 并在内部向客户端响应, 不再透传给上游业务服务
			return queryQuota(context, config, consumer, path)
		}

		// 场景 B-2: 管理员需要充值(Refresh)或增减(Delta)配额
		if adminMode == AdminModeRefresh || adminMode == AdminModeDelta {
			// 这些写操作需要读取 HTTP 的 Body(以获取充值具体的数值或 JSON),
			// 因而在此处调用 BufferRequestBody 告知 Envoy 缓存请求体,
			// 并返回 HeaderStopIteration 挂起 Headers 阶段, 等待 Body 接收完毕后再进入 Body 处理回调.
			context.BufferRequestBody()
			return types.HeaderStopIteration
		}
		return types.ActionContinue
	}

	// 6. 分流处理: 场景 C - 普通 AI 对话请求配额校验(AI-Quota 核心逻辑)

	// [性能优化 1]: 既然在此阶段仅仅是"校验"用户是否还有配额(仅查 Redis 即可),
	// 并不需要解析请求体里的 Prompt, 所以使用 DontReadRequestBody 告知 Envoy
	// 跳过当前插件对请求 Body 的读取, 从而节省内存拷贝和 CPU 开销.
	// there is no need to read request body when it is on chat completion mode
	context.DontReadRequestBody()

	// [非阻塞 I/O 1]: 调用 Redis 异步客户端发起 Get 请求, 查询该 Consumer 的剩余配额.
	// 其内部实现是基于 Envoy 宿主机非阻塞机制的.
	// check quota here
	config.redisClient.Get(config.RedisKeyPrefix+consumer, func(response resp.Value) {
		isDenied := false

		// 校验 Redis 查询结果
		if err := response.Error(); err != nil {
			// Redis 异常, 默认拒绝请求(防穿透兜底, 也可以根据业务设计选择降级放行)
			isDenied = true
		}
		if response.IsNull() {
			// Redis 中没有该用户的额度记录, 拒绝请求
			isDenied = true
		}
		if response.Integer() <= 0 {
			// 额度已用尽, 拒绝请求
			isDenied = true
		}
		log.Debugf("get consumer:%s quota:%d isDenied:%t", consumer, response.Integer(), isDenied)
		if isDenied {
			// 如果配额不足, 直接通过 WASM API 向客户端返回 HTTP 403 Forbidden
			util.SendResponse(http.StatusForbidden, "ai-quota.noquota", "text/plain", "Request denied by ai quota check, No quota left")
			return
		}

		// 如果校验通过, 异步回调结束前, 必须显式调用 ResumeHttpRequest() 恢复已被挂起的 HTTP 请求
		proxywasm.ResumeHttpRequest()
	})

	// [非阻塞 I/O 2]: 由于上面的 Redis 查询是异步进行的, 我们绝不能让 Envoy 线程阻塞等待.
	// 这里必须立即返回 HeaderStopAllIterationAndWatermark, 告知网关:
	// "暂停此请求所有后续过滤器的执行, 且开启流控缓存(Watermark)防止数据溢出, 等待我的 Resume 信号再继续."
	return types.HeaderStopAllIterationAndWatermark
}

func onHttpRequestBody(ctx wrapper.HttpContext, config QuotaConfig, body []byte) types.Action {
	log.Debugf("onHttpRequestBody()")

	// ChatModeAdmin      ChatMode = "admin"
	// AdminModeRefresh AdminMode = "refresh"

	// ChatModeAdmin      ChatMode = "admin"
	// AdminModeDelta   AdminMode = "delta"

	// ChatModeAdmin      ChatMode = "admin"
	// AdminModeQuery   AdminMode = "query"

	// ChatModeCompletion ChatMode = "completion"
	// AdminModeNone    AdminMode = "none"

	// ChatModeNone       ChatMode = "none"
	// AdminModeNone    AdminMode = "none"
	chatMode, ok := ctx.GetContext("chatMode").(ChatMode)
	if !ok {
		return types.ActionContinue
	}
	if chatMode == ChatModeNone || chatMode == ChatModeCompletion {
		return types.ActionContinue
	}
	adminMode, ok := ctx.GetContext("adminMode").(AdminMode)
	if !ok {
		return types.ActionContinue
	}
	adminConsumer, ok := ctx.GetContext("consumer").(string)
	if !ok {
		return types.ActionContinue
	}

	if adminMode == AdminModeRefresh {
		return refreshQuota(ctx, config, adminConsumer, string(body))
	}
	if adminMode == AdminModeDelta {
		return deltaQuota(ctx, config, adminConsumer, string(body))
	}

	return types.ActionContinue
}

func onHttpStreamingResponseBody(ctx wrapper.HttpContext, config QuotaConfig, data []byte, endOfStream bool) []byte {
	chatMode, ok := ctx.GetContext("chatMode").(ChatMode)
	if !ok {
		return data
	}
	if chatMode == ChatModeNone || chatMode == ChatModeAdmin {
		return data
	}
	if usage := tokenusage.GetTokenUsage(ctx, data); usage.TotalToken > 0 {
		ctx.SetContext(tokenusage.CtxKeyInputToken, usage.InputToken)
		ctx.SetContext(tokenusage.CtxKeyOutputToken, usage.OutputToken)
	}

	// chat completion mode
	if !endOfStream {
		return data
	}

	if ctx.GetContext(tokenusage.CtxKeyInputToken) == nil || ctx.GetContext(tokenusage.CtxKeyOutputToken) == nil || ctx.GetContext("consumer") == nil {
		return data
	}

	inputToken := ctx.GetContext(tokenusage.CtxKeyInputToken).(int64)
	outputToken := ctx.GetContext(tokenusage.CtxKeyOutputToken).(int64)
	consumer := ctx.GetContext("consumer").(string)
	totalToken := int(inputToken + outputToken)
	log.Debugf("update consumer:%s, totalToken:%d", consumer, totalToken)
	config.redisClient.DecrBy(config.RedisKeyPrefix+consumer, totalToken, nil)
	return data
}

func deniedNoKeyAuthData() types.Action {
	util.SendResponse(http.StatusUnauthorized, "ai-quota.no_key", "text/plain", "Request denied by ai quota check. No Key Authentication information found.")
	return types.ActionContinue
}

func deniedUnauthorizedConsumer() types.Action {
	util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. Unauthorized consumer.")
	return types.ActionContinue
}

func getOperationMode(path string, adminPath string, pathSuffixes []string) (ChatMode, AdminMode) {
	fullAdminPath := "/v1/chat/completions" + adminPath
	if strings.HasSuffix(path, fullAdminPath+"/refresh") {
		// ChatModeAdmin      ChatMode = "admin"
		// AdminModeRefresh AdminMode = "refresh"
		return ChatModeAdmin, AdminModeRefresh
	}
	if strings.HasSuffix(path, fullAdminPath+"/delta") {
		// ChatModeAdmin      ChatMode = "admin"
		// AdminModeDelta   AdminMode = "delta"
		return ChatModeAdmin, AdminModeDelta
	}
	if strings.HasSuffix(path, fullAdminPath) {
		// ChatModeAdmin      ChatMode = "admin"
		// AdminModeQuery   AdminMode = "query"
		return ChatModeAdmin, AdminModeQuery
	}
	for _, suffix := range pathSuffixes {
		if strings.HasSuffix(path, suffix) {
			// ChatModeCompletion ChatMode = "completion"
			// AdminModeNone    AdminMode = "none"
			return ChatModeCompletion, AdminModeNone
		}
	}
	// ChatModeNone       ChatMode = "none"
	// AdminModeNone    AdminMode = "none"
	return ChatModeNone, AdminModeNone
}

func refreshQuota(ctx wrapper.HttpContext, config QuotaConfig, adminConsumer string, body string) types.Action {
	// check consumer
	if adminConsumer != config.AdminConsumer {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. Unauthorized admin consumer.")
		return types.ActionContinue
	}

	queryValues, _ := url.ParseQuery(body)
	values := make(map[string]string, len(queryValues))
	for k, v := range queryValues {
		values[k] = v[0]
	}
	queryConsumer := values["consumer"]
	quota, err := strconv.Atoi(values["quota"])
	if queryConsumer == "" || err != nil {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. consumer can't be empty and quota must be integer.")
		return types.ActionContinue
	}
	err2 := config.redisClient.Set(config.RedisKeyPrefix+queryConsumer, quota, func(response resp.Value) {
		log.Debugf("Redis set key = %s quota = %d", config.RedisKeyPrefix+queryConsumer, quota)
		if err := response.Error(); err != nil {
			util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
			return
		}
		util.SendResponse(http.StatusOK, "ai-quota.refreshquota", "text/plain", "refresh quota successful")
	})

	if err2 != nil {
		util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
		return types.ActionContinue
	}

	return types.ActionPause
}

func queryQuota(ctx wrapper.HttpContext, config QuotaConfig, adminConsumer string, url *url.URL) types.Action {
	// check consumer
	if adminConsumer != config.AdminConsumer {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. Unauthorized admin consumer.")
		return types.ActionContinue
	}
	// check url
	queryValues := url.Query()
	values := make(map[string]string, len(queryValues))
	for k, v := range queryValues {
		values[k] = v[0]
	}
	if values["consumer"] == "" {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. consumer can't be empty.")
		return types.ActionContinue
	}
	queryConsumer := values["consumer"]
	err := config.redisClient.Get(config.RedisKeyPrefix+queryConsumer, func(response resp.Value) {
		quota := 0
		if err := response.Error(); err != nil {
			util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
			return
		} else if response.IsNull() {
			quota = 0
		} else {
			quota = response.Integer()
		}
		result := struct {
			Consumer string `json:"consumer"`
			Quota    int    `json:"quota"`
		}{
			Consumer: queryConsumer,
			Quota:    quota,
		}
		body, _ := json.Marshal(result)
		util.SendResponse(http.StatusOK, "ai-quota.queryquota", "application/json", string(body))
	})
	if err != nil {
		util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
		return types.ActionContinue
	}
	return types.ActionPause
}

func deltaQuota(ctx wrapper.HttpContext, config QuotaConfig, adminConsumer string, body string) types.Action {
	// check consumer
	if adminConsumer != config.AdminConsumer {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. Unauthorized admin consumer.")
		return types.ActionContinue
	}

	queryValues, _ := url.ParseQuery(body)
	values := make(map[string]string, len(queryValues))
	for k, v := range queryValues {
		values[k] = v[0]
	}
	queryConsumer := values["consumer"]
	value, err := strconv.Atoi(values["value"])
	if queryConsumer == "" || err != nil {
		util.SendResponse(http.StatusForbidden, "ai-quota.unauthorized", "text/plain", "Request denied by ai quota check. consumer can't be empty and value must be integer.")
		return types.ActionContinue
	}

	if value >= 0 {
		err := config.redisClient.IncrBy(config.RedisKeyPrefix+queryConsumer, value, func(response resp.Value) {
			log.Debugf("Redis Incr key = %s value = %d", config.RedisKeyPrefix+queryConsumer, value)
			if err := response.Error(); err != nil {
				util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
				return
			}
			util.SendResponse(http.StatusOK, "ai-quota.deltaquota", "text/plain", "delta quota successful")
		})
		if err != nil {
			util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
			return types.ActionContinue
		}
	} else {
		err := config.redisClient.DecrBy(config.RedisKeyPrefix+queryConsumer, 0-value, func(response resp.Value) {
			log.Debugf("Redis Decr key = %s value = %d", config.RedisKeyPrefix+queryConsumer, 0-value)
			if err := response.Error(); err != nil {
				util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
				return
			}
			util.SendResponse(http.StatusOK, "ai-quota.deltaquota", "text/plain", "delta quota successful")
		})
		if err != nil {
			util.SendResponse(http.StatusServiceUnavailable, "ai-quota.error", "text/plain", fmt.Sprintf("redis error:%v", err))
			return types.ActionContinue
		}
	}

	return types.ActionPause
}

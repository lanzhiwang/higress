```bash
$ tree -a ./plugins/wasm-go/extensions/ai-history
./plugins/wasm-go/extensions/ai-history
├── .gitignore-bak
├── README.md
├── README_EN.md
├── go.mod
├── go.sum
├── learn.md
├── main.go
├── main_test.go
└── option.yaml

1 directory, 9 files
$
```

------------------------------------------------------------------------------------------------------------------------------

Q: 假设你是一位精通 go 语言, 并且精通 higress 生态的高级开发专家, 我看了 higress 插件 ai-history 的源码, 其中在 main.go 里面有如下代码
```go
func init() {
	wrapper.SetCtx(
		"ai-history",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamResponseBody),
	)
}
```
请为这个代码加上详细的注释, 说明代码的含义, 作用和执行的时机

这段代码是 Higress Wasm 插件的核心初始化入口. 它使用 Higress 的 Go SDK(`higress-wasm-go-plugin`)将插件的各个生命周期钩子(Callbacks)注册到网关底层的 Envoy 运行时中.

以下是为这段代码添加的详细中文注释. 它详细说明了每一行代码的含义、作用以及在网关处理流量时的执行时机:

```go
// init 函数是 Go 语言的包初始化函数, 在 Go 插件模块被网关加载并启动时自动运行.
// 注意: 在 Go 原生支持 Wasm 编译(Go 1.24+)的规范中, 插件的注册逻辑要求统一放置在 init() 中.
func init() {
	// wrapper.SetCtx 是 Higress SDK 提供的核心注册函数.
	// 它的作用是向 Envoy 宿主环境声明一个名为 "ai-history" 的插件,
	// 并绑定一系列流式处理(Streaming)和生命周期管理的回调函数.
	wrapper.SetCtx(
		// 1. 插件的唯一标识名称. 网关配置此插件时通过该名称进行关联.
		"ai-history",

		// 2. 注册配置解析器
		// [作用]: 将网关控制台下发的 YAML/JSON 配置反序列化为 Go 结构体.
		// [执行时机]:
		//    - 插件首次加载/初始化时.
		//    - 每次在控制台热更新(修改)该插件配置时.
		wrapper.ParseConfigBy(parseConfig),

		// 3. 注册请求头处理回调
		// [作用]: 拦截并处理客户端发送的请求头. 在 ai-history 插件中, 通常用于提取用户身份标识(如 Authorization、API Key 或自定义 Header), 用以作为后续 Redis 中历史对话的 key 标识.
		// [执行时机]: 请求阶段. 当 Higress 网关接收到客户端完整的 HTTP Request Headers 之后触发, 此时请求体(Body)通常尚未到达或尚未被读取.
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),

		// 4. 注册请求体处理回调
		// [作用]: 拦截并解析 HTTP 请求体. 在 ai-history 中, 它负责解析用户发送的 Chat JSON 请求, 提取最新的 Prompt(即 user 的当前提问), 并在此阶段准备拼接 Redis 中查询到的历史上下文.
		// [执行时机]: 请求阶段. 在请求头处理完成之后, 当网关读取到 HTTP Request Body 的数据时触发(根据网关配置, 可以是分片流式触发或等待 Body 接收完整后一次性触发).
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),

		// 5. 注册响应头处理回调
		// [作用]: 拦截并处理上游 AI 服务的响应头. 常用于判断 AI 服务是否返回 200 OK, 并检查 Content-Type 是否为流式(如 text/event-stream), 从而决定后续是否启用 SSE 流式解析.
		// [执行时机]: 响应阶段. 当上游大模型服务产生响应, 且其 Response Headers 返回到 Higress 网关时触发. 此时响应体(Body)尚未被网关转发给客户端.
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),

		// 6. 注册流式响应体处理回调
		// [作用]: 实时拦截并解析上游返回的流式响应分片.
		//        在 ai-history 插件中, 这是最核心的一步: 大模型(如通义千问、ChatGPT 等)通常以 SSE 格式分片输出回答, 该回调会提取每一次分片中的文字, 在内存中拼接成完整的大模型回复, 并在流结束时将"用户的问题 + 模型的回答"异步写入 Redis 缓存.
		// [执行时机]: 响应阶段. 当上游 AI 服务以 Streaming(流式)方式回传 HTTP Response Body 块时, 每到达一个数据分片(Chunk)就会触发一次该函数, 直至流传输结束.
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamResponseBody),
	)
}
```

### 补充: 这几个阶段在 `ai-history` 插件中的整体协同流程

为了实现"多轮对话历史管理", 这几个钩子在网关执行时是环环相扣的:

1. 配置下发 (`ParseConfigBy`): 网关启动或配置更新, 解析出 Redis 的连接地址、缓存的轮数上限(如只保留最近 3 轮对话)、历史记录的缓存 TTL(过期时间)等.

2. 请求到达 (`ProcessRequestHeadersBy`): 提取请求中携带的用户唯一 ID(如从 `Authorization` 中提取 Token), 确定要到 Redis 读写哪个用户的历史记录.

3. 读取并拼接历史 (`ProcessRequestBodyBy`):

   - 提取当前请求中的用户新提问.

   - 异步从 Redis 中读取出该用户之前的对话历史.

   - 核心改写: 将历史上下文与最新提问组合成一个新的多轮对话数组, 改写请求体后, 发送给上游大模型服务.

4. 验证响应 (`ProcessResponseHeadersBy`): 确认上游大模型返回成功, 并标记该响应是否为流式.

5. 记录历史并返回 (`ProcessStreamingResponseBodyBy`): 大模型开始吐字(流式响应), 该函数在不阻塞用户客户端接收的前提下, 悄悄将分片中的大模型回复在内存中拼接. 当接收到流结束的信号时, 将本次的"提问 + 回答"以队列形式追加写入 Redis, 以便下一次请求使用.

------------------------------------------------------------------------------------------------------------------------------

Q: 假设你是一位精通 go 语言, 并且精通 higress 生态的高级开发专家, 我看了 higress 插件 ai-history 的源码, 其中在 main.go 里面有如下代码
```go
func parseConfig(json gjson.Result, c *PluginConfig, log log.Log) error {
	c.RedisInfo.ServiceName = json.Get("redis.serviceName").String()
	if c.RedisInfo.ServiceName == "" {
		return errors.New("redis service name must not be empty")
	}
	c.RedisInfo.ServicePort = int(json.Get("redis.servicePort").Int())
	if c.RedisInfo.ServicePort == 0 {
		if strings.HasSuffix(c.RedisInfo.ServiceName, ".static") {
			// use default logic port which is 80 for static service
			c.RedisInfo.ServicePort = 80
		} else {
			c.RedisInfo.ServicePort = 6379
		}
	}
	c.RedisInfo.Username = json.Get("redis.username").String()
	c.RedisInfo.Password = json.Get("redis.password").String()
	c.RedisInfo.Timeout = int(json.Get("redis.timeout").Int())
	if c.RedisInfo.Timeout == 0 {
		c.RedisInfo.Timeout = 1000
	}
	c.RedisInfo.Database = int(json.Get("redis.database").Int())
	c.QuestionFrom.RequestBody = "messages.@reverse.0.content"
	c.AnswerValueFrom.ResponseBody = "choices.0.message.content"
	c.AnswerStreamValueFrom.ResponseBody = "choices.0.delta.content"

	c.CacheKeyPrefix = json.Get("cacheKeyPrefix").String()
	if c.CacheKeyPrefix == "" {
		c.CacheKeyPrefix = DefaultCacheKeyPrefix
	}
	c.IdentityHeader = json.Get("identityHeader").String()
	if c.IdentityHeader == "" {
		c.IdentityHeader = "Authorization"
	}
	c.FillHistoryCnt = int(json.Get("fillHistoryCnt").Int())
	if c.FillHistoryCnt == 0 {
		c.FillHistoryCnt = 3
	}
	c.CacheTTL = int(json.Get("cacheTTL").Int())
	c.redisClient = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
		FQDN: c.RedisInfo.ServiceName,
		Port: int64(c.RedisInfo.ServicePort),
	})
	return c.redisClient.Init(c.RedisInfo.Username, c.RedisInfo.Password, int64(c.RedisInfo.Timeout), wrapper.WithDataBase(c.RedisInfo.Database))
}
```
请为这个代码加上详细的注释, 说明代码的含义, 作用和执行的时机

这段 Go 语言代码是 Higress Wasm 插件中用于解析配置和初始化外部服务连接(如 Redis)的核心函数 `parseConfig`.

以下是为该函数添加的详细中文注释, 以及关于它的含义、作用和执行时机的深度解析:

### 代码中文注释

```go
// parseConfig 是插件的配置解析与初始化回调函数.
// [入参]:
//   - json: 通过 gjson 解析后的原始 JSON 配置对象, 代表用户在 Higress 控制台配置该插件时填写的 JSON 参数.
//   - c: 一个指向自定义结构体 PluginConfig 的指针, 解析出来的配置会保存在该结构体中, 供请求处理阶段读取.
//   - log: Higress 提供的日志打印工具.
// [返回值]: 如果配置不合法或 Redis 初始化失败, 则返回 error, 此时插件将拒绝加载该配置.
func parseConfig(json gjson.Result, c *PluginConfig, log log.Log) error {

	// 1. 获取并校验 Redis 的服务名称 (通常对应 K8s 中的 Service 域名或 Higress 的 DNS Egress 域名)
	c.RedisInfo.ServiceName = json.Get("redis.serviceName").String()
	if c.RedisInfo.ServiceName == "" {
		return errors.New("redis service name must not be empty")
	}

	// 2. 获取并处理 Redis 服务的端口号
	c.RedisInfo.ServicePort = int(json.Get("redis.servicePort").Int())
	if c.RedisInfo.ServicePort == 0 {
		// 如果未配置端口号, 根据服务类型自动填充默认端口:
		// 如果服务名以 ".static" 结尾(Higress 内置的静态 IP 服务定义),
		// 则在 Wasm 环境中, 网关与其通信的逻辑端口默认为 80(再由网关在转发时转换为实际外部 Redis 端口).
		if strings.HasSuffix(c.RedisInfo.ServiceName, ".static") {
			c.RedisInfo.ServicePort = 80
		} else {
			// 如果是普通的 K8s 内部服务, 使用 Redis 的标准默认端口 6379.
			c.RedisInfo.ServicePort = 6379
		}
	}

	// 3. 读取 Redis 的鉴权信息、连接超时及选择的数据库索引
	c.RedisInfo.Username = json.Get("redis.username").String()
	c.RedisInfo.Password = json.Get("redis.password").String()
	c.RedisInfo.Timeout = int(json.Get("redis.timeout").Int())
	if c.RedisInfo.Timeout == 0 {
		c.RedisInfo.Timeout = 1000 // 默认连接超时时间为 1000 毫秒
	}
	c.RedisInfo.Database = int(json.Get("redis.database").Int())

	// 4. 硬编码定义从 LLM (大语言模型) 请求和响应中提取对话内容的 gjson 路径.
	// 这里默认适配的是 OpenAI 兼容接口标准:
	// - QuestionFrom.RequestBody: "messages.@reverse.0.content"
	//   利用 gjson 的 @reverse 语法, 将整个 messages 数组翻转并取第 0 个元素(即最后一条消息, 代表用户的最新提问).
	c.QuestionFrom.RequestBody = "messages.@reverse.0.content"

	// - AnswerValueFrom.ResponseBody: 针对非流式响应, 提取回答内容的 JSON 路径(OpenAI 格式: choices[0].message.content).
	c.AnswerValueFrom.ResponseBody = "choices.0.message.content"

	// - AnswerStreamValueFrom.ResponseBody: 针对流式(SSE)响应分片, 提取当前文字块的 JSON 路径(OpenAI 格式: choices[0].delta.content).
	c.AnswerStreamValueFrom.ResponseBody = "choices.0.delta.content"

	// 5. 获取或设置缓存 Key 的前缀, 用于区分 Redis 中的其他 Key(默认为配置中指定值或预设常量)
	c.CacheKeyPrefix = json.Get("cacheKeyPrefix").String()
	if c.CacheKeyPrefix == "" {
		c.CacheKeyPrefix = DefaultCacheKeyPrefix
	}

	// 6. 获取或设置识别用户身份的 HTTP 请求头. 通过该 Header 的值(如 JWT、Token等)作为用户隔离的 Key(默认为 Authorization).
	c.IdentityHeader = json.Get("identityHeader").String()
	if c.IdentityHeader == "" {
		c.IdentityHeader = "Authorization"
	}

	// 7. 设置多轮对话回填的最大历史轮数, 默认为 3 轮(即最近 3 次问答).
	c.FillHistoryCnt = int(json.Get("fillHistoryCnt").Int())
	if c.FillHistoryCnt == 0 {
		c.FillHistoryCnt = 3
	}

	// 8. 获取缓存生存周期(Time To Live, 单位通常为秒), 0 可能表示依赖默认或不自动过期.
	c.CacheTTL = int(json.Get("cacheTTL").Int())

	// 9. 基于 Higress Wasm Go SDK 初始化 Redis 客户端.
	// 这里的 NewRedisClusterClient 会向 Envoy 宿主环境声明该 Redis 对应的网关 Cluster.
	c.redisClient = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
		FQDN: c.RedisInfo.ServiceName,
		Port: int64(c.RedisInfo.ServicePort),
	})

	// 10. 初始化 Redis 客户端连接, 传递鉴权参数和超时时间.
	// 如果此时连接或配置校验失败, 将直接向上层抛出错误.
	return c.redisClient.Init(c.RedisInfo.Username, c.RedisInfo.Password, int64(c.RedisInfo.Timeout), wrapper.WithDataBase(c.RedisInfo.Database))
}
```

---

### 一、函数的含义与作用

1. 配置反序列化(Deserialization): 此函数充当控制面(如 Higress 控制台)与 Wasm 插件运行时的桥梁. 利用 `gjson` 库, 从控制面分发的 JSON 配置中读取 Redis 地址、端口、用户名密码、对话轮数等业务参数, 并组装到强类型的 Go 结构体 `c *PluginConfig` 中.

2. 缺省值兜底(Defaults Initialization): 对于非必填配置, 进行了合理的默认值填充(例如 Redis 默认端口 `6379`、默认历史轮数 `3` 轮、身份校验头默认 `Authorization` ), 以此保证插件开箱即用.

3. OpenAI 协议适配: 硬编码了针对 OpenAI 格式 API 的提取规则(如 `messages.@reverse.0.content`), 免去了用户手动配置请求体解析规则的麻烦.

4. Redis 代理初始化:
   * 注意: Wasm 容器无法像标准 Go 程序那样直接通过 TCP Socket 库(如 `go-redis`)去建立底层 TCP 连接.
   * 因此, 它调用了 Higress SDK 封装的 `wrapper.NewRedisClusterClient`. 该函数会让网关底层 Envoy 扮演代理, 向外部 Redis 发起数据交互.

---

### 二、函数的执行时机

`parseConfig` 并不会在每次 HTTP 请求到来时都执行. 它的执行时机有且仅有以下两个阶段:

1. 插件首次初始化加载时:
   * 在网关启动或路由首次绑定该插件时, Wasm 虚拟机被实例化, 配置被第一次下发给插件, 该函数随之被触发一次.

2. 插件配置被动态更新(热更新)时:
   * 运维人员通过 Higress 控制台修改了该插件的配置(例如调大了历史轮数 `fillHistoryCnt`, 或者修改了 Redis 的密码), 并点击发布.
   * 网关在不中断流量(对流量无损)的情况下, 会启动新的 Wasm 实例加载新配置并调用 `parseConfig`. 配置成功加载后, 流量会被平滑地切换到新配置的实例上, 随后旧实例被销毁.

------------------------------------------------------------------------------------------------------------------------------

Q: 假设你是一位精通 go 语言, 并且精通 higress 生态的高级开发专家, 我看了 higress 插件 ai-history 的源码, 其中在 main.go 里面有如下代码
```go
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {
	ctx.DisableReroute()
	contentType, _ := proxywasm.GetHttpRequestHeader("content-type")
	if !strings.Contains(contentType, "application/json") {
		log.Warnf("content is not json, can't process:%s", contentType)
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}
	// get identity key
	identityKey, _ := proxywasm.GetHttpRequestHeader(config.IdentityHeader)
	if identityKey == "" {
		log.Warnf("identity key is empty")
		return types.ActionContinue
	}
	identityKey = strings.ReplaceAll(identityKey, " ", "")
	ctx.SetContext(IdentityKey, identityKey)
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	// The request has a body and requires delaying the header transmission until a cache miss occurs,
	// at which point the header should be sent.
	return types.HeaderStopIteration
}
```
请为这个代码加上详细的注释, 说明代码的含义, 作用和执行的时机

这段 Go 语言代码是 Higress Wasm 插件中用于拦截和预处理 HTTP 请求头的回调函数 `onHttpRequestHeaders`.

它的主要职责是对请求头进行安全校验、获取用户身份特征、清理可能影响后续 Body 修改的 Headers, 并控制网关的过滤链, 暂停 Headers 的发送以等待后续对请求体(Body)的重写.

以下是为该函数添加的详细中文注释以及对代码含义、作用、执行时机的深度解析:

### 代码中文注释

```go
// onHttpRequestHeaders 是请求头处理阶段的回调函数.
// [入参]:
//   - ctx: 当前 HTTP 请求的上下文对象, 可用于在不同生命周期阶段传递数据或控制网关行为.
//   - config: 已经在 parseConfig 阶段解析好的插件全局配置.
//   - log: 日志打印工具.
// [返回值]: types.Action, 告知网关如何处理当前请求(是继续、暂停还是直接中断).
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {

	// 1. 禁用重定向路由重计算 (Re-route)
	// [作用]: 告知 Envoy 网关在当前请求被修改(如重写 Path 或 Headers)后, 无需重新评估和匹配路由规则,
	//        以此节省 CPU 资源并防止路由冲突, 从而优化网关性能.
	ctx.DisableReroute()

	// 2. 检查 Content-Type, 确保是 JSON 请求
	// [作用]: AI 对话服务(符合 OpenAI 标准)交互格式必须是 JSON. 如果是其他类型(如 GET、文件上传或表单请求),
	//        该插件无法识别和回填历史, 应当直接跳过处理.
	contentType, _ := proxywasm.GetHttpRequestHeader("content-type")
	if !strings.Contains(contentType, "application/json") {
		log.Warnf("content is not json, can't process:%s", contentType)

		// 显式告知网关"不要去读取和缓存请求体", 防止不必要的内存分配, 提升整体吞吐量.
		ctx.DontReadRequestBody()

		// 返回 ActionContinue, 让请求继续往后传递, 不执行当前插件后续的 Body 修改逻辑.
		return types.ActionContinue
	}

	// 3. 获取用户身份标识 (Identity Key)
	// [作用]: 从配置指定的 Header(默认是 Authorization)中提取用户的 Token 或 API Key.
	identityKey, _ := proxywasm.GetHttpRequestHeader(config.IdentityHeader)
	if identityKey == "" {
		// 如果未提供身份 Header, 无法对对话历史进行用户隔离, 因此直接放行(不记录也不回填历史).
		log.Warnf("identity key is empty")
		return types.ActionContinue
	}

	// 4. 清理并暂存身份标识
	// 去除字符串中多余的空格(例如剔除 Bearer 后面的空格等, 将其格式化为紧凑的 Key, 用于后续作为 Redis Key 的一部分).
	identityKey = strings.ReplaceAll(identityKey, " ", "")

	// 将处理好的 identityKey 存入 ctx 的上下文中.
	// Wasm 插件是生命周期驱动的, 这样后续在 `onHttpRequestBody`(请求体处理)和响应处理阶段, 就能直接取出这个 Key.
	ctx.SetContext(IdentityKey, identityKey)

	// 5. 移除 Accept-Encoding
	// [作用]: 强制移除客户端发来的压缩声明. 如果客户端声称支持 gzip/br, 上游 AI 大模型也可能会返回压缩后的响应体.
	//        由于该插件后续需要在 `onHttpStreamResponseBody` 中"明文解析"大模型返回的 SSE(流式响应)数据并记录历史,
	//        因此移除该 Header 可以强迫上游服务返回明文数据, 避免网关层消耗资源去解压.
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")

	// 6. 移除 Content-Length
	// [作用]: 移除原始请求体长度. 在随后的 `onHttpRequestBody` 阶段, 该插件会向 Redis 查询历史消息,
	//        并把多轮历史对话重写(追加插入)到原请求中.
	//        由于重写后的 Body 长度已经改变, 原本的 Content-Length 必然失效, 必须将其移除.
	//        后续网关发送请求给上游大模型时, 会自动重新计算并填充, 或使用 chunked 编码.
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	// 7. 返回暂停状态并继续等待数据
	// [作用]: 告诉 Envoy 暂停把当前的 Request Headers 发送给上游, 同时也不要传递给下一个 Filter.
	//        但是, 网关并不停止从连接中读取数据, 而是继续触发 Body 的处理流程.
	//        这使我们可以在"请求体处理阶段 (`onHttpRequestBody`)"通过 Redis 异步加载完对话历史并成功修改 Body 后,
	//        再将头和修改后的体一起发给下一个 Filter/上游大模型.
	return types.HeaderStopIteration
}
```

---

### 二、核心代码设计含义与作用

1. `ctx.SetContext(IdentityKey, identityKey)` 状态跨阶段流转
   由于 Wasm 插件处理请求是分阶段执行的(Headers 阶段 -> Body 阶段), 每个阶段对应的底层 Wasm Callback 是独立的. 这里的 `SetContext` 机制允许我们在 Header 阶段识别并提取出用户身份标识后暂时存在上下文容器中, 以便在随后的 `onHttpRequestBody`(去 Redis 查该用户的历史)和响应体处理阶段直接拿来用, 避免了在每个阶段都重复解析 Headers.

2. 移除 `Accept-Encoding` 和 `Content-Length`
   这是网关层做 Request/Response Rewrite(请求/响应重写) 的标准防御性设计.
   * 移除 `Accept-Encoding` 避免了响应体压缩, 从而在 Wasm 内部能以明文解析大模型回答.
   * 移除 `Content-Length` 避免了大模型服务在收到重写后的(通常体积变大)请求体后, 因为原 `Content-Length` 与实际体积不一致而直接报错或导致截断.

3. `types.HeaderStopIteration` 暂停控制
   在标准的网关处理中, Headers 只要解析完就会立即发往上游. 但此插件必须拦截并重写请求. 通过返回 `HeaderStopIteration`, 相当于拉下了"暂缓发送 Headers"的闸门, 等待 Body 数据传输并在 Body 处理阶段加载、拼接完 Redis 中的多轮历史对话后, Headers 才会与重组后的 Body 一起发给大模型服务.

---

### 三、函数的执行时机

1. 请求流向阶段(Request Path):
   在网关接收到客户端发起的 HTTP 请求时触发, 属于请求处理的第一个核心阶段.

2. 特定的数据就绪时点:
   具体是在客户端发送的 HTTP Request Headers(请求头)被 Higress 网关完全接收并解析完毕的那一刻. 此时, HTTP Request Body(请求体)可能还在网络传输中, 尚未被网关完全接收.

------------------------------------------------------------------------------------------------------------------------------

Q: 假设你是一位精通 go 语言, 并且精通 higress 生态的高级开发专家, 我看了 higress 插件 ai-history 的源码, 其中在 main.go 里面有如下代码
```go
func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log log.Log) types.Action {
	bodyJson := gjson.ParseBytes(body)
	if bodyJson.Get("stream").Bool() {
		ctx.SetContext(StreamContextKey, struct{}{})
	}
	identityKey := ctx.GetStringContext(IdentityKey, "")
	question := TrimQuote(bodyJson.Get(config.QuestionFrom.RequestBody).String())
	if question == "" {
		log.Debug("parse question from request body failed")
		return types.ActionContinue
	}
	ctx.SetContext(QuestionContextKey, question)
	err := config.redisClient.Get(config.CacheKeyPrefix+identityKey, func(response resp.Value) {
		if err := response.Error(); err != nil {
			log.Errorf("redis get  failed, err:%v", err)
			_ = proxywasm.ResumeHttpRequest()
			return
		}
		if response.IsNull() {
			log.Debugf("cache miss, identityKey:%s", identityKey)
			_ = proxywasm.ResumeHttpRequest()
			return
		}
		chatHistories := response.String()
		ctx.SetContext(ChatHistories, chatHistories)
		var chat []ChatHistory
		err := json.Unmarshal([]byte(chatHistories), &chat)
		if err != nil {
			log.Errorf("unmarshal chatHistories:%s failed, err:%v", chatHistories, err)
			_ = proxywasm.ResumeHttpRequest()
			return
		}
		path := ctx.Path()
		if isQueryHistory(path) {
			cnt := getIntQueryParameter("cnt", path, len(chat)/2) * 2
			if cnt > len(chat) {
				cnt = len(chat)
			}
			chat = chat[len(chat)-cnt:]
			res, err := json.Marshal(chat)
			if err != nil {
				log.Errorf("marshal chat:%v failed, err:%v", chat, err)
				_ = proxywasm.ResumeHttpRequest()
				return
			}
			_ = proxywasm.SendHttpResponseWithDetail(200, "OK", [][2]string{{"content-type", "application/json; charset=utf-8"}}, res, -1)
			return
		}
		fillHistoryCnt := getIntQueryParameter("fill_history_cnt", path, config.FillHistoryCnt) * 2
		currJson := bodyJson.Get("messages").String()
		var currMessage []ChatHistory
		err = json.Unmarshal([]byte(currJson), &currMessage)
		if err != nil {
			log.Errorf("unmarshal currMessage:%s failed, err:%v", currJson, err)
			_ = proxywasm.ResumeHttpRequest()
			return
		}
		finalChat := fillHistory(chat, currMessage, fillHistoryCnt)
		var parameter map[string]any
		err = json.Unmarshal(body, &parameter)
		if err != nil {
			log.Errorf("unmarshal body:%s failed, err:%v", body, err)
			_ = proxywasm.ResumeHttpRequest()
			return
		}
		parameter["messages"] = finalChat
		parameterJson, err := json.Marshal(parameter)
		if err != nil {
			log.Errorf("marshal parameter:%v failed, err:%v", parameter, err)
			_ = proxywasm.ResumeHttpRequest()
			return
		}
		log.Infof("start to replace request body, parameter:%s", string(parameterJson))
		_ = proxywasm.ReplaceHttpRequestBody(parameterJson)
		_ = proxywasm.ResumeHttpRequest()
	})
	if err != nil {
		log.Error("redis access failed")
		return types.ActionContinue
	}
	return types.ActionPause
}
```
请为这个代码加上详细的注释, 说明代码的含义, 作用和执行的时机

这段 Go 语言代码是 `ai-history` 插件中最核心的业务逻辑所在 - 请求体(Request Body)处理函数 `onHttpRequestBody`.

它的核心职责是: 拦截客户端发送给 AI 大模型的 Prompt 请求体, 解析出最新的提问; 然后异步向 Redis 查询该用户的历史对话记录; 接着将历史记录与当前新问题进行拼接(上下文回填); 最后重写请求体并发送给上游大模型服务. 此外, 它还承担了"直接查询并返回历史记录"的旁路接口功能.

以下是为该函数添加的详细中文注释以及对代码含义、作用、执行时机的深度解析:

### 代码中文注释

```go
// onHttpRequestBody 是请求体处理阶段的回调函数.
// [入参]:
//   - ctx: 包含请求元数据的上下文.
//   - config: 插件的全局配置.
//   - body: 完整的请求体原始字节流(在 Header 阶段之后, 网关已完整缓存了该 JSON 报文).
//   - log: 日志工具.
// [返回值]: types.Action(指示网关是暂停、继续还是终止请求).
func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log log.Log) types.Action {

	// 1. 解析请求体 JSON
	bodyJson := gjson.ParseBytes(body)

	// 2. 检测是否为流式响应请求
	// 如果用户请求参数中含有 `"stream": true`, 说明大模型将采用 SSE 流式返回.
	// 将该状态存入上下文, 以便在后续的"响应体处理阶段"决定是否按流式解析 SSE.
	if bodyJson.Get("stream").Bool() {
		ctx.SetContext(StreamContextKey, struct{}{})
	}

	// 3. 提取用户身份标识(从 Header 阶段存入的上下文中取出)
	identityKey := ctx.GetStringContext(IdentityKey, "")

	// 4. 解析并提取用户的最新提问 (Question)
	// 利用在 parseConfig 阶段定义好的 JSON Path(如 messages.@reverse.0.content)提取最后一个用户的提问
	question := TrimQuote(bodyJson.Get(config.QuestionFrom.RequestBody).String())
	if question == "" {
		log.Debug("parse question from request body failed")
		return types.ActionContinue // 提取失败则不干预, 直接让原始请求继续传递
	}

	// 暂存当前用户的问题到上下文, 后续在收到大模型回答时, 需要和此问题成对写入 Redis
	ctx.SetContext(QuestionContextKey, question)

	// 5. 异步向 Redis 查询该用户的历史对话
	// [注意]: Wasm 虚拟机在 Envoy 中是单线程或基于事件驱动执行的, 网络 I/O 必须是异步的.
	// 这里的 Get 方法会发起异步调用, 并立即返回. 我们传入一个回调函数(func(response resp.Value))来处理 Redis 返回的结果.
	err := config.redisClient.Get(config.CacheKeyPrefix+identityKey, func(response resp.Value) {

		// --- 以下代码为 Redis 响应到达后的回调逻辑(异步执行) ---

		// 5.1. Redis 查询异常处理
		if err := response.Error(); err != nil {
			log.Errorf("redis get failed, err:%v", err)
			_ = proxywasm.ResumeHttpRequest() // 恢复之前被暂停的 HTTP 请求(即使没有历史记录, 也要保证用户请求能发给 AI)
			return
		}

		// 5.2. 缓存未命中(说明该用户是第一次或者会话已过期)
		if response.IsNull() {
			log.Debugf("cache miss, identityKey:%s", identityKey)
			_ = proxywasm.ResumeHttpRequest() // 直接恢复请求(不拼接历史, 直接将原请求发送给 AI)
			return
		}

		// 5.3. 缓存命中, 解析历史对话
		chatHistories := response.String()
		ctx.SetContext(ChatHistories, chatHistories) // 暂存原始历史 JSON, 后续更新缓存时需要追加它

		var chat []ChatHistory
		err := json.Unmarshal([]byte(chatHistories), &chat)
		if err != nil {
			log.Errorf("unmarshal chatHistories:%s failed, err:%v", chatHistories, err)
			_ = proxywasm.ResumeHttpRequest()
			return
		}

		path := ctx.Path()
		// 5.4. 分支 A: 检测是否为"查询历史记录"的特殊接口请求
		if isQueryHistory(path) {
			// 从 URL 参数 cnt 中提取需要返回的历史轮数(未传则返回全部)
			cnt := getIntQueryParameter("cnt", path, len(chat)/2) * 2
			if cnt > len(chat) {
				cnt = len(chat)
			}
			// 截取最后 cnt 条消息
			chat = chat[len(chat)-cnt:]
			res, err := json.Marshal(chat)
			if err != nil {
				log.Errorf("marshal chat:%v failed, err:%v", chat, err)
				_ = proxywasm.ResumeHttpRequest()
				return
			}
			// 拦截此请求, 不发往后端, 直接向客户端返回 200 OK 以及 JSON 格式的历史对话数据
			_ = proxywasm.SendHttpResponseWithDetail(200, "OK", [][2]string{{"content-type", "application/json; charset=utf-8"}}, res, -1)
			return
		}

		// 5.5. 分支 B: 常规 AI 聊天请求, 执行历史上下文回填逻辑
		// 确定要回填的历史消息数上限(一条对话包含一问一答, 故轮数乘以 2)
		fillHistoryCnt := getIntQueryParameter("fill_history_cnt", path, config.FillHistoryCnt) * 2

		// 提取原请求中的当前 messages 列表
		currJson := bodyJson.Get("messages").String()
		var currMessage []ChatHistory
		err = json.Unmarshal([]byte(currJson), &currMessage)
		if err != nil {
			log.Errorf("unmarshal currMessage:%s failed, err:%v", currJson, err)
			_ = proxywasm.ResumeHttpRequest()
			return
		}

		// 5.6. 核心逻辑: 将 Redis 的历史对话合并到当前请求消息数组中(同时限制最大轮数)
		finalChat := fillHistory(chat, currMessage, fillHistoryCnt)

		// 5.7. 重构并替换原始请求体
		var parameter map[string]any
		err = json.Unmarshal(body, &parameter)
		if err != nil {
			log.Errorf("unmarshal body:%s failed, err:%v", body, err)
			_ = proxywasm.ResumeHttpRequest()
			return
		}

		// 用融合了历史记录的新 finalChat 数组替换原请求中的 "messages" 字段
		parameter["messages"] = finalChat
		parameterJson, err := json.Marshal(parameter)
		if err != nil {
			log.Errorf("marshal parameter:%v failed, err:%v", parameter, err)
			_ = proxywasm.ResumeHttpRequest()
			return
		}

		// 调用 Proxy-Wasm 底层 API, 用新构造的 JSON 覆盖网关中当前的请求体
		log.Infof("start to replace request body, parameter:%s", string(parameterJson))
		_ = proxywasm.ReplaceHttpRequestBody(parameterJson)

		// 5.8. 关键步骤: 重写完毕, 通知网关恢复向后传输 HTTP 请求
		_ = proxywasm.ResumeHttpRequest()
	})

	// 6. Redis 客户端发起请求失败的处理
	if err != nil {
		log.Error("redis access failed")
		return types.ActionContinue // 客户端连接发起阶段报错, 直接让原请求继续, 避免服务被拖垮
	}

	// 7. 返回 Pause 暂停指令
	// [核心设计]: 因为去 Redis 获取数据是异步网络 I/O. 当前 Go 协程无法同步等待 Redis 返回.
	// 通过返回 ActionPause, 我们指示 Envoy: "先别把请求发送给大模型, 帮我把请求暂停住. 等我异步拿到 Redis 结果并完成 Body 重写后, 我会在回调函数里主动调用 ResumeHttpRequest 来唤醒你."
	return types.ActionPause
}
```

---

### 二、核心机制与作用解析

1. 异步暂停与恢复(`ActionPause` / `ResumeHttpRequest`)

   Wasm 虚拟机是事件驱动的. 由于 Redis 查询属于网络 I/O 操作, 在单线程模型下如果采用阻塞(Synchronous)等待, 会导致网关的整个 Worker 线程卡死.

   * 作用: 代码在此处优雅地返回了 `types.ActionPause`, 告诉网关"先暂停这个请求". 当 Redis 的数据在另一个事件循环中返回并触发回调时, 回调函数完成 JSON 的拼接与替换, 再通过调用 `proxywasm.ResumeHttpRequest()` 告诉网关可以继续转发了. 这是网关高性能异步处理的经典设计.

2. 旁路接口设计(`/v1/chat/history`)

   * 作用: 该插件不仅作为过滤器拦截请求, 还可以作为一个独立的"虚拟服务". 在代码中通过 `isQueryHistory(path)` 检查. 如果是查询历史的请求, 它不会将请求发给后端的 AI 模型服务, 而是直接调用 `proxywasm.SendHttpResponseWithDetail` 封包并返回, 实现了一套逻辑同时完成"历史注入"和"历史查询".

3. 历史回填(Context Injection)

   * 作用: 大模型(LLM)本身是无状态(Stateless)的. 要实现连续对话, 每次调用必须把之前的问答都作为上下文发过去.

   * 此处的 `fillHistory` 结合 `ReplaceHttpRequestBody` 实现了这一无感知代理层. 用户的客户端只需发送最新的一句话, 网关自动在 Redis 里拿出前几轮对话组合好, 送给大模型.

---

### 三、函数的执行时机

1. 请求流向阶段(Request Path):

   位于请求链路的核心位置. 它在 `onHttpRequestHeaders` 执行成功、并且 Headers 已经被挂起(StopIteration)之后触发.

2. 数据就绪时点:

   当客户端发送的 HTTP Request Body(请求体, 即包含当前 Prompt 的 JSON 数据)被 Higress 网关完全接收并缓存完毕时触发. 在这个时点, 请求头和请求体都已经整装待发, 正是对其进行拦截、解包并进行最后重写的最合适时机.


------------------------------------------------------------------------------------------------------------------------------

Q: 假设你是一位精通 go 语言, 并且精通 higress 生态的高级开发专家, 我看了 higress 插件 ai-history 的源码, 其中在 main.go 里面有如下代码
```go
func fillHistory(chat []ChatHistory, currMessage []ChatHistory, fillHistoryCnt int) []ChatHistory {
	userInputCnt := 0
	for i := 0; i < len(currMessage); i++ {
		if currMessage[i].Role == "user" {
			userInputCnt++
		}
	}
	if userInputCnt > 1 {
		return currMessage
	}
	if fillHistoryCnt > len(chat) {
		fillHistoryCnt = len(chat)
	}
	finalChat := append(chat[len(chat)-fillHistoryCnt:], currMessage...)
	return finalChat
}
```
请为这个代码加上详细的注释, 说明代码的含义, 作用和执行的时机

这段 Go 语言代码是 `ai-history` 插件中的一个纯工具函数(Helper Function) `fillHistory`.

它的核心职责是: 根据合理的限制规则, 将从 Redis 中查询出来的历史对话记录与当前用户发送的新消息进行裁剪与拼接, 组装出最终发给大模型(LLM)的完整上下文(Context)数组. 该函数包含了一层关键的"防御性设计", 能有效避免客户端自主携带历史记录时的重复填充问题.

以下是为该函数添加的详细中文注释以及对代码含义、作用、执行时机的深度解析:

### 代码中文注释

```go
// fillHistory 用于合并历史对话与当前消息.
// [入参]:
//   - chat: 从 Redis 中读取出来的该用户历史对话切片(ChatHistory 结构体数组, 包含过往所有的 user 和 assistant 对话).
//   - currMessage: 客户端本次请求里携带的消息切片(在标准 OpenAI 接口中, 通常是 messages 字段对应的内容).
//   - fillHistoryCnt: 需要回填的最大消息条数. 例如配置的轮数是 3 轮, 则最大条数为 6 条(3 问 + 3 答).
// [返回值]: 合并和裁剪后的最新完整消息切片, 将被替换到请求体中.
func fillHistory(chat []ChatHistory, currMessage []ChatHistory, fillHistoryCnt int) []ChatHistory {

	// 1. 统计当前客户端请求体中 "user" 角色(即用户提问)的数量.
	userInputCnt := 0
	for i := 0; i < len(currMessage); i++ {
		if currMessage[i].Role == "user" {
			userInputCnt++
		}
	}

	// 2. 防御性设计: 判断客户端是否已经自行管理并携带了历史上下文.
	// 如果 currMessage 里的 "user" 消息多于 1 条, 意味着客户端本身已经把多轮历史对话包装在请求体里发过来了.
	// 在这种情况下, 如果插件强行把 Redis 里的历史再拼接上去, 会导致上下文出现"双重重复".
	// [处理]: 直接返回客户端原本的消息体 currMessage, 不做任何 Redis 历史回填.
	if userInputCnt > 1 {
		return currMessage
	}

	// 3. 边界值处理: 如果需要填充的数量超过了 Redis 中实际存储的历史总数,
	// 则将填充长度限制为 Redis 历史的最大实际长度, 防止切片截取时发生越界(Index out of range).
	if fillHistoryCnt > len(chat) {
		fillHistoryCnt = len(chat)
	}

	// 4. 执行拼接.
	// - chat[len(chat)-fillHistoryCnt:]: 从 Redis 历史数组中, 截取最后几条消息(例如只取最近的 3 问 3 答, 排在前面的更旧历史被丢弃).
	// - append(..., currMessage...): 将截取出的历史与当前客户端发送的新提问(currMessage)进行前后拼接.
	//   拼接顺序是: [旧对话1, 旧对话2, ..., 本次新问题].
	finalChat := append(chat[len(chat)-fillHistoryCnt:], currMessage...)

	// 5. 返回拼接完成后的消息数组.
	return finalChat
}
```

---

### 二、核心设计含义与作用

该函数虽然简短, 但在多轮对话管理中起到了决策过滤器和滑动窗口裁剪的作用:

1. 防重填检测(`userInputCnt > 1`)

   * 作用: 兼容不同的客户端. 有些客户端(比如某些高度定制的 Web Chat UI)本身会在本地维护对话历史, 并在每次请求时把全量历史放到 `messages` 数组里打包发过来. 如果不做此项检测, 网关会将 Redis 中的历史与客户端自带的历史拼接在一起, 导致大模型收到一堆语无伦次的重复上下文. 当检测到 `userInputCnt > 1` 时, 直接放弃注入, 极大提升了插件的兼容性.

2. 滑动窗口裁剪(`chat[len(chat)-fillHistoryCnt:]`)

   * 作用: 限制多轮对话的长度. 大模型的 Token 费用是按量计费的, 且模型能处理的上下文窗口也是有限的. 我们不能无节制地把过去所有的历史全部塞给大模型. 通过这个切片操作, 插件只保留了最近 `fillHistoryCnt` 条(即最新的 N 轮)消息, 旧的消息会被自动"滑出"窗口.

---

### 三、函数的执行时机

1. 执行阶段:

   在 请求阶段(Request Path) 执行.

2. 具体的业务时点:

   该函数作为一个纯计算性质的本地辅助函数, 它在 `onHttpRequestBody` 函数的回调中同步执行. 执行时机精确发生在:

   * 网关已经接收到完整的请求体(Body).

   * 插件已经成功异步获取到 Redis 中的历史数据.

   * 且判定当前请求不是 `isQueryHistory`(查询历史记录接口), 而是需要发送给 AI 服务的常规聊天请求时.

   * 在对 JSON 进行重构序列化(`json.Marshal`)并替换请求体(`proxywasm.ReplaceHttpRequestBody`)之前, 该函数被调用以计算出最新的 messages 列表.


------------------------------------------------------------------------------------------------------------------------------

Q: 假设你是一位精通 go 语言, 并且精通 higress 生态的高级开发专家, 我看了 higress 插件 ai-history 的源码, 其中在 main.go 里面有如下代码
```go
func getIntQueryParameter(name string, path string, defaultValue int) int {
	// 解析 URL
	parsedURL, err := url.ParseRequestURI(path)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return defaultValue
	}

	// 获取查询参数
	values := parsedURL.Query()

	// 获取特定的查询参数 "defaultValue"
	queryStr := values.Get(name)
	if queryStr == "" {
		return defaultValue
	}
	num, err := strconv.Atoi(queryStr)
	if err != nil {
		return defaultValue
	}
	return num
}
```
请为这个代码加上详细的注释, 说明代码的含义, 作用和执行的时机

这段 Go 语言代码是 `ai-history` 插件中的一个通用工具函数 `getIntQueryParameter`.

它的核心职责是: 从传入的 HTTP 请求路径(URI)中安全地提取出指定名称的整数类型(`int`)查询参数(Query Parameter). 如果在解析、查找或转换过程中发生任何错误, 它将安全地返回调用方提供的默认值.

以下是为该函数添加的详细中文注释以及对代码含义、作用、执行时机的深度解析:

### 代码中文注释

```go
// getIntQueryParameter 用于从 HTTP 请求路径中安全地提取并解析整型的 Query 参数.
// [入参]:
//   - name: 要查找的 Query 参数名称(例如 "cnt" 或 "fill_history_cnt").
//   - path: 完整的 HTTP 请求路径(例如 "/v1/chat/completions?fill_history_cnt=5").
//   - defaultValue: 兜底默认值. 如果在路径中未找到该参数, 或参数格式非法, 将直接返回此默认值.
// [返回值]: 解析成功的整型数值或指定的默认值.
func getIntQueryParameter(name string, path string, defaultValue int) int {

	// 1. 利用 Go 标准库中的 net/url 解析完整的请求 URI.
	// ParseRequestURI 会解析路径和后面的 Query 字符串(如 ?key=val).
	parsedURL, err := url.ParseRequestURI(path)
	if err != nil {
		// 注意: 在 Wasm 环境中, fmt.Println 的输出通常会定向到 Envoy 容器的标准输出,
		// 相比于使用 SDK 的 log.Log, 这种写法较为底层, 但同样能起到记录错误的作用.
		fmt.Println("Error parsing URL:", err)
		return defaultValue
	}

	// 2. 解析并获取 Query 参数键值对映射表(url.Values 实际上是 map[string][]string)
	values := parsedURL.Query()

	// 3. 根据指定的名称获取参数值(Get 方法默认获取同名参数列表中的第一个值)
	queryStr := values.Get(name)

	// 4. 如果客户端请求中未携带该查询参数, 直接安全地返回配置中的默认值
	if queryStr == "" {
		return defaultValue
	}

	// 5. 将字符串类型的数值(如 "5")转换为整型(int)
	num, err := strconv.Atoi(queryStr)
	if err != nil {
		// 如果转换失败(例如用户故意输入了非数字格式 ?fill_history_cnt=abc),
		// 则放弃转换, 并安全地返回默认值, 防止插件因类型强转或错误数据导致 panic.
		return defaultValue
	}

	// 6. 返回解析成功的整数
	return num
}
```

---

### 二、核心作用与设计目的

该函数在 `ai-history` 插件中起到了动态配置覆盖(Dynamic Parameter Overriding)的作用.

在 Wasm 插件中, 虽然我们在全局配置(`PluginConfig`)里定义了固定的默认行为(例如默认回填 3 轮对话), 但不同用户、不同客户端在不同场景下对历史记录的需求是不同的.

* 作用一(历史查询控制): 当用户访问 `/v1/chat/history?cnt=5` 查询历史记录时, 该工具函数被调用来解析 `cnt` 参数. 它允许客户端自主决定要一次性读取多少条过往的对话(如果未传, 则默认为全部历史).

* 作用二(回填轮数控制): 当用户请求大模型 `/v1/chat/completions?fill_history_cnt=4` 时, 该工具函数用于解析 `fill_history_cnt`. 这使得客户端可以在 HTTP 请求中通过 Query 参数临时覆盖插件在控制台里配置的全局回填轮数, 实现请求级别的精细化控制.

通过这种设计, 插件既有全局的兜底配置(`defaultValue`), 又赋予了客户端根据实际业务灵活调整的能力.

---

### 三、函数的执行时机

1. 执行阶段:

   在请求阶段(Request Path) 执行.

2. 具体的业务时点:

   该函数作为一个纯计算性质的本地工具函数, 是在 `onHttpRequestBody`(请求体处理阶段)内同步执行的. 具体的触发时点为:

   * 网关已经完整接收到了 HTTP 的请求体.

   * 插件已经通过异步调用从 Redis 获得了该用户的历史对话.

   * 在通过 `ctx.Path()` 获取到当前请求的完整 URI 之后, 开始处理回填(或直接返回历史)逻辑之前, 会同步调用该函数来计算应当取多少条历史记录进行操作.

------------------------------------------------------------------------------------------------------------------------------

Q: 假设你是一位精通 go 语言, 并且精通 higress 生态的高级开发专家, 我看了 higress 插件 ai-history 的源码, 其中在 main.go 里面有如下代码
```go
func processSSEMessage(ctx wrapper.HttpContext, config PluginConfig, sseMessage string, log log.Log) string {
	content := ""
	for _, chunk := range strings.Split(sseMessage, "\n\n") {
		subMessages := strings.Split(chunk, "\n")
		var message string
		for _, msg := range subMessages {
			if strings.HasPrefix(msg, "data:") {
				message = msg
				break
			}
		}
		if len(message) < 6 {
			log.Errorf("invalid message:%s", message)
			return content
		}
		// skip the prefix "data:"
		bodyJson := message[5:]
		if gjson.Get(bodyJson, config.AnswerStreamValueFrom.ResponseBody).Exists() {
			tempContentI := ctx.GetContext(AnswerContentContextKey)
			if tempContentI == nil {
				content = TrimQuote(gjson.Get(bodyJson, config.AnswerStreamValueFrom.ResponseBody).Raw)
				ctx.SetContext(AnswerContentContextKey, content)
			} else {
				append := TrimQuote(gjson.Get(bodyJson, config.AnswerStreamValueFrom.ResponseBody).Raw)
				content = tempContentI.(string) + append
				ctx.SetContext(AnswerContentContextKey, content)
			}
		} else if gjson.Get(bodyJson, "choices.0.delta.content.tool_calls").Exists() {
			// TODO: compatible with other providers
			ctx.SetContext(ToolCallsContextKey, struct{}{})
		}
		log.Debugf("unknown message:%s", bodyJson)
	}
	return content
}
```
请为这个代码加上详细的注释, 说明代码的含义, 作用和执行的时机

这段 Go 语言代码是 `ai-history` 插件中处理流式响应(Server-Sent Events, 简称 SSE)的核心解析函数 `processSSEMessage`.

它的主要职责是: 当大模型以流式(Streaming)方式吐出回答时, 网关会拦截到一包包碎片化的数据(Tokens). 该函数负责将这些符合 SSE 协议的碎片数据进行分包、过滤、解析, 并从中提取出模型每次生成的文字片段, 然后累加存入上下文中, 最终拼凑成一个完整的大模型回复.

以下是为该代码添加的详细中文注释, 以及对其含义、作用和执行时机的深度解析:

### 代码中文注释

```go
// processSSEMessage 专门用于解析并聚合大模型返回的 SSE(流式)消息.
// [入参]:
//   - ctx: 当前 HTTP 请求的上下文, 用于跨阶段存储拼接中的完整回答.
//   - config: 插件全局配置(内含要解析的 JSON Path).
//   - sseMessage: 本次网关收到的、转化为字符串的流式数据片段.
//   - log: 日志工具.
// [返回值]: 本次解析并累加后的最新回复文本.
func processSSEMessage(ctx wrapper.HttpContext, config PluginConfig, sseMessage string, log log.Log) string {
	content := ""

	// 1. SSE 规范中, 不同的事件数据块(Event Blocks)之间是用双换行符 "\n\n" 进行分隔的.
	//    这里通过 "\n\n" 对可能包含多条 SSE 消息的数据进行切分.
	for _, chunk := range strings.Split(sseMessage, "\n\n") {

		// 2. 一个 SSE 数据块内部可能包含多行(如 id: xxx\nevent: xxx\ndata: {...}).
		//    我们需要通过单换行符 "\n" 进行切分, 找出真正包含业务数据的 data 行.
		subMessages := strings.Split(chunk, "\n")
		var message string
		for _, msg := range subMessages {
			// 过滤出以 "data:" 开头的行, 这行才是大模型输出的 JSON 载荷
			if strings.HasPrefix(msg, "data:") {
				message = msg
				break
			}
		}

		// 3. 安全校验: 如果拿到的 data 行长度不足 6("data:" 占 5 个字符, 后面至少需要有内容), 则视为无效消息并跳过.
		//    注: 流式结束标志通常是 "data: [DONE]", 长度为 12, 也会通过此校验.
		if len(message) < 6 {
			log.Errorf("invalid message:%s", message)
			return content
		}

		// 4. 裁剪掉前缀 "data:"(占 5 个字符: d-a-t-a-:), 获取纯 JSON 字符串
		bodyJson := message[5:]

		// 5. 判断当前 JSON 中是否存在大模型输出的文字片段(Delta Content).
		//    根据 parseConfig, config.AnswerStreamValueFrom.ResponseBody 对应的路径通常是 "choices.0.delta.content".
		if gjson.Get(bodyJson, config.AnswerStreamValueFrom.ResponseBody).Exists() {

			// 5.1. 从上下文(Context)中取出之前批次中已经拼接好的历史文本
			tempContentI := ctx.GetContext(AnswerContentContextKey)

			if tempContentI == nil {
				// 如果是第一条吐出的文本:
				// 解析提取出第一个字, 并去掉多余的双引号(TrimQuote), 然后首次存入上下文
				content = TrimQuote(gjson.Get(bodyJson, config.AnswerStreamValueFrom.ResponseBody).Raw)
				ctx.SetContext(AnswerContentContextKey, content)
			} else {
				// 如果不是第一条文本:
				// 解析提取出当前的字片段(append), 将其与之前已经累加的历史文本(tempContentI)进行字符串拼接,
				// 然后将更新后的完整文本重新写回上下文.
				append := TrimQuote(gjson.Get(bodyJson, config.AnswerStreamValueFrom.ResponseBody).Raw)
				content = tempContentI.(string) + append
				ctx.SetContext(AnswerContentContextKey, content)
			}

		} else if gjson.Get(bodyJson, "choices.0.delta.content.tool_calls").Exists() {
			// 6. 防御性检测: 如果检测到大模型当前的输出并不是普通的文本, 而是在执行"工具调用"(Tool Calls/Function Calling),
			//    则在上下文中打上一个 ToolCalls 标记.
			//    这是因为工具调用属于中间状态的系统 JSON, 不需要也不应该作为标准的"用户对话历史"被缓存到 Redis 中.
			// TODO: 后续可兼容其他不同大模型厂商的 Tool Calls 路径格式
			ctx.SetContext(ToolCallsContextKey, struct{}{})
		}

		// 7. 打印调试日志, 记录解析出的 JSON 内容.
		log.Debugf("unknown message:%s", bodyJson)
	}

	// 返回本次处理完毕后, 累计拼接得到的回答文本.
	return content
}
```

---

### 二、核心机制与作用解析

1. 解决流式数据"碎片化"问题(Token Aggregation)

   当 AI 聊天设置为流式输出时, 上游大模型不会一次性返回完整段落, 而是每产生一个或几个字就向客户端发送一个 SSE 数据包.

   * 痛点: 网关只充当管道, 它每次收到一部分网络报文, 就必须立刻转发给客户端. 这就导致网关在正常的流传输过程中, 拿不到一个"完整"的回答, 从而无法将其存入 Redis.

   * 解决方案: 此函数通过 `AnswerContentContextKey` 建立了一个内存"寄存器". 每当有新的数据片经过网关时, 网关一边把数据转发给客户端, 一边调用此函数把新字拼接到"寄存器"中. 当整个流结束时, 寄存器里就存放了最终完整的回答, 随后便可以将其一次性写入 Redis.

2. 区分内容输出与工具调用(Tool Call Detection)

   * 作用: 当大模型认为需要调用外部 API 时, 它输出的内容会是结构化的 `tool_calls` JSON, 而不是可读的文本. 这段代码通过检测 `choices.0.delta.content.tool_calls`, 能够及时感知到模型处于 Agent 决策阶段, 并在上下文中打上标记, 避免把模型的内部 Tool 调用代码当做"聊天记录"错误地存档到 Redis 中.

---

### 三、函数的执行时机

1. 执行阶段:

   在响应流向阶段(Response Path) 触发.

2. 具体的业务时点:

   当上游大模型服务(LLM)开始向客户端回传流式(SSE)响应体时.

   * 具体来说, 每当 Higress 网关从网络连接中接收到一部分响应数据分片(Chunk), 网关就会触发 `onHttpStreamResponseBody` 回调.

   * 在 `onHttpStreamResponseBody` 内部, 若判断当前响应为流式, 则会同步调用 `processSSEMessage` 来解析和聚合这部分新到达的数据块. 该过程会伴随整个流式传输过程, 被多次触发调用, 直到数据传输完全结束(接收到 `[DONE]` 标志).

------------------------------------------------------------------------------------------------------------------------------

Q: 假设你是一位精通 go 语言, 并且精通 higress 生态的高级开发专家, 我看了 higress 插件 ai-history 的源码, 其中在 main.go 里面有如下代码
```go
func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if strings.Contains(contentType, "text/event-stream") {
		ctx.SetContext(StreamContextKey, struct{}{})
	}
	return types.ActionContinue
}
```
请为这个代码加上详细的注释, 说明代码的含义, 作用和执行的时机

这段 Go 语言代码是 `ai-history` 插件中用于拦截和解析 HTTP 响应头(Response Headers)的回调函数 `onHttpResponseHeaders`.

它的核心职责是: 当大模型服务(Upstream)返回响应时, 通过检查其 `Content-Type` 响应头, 权威地判定这笔响应是否为流式传输(SSE, Server-Sent Events). 如果是, 则在上下文中记录该状态, 为后续正确解析响应体(Body)做准备.

以下是为该代码添加的详细中文注释以及对其含义、作用和执行时机的深度解析:

### 代码中文注释

```go
// onHttpResponseHeaders 是响应头处理阶段的回调函数.
// [入参]:
//   - ctx: 当前请求的上下文对象, 用于在响应阶段继续流转变量.
//   - config: 插件全局配置.
//   - log: 日志工具.
// [返回值]: types.Action(指示网关继续、暂停或中断响应流).
func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {

	// 1. 获取上游 AI 服务返回的 Content-Type 响应头
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")

	// 2. 检查 Content-Type 中是否包含 "text/event-stream"
	// "text/event-stream" 是标准的 Server-Sent Events (SSE) 协议内容类型, 通常是大模型流式输出(吐字效果)时的标配.
	if strings.Contains(contentType, "text/event-stream") {
		// 如果上游返回的确是流式响应, 则在上下文中记录一个 Stream 标记(StreamContextKey).
		// 这样后续的"流式响应体处理函数" `onHttpStreamResponseBody` 就能据此采用 SSE 协议去剥离和解析每个分片.
		ctx.SetContext(StreamContextKey, struct{}{})
	}

	// 3. 返回 ActionContinue, 允许网关继续处理并将响应头发送给客户端.
	// 这里不需要暂停或截断响应头, 因此直接向下传递.
	return types.ActionContinue
}
```

---

### 二、核心作用与设计目的

该函数虽小, 但在流式网关插件中具有非常关键的状态确认(Double-Check)作用:

1. 权威判定流式状态:

   * 痛点: 在之前的请求体处理阶段(`onHttpRequestBody`), 我们曾通过解析客户端请求中的 `"stream": true` 来猜测这笔请求是否为流式.

   * 实际问题: 客户端虽然请求了流式(`stream: true`), 但如果大模型服务本身报错(如触发限流 429、API Key 无效 401、或发生服务端错误 500), 上游服务器实际上返回的是普通的 JSON 报错信息(`application/json`), 而不是流.

   * 解决方案: 此函数在大模型服务开始回包时, 以响应头中的 `Content-Type` 为准. 只有当上游大模型确实返回了 `text/event-stream`, 才权威地激活 `StreamContextKey`. 这能有效避免因 AI 服务报错而导致后续流式解析器(`processSSEMessage`)去错误地解析 JSON 报错报文, 从而提高了插件的健壮性.

2. 零阻碍放行:

   * 此阶段不需要对响应头进行重写或拦截, 因此返回 `types.ActionContinue` 使得响应头能在网关层以极高的吞吐率直接透传给客户端, 保证首字延迟(TTFT - Time to First Token)不受到不必要的干扰.

---

### 三、函数的执行时机

1. 响应流向阶段(Response Path):

   位于响应处理链路的起点.

2. 具体的业务时点:
   当上游大模型服务处理完请求并产生响应, 其 HTTP 响应头(Response Headers)刚刚到达 Higress 网关的那一刻. 此时, 网关尚未收到任何实质性的响应体(Response Body)内容, 且响应头还未被发送给客户端.

------------------------------------------------------------------------------------------------------------------------------

Q: 假设你是一位精通 go 语言, 并且精通 higress 生态的高级开发专家, 我看了 higress 插件 ai-history 的源码, 其中在 main.go 里面有如下代码
```go
func onHttpStreamResponseBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isLastChunk bool, log log.Log) []byte {
	if ctx.GetContext(ToolCallsContextKey) != nil {
		// we should not cache tool call result
		return chunk
	}
	questionI := ctx.GetContext(QuestionContextKey)
	if questionI == nil {
		return chunk
	}
	if isQueryHistory(ctx.Path()) {
		return chunk
	}
	if !isLastChunk {
		stream := ctx.GetContext(StreamContextKey)
		if stream == nil {
			tempContentI := ctx.GetContext(AnswerContentContextKey)
			if tempContentI == nil {
				ctx.SetContext(AnswerContentContextKey, chunk)
				return chunk
			}
			tempContent := tempContentI.([]byte)
			tempContent = append(tempContent, chunk...)
			ctx.SetContext(AnswerContentContextKey, tempContent)
		} else {
			var partialMessage []byte
			partialMessageI := ctx.GetContext(PartialMessageContextKey)
			if partialMessageI != nil {
				partialMessage = append(partialMessageI.([]byte), chunk...)
			} else {
				partialMessage = chunk
			}
			messages := strings.Split(string(partialMessage), "\n\n")
			for i, msg := range messages {
				if i < len(messages)-1 {
					// process complete message
					processSSEMessage(ctx, config, msg, log)
				}
			}
			if !strings.HasSuffix(string(partialMessage), "\n\n") {
				ctx.SetContext(PartialMessageContextKey, []byte(messages[len(messages)-1]))
			} else {
				ctx.SetContext(PartialMessageContextKey, nil)
			}
		}
		return chunk
	}

	stream := ctx.GetContext(StreamContextKey)
	var value string
	if stream == nil {
		var body []byte
		tempContentI := ctx.GetContext(AnswerContentContextKey)
		if tempContentI != nil {
			body = append(tempContentI.([]byte), chunk...)
		} else {
			body = chunk
		}
		bodyJson := gjson.ParseBytes(body)

		value = TrimQuote(bodyJson.Get(config.AnswerValueFrom.ResponseBody).Raw)
		if value == "" {
			log.Warnf("parse value from response body failded, body:%s", body)
			return chunk
		}
	} else {
		if len(chunk) > 0 {
			var lastMessage []byte
			partialMessageI := ctx.GetContext(PartialMessageContextKey)
			if partialMessageI != nil {
				lastMessage = append(partialMessageI.([]byte), chunk...)
			} else {
				lastMessage = chunk
			}
			if !strings.HasSuffix(string(lastMessage), "\n\n") {
				log.Warnf("invalid lastMessage:%s", lastMessage)
				return chunk
			}
			// remove the last \n\n
			lastMessage = lastMessage[:len(lastMessage)-2]
			value = processSSEMessage(ctx, config, string(lastMessage), log)
		} else {
			tempContentI := ctx.GetContext(AnswerContentContextKey)
			if tempContentI == nil {
				return chunk
			}
			value = tempContentI.(string)
		}
	}
	saveChatHistory(ctx, config, questionI, value, log)
	return chunk
}
```
请为这个代码加上详细的注释, 说明代码的含义, 作用和执行的时机

这段 Go 语言代码是 `ai-history` 插件中最具技术含量的部分 - 响应体(Response Body)分片处理函数 `onHttpStreamResponseBody` [0].

它的核心职责是: 拦截大模型返回的 HTTP 响应体数据. 由于响应体通常很大(尤其是流式传输时会分片多次返回), 网关会多次调用该函数. 该函数通过精妙的网络分片重组算法, 解决了 TCP 数据截断(分片切割)的问题, 并分别在"非流式"和"流式"两种场景下, 完整还原出大模型的最终回答, 并在最后一个数据包到达时, 将其与用户的提问一起存储到 Redis.

以下是为该函数添加的详细中文注释以及对其含义、作用和执行时机的深度解析:

### 代码中文注释

```go
// onHttpStreamResponseBody 处理响应体分片(Chunk)的核心回调.
// [入参]:
//   - ctx: 包含请求元数据的上下文, 用于跨分片累加临时数据.
//   - config: 插件全局配置.
//   - chunk: 当前网络包收到的响应体字节片段(可能是几百字节到几 KB).
//   - isLastChunk: 标志位. 如果为 true, 说明当前数据片段已经是本次 HTTP 响应的最后一个包了.
//   - log: 日志工具.
// [返回值]: 必须返回处理后或原始的字节片段, 网关会将其转发给客户端.
func onHttpStreamResponseBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isLastChunk bool, log log.Log) []byte {

	// 1. 安全过滤 A: 如果是 Tool Calls(大模型工具/函数调用), 由于属于系统指令, 不作缓存, 直接原样返回
	if ctx.GetContext(ToolCallsContextKey) != nil {
		return chunk
	}

	// 2. 安全过滤 B: 如果在请求阶段没有成功提取到用户的 Question(提问), 无法建立问答对, 直接放行
	questionI := ctx.GetContext(QuestionContextKey)
	if questionI == nil {
		return chunk
	}

	// 3. 安全过滤 C: 如果是查询历史记录的接口路由, 直接放行不记录
	if isQueryHistory(ctx.Path()) {
		return chunk
	}

	// ==========================================
	// [阶段一]: 处理非最后一个数据包(!isLastChunk)
	// ==========================================
	if !isLastChunk {
		stream := ctx.GetContext(StreamContextKey)

		// 情况 1.1: 常规"非流式"响应(一次性返回完整 JSON, 但受 TCP 分包影响, 当前还不是最后一个包)
		if stream == nil {
			// 在上下文中初始化或累加响应体原始字节数组(AnswerContentContextKey)
			tempContentI := ctx.GetContext(AnswerContentContextKey)
			if tempContentI == nil {
				ctx.SetContext(AnswerContentContextKey, chunk)
				return chunk
			}
			tempContent := tempContentI.([]byte)
			tempContent = append(tempContent, chunk...)
			ctx.SetContext(AnswerContentContextKey, tempContent)

		} else {
			// 情况 1.2: 大模型"流式"响应(SSE text/event-stream 模式)
			// [核心技术难点]: TCP 分片可能正好切割在一条 SSE data JSON 数据的中间(例如: 上一包是 "data: {\"con", 下一包是 "tent\": \"hi\"}").
			// 此时我们不能直接解析, 必须使用缓冲池(PartialMessageContextKey)拼接未读完的数据.

			var partialMessage []byte
			partialMessageI := ctx.GetContext(PartialMessageContextKey)
			if partialMessageI != nil {
				// 如果历史分片里有残余的半条数据, 先取出并与新来的 chunk 拼接
				partialMessage = append(partialMessageI.([]byte), chunk...)
			} else {
				partialMessage = chunk
			}

			// SSE 协议规定每条完整的数据块均以双换行符 "\n\n" 结尾.
			// 我们通过 "\n\n" 切分这包数据.
			messages := strings.Split(string(partialMessage), "\n\n")
			for i, msg := range messages {
				// 除最后一个元素外, 前面的所有元素都保证是已经被 "\n\n" 包裹的、健全的 SSE 消息.
				// 立即调用 processSSEMessage 提取里面的文字并累加.
				if i < len(messages)-1 {
					processSSEMessage(ctx, config, msg, log)
				}
			}

			// 如果当前合并的数据不以 "\n\n" 结尾, 说明最后切分出的 messages[len-1] 是一个"半成品"(被截断了).
			// 我们将其存回 PartialMessageContextKey, 留到下一个 Chunk 接收时拼接起来.
			if !strings.HasSuffix(string(partialMessage), "\n\n") {
				ctx.SetContext(PartialMessageContextKey, []byte(messages[len(messages)-1]))
			} else {
				// 如果恰好是以 "\n\n" 结尾, 代表本批次数据完美消化, 清空残余缓冲池.
				ctx.SetContext(PartialMessageContextKey, nil)
			}
		}
		return chunk
	}

	// ==========================================
	// [阶段二]: 处理最后一个数据包(isLastChunk == true)
	// ==========================================
	stream := ctx.GetContext(StreamContextKey)
	var value string // 存放最终提取出的大模型文本答案

	// 情况 2.1: 非流式响应的最后一个包到达
	if stream == nil {
		var body []byte
		tempContentI := ctx.GetContext(AnswerContentContextKey)
		if tempContentI != nil {
			// 将之前累加的数据与这最后一个包拼起来, 形成完整的 Response Body JSON
			body = append(tempContentI.([]byte), chunk...)
		} else {
			body = chunk
		}

		// 解析完整的非流式 JSON, 提取出最终的文本回答内容
		bodyJson := gjson.ParseBytes(body)
		value = TrimQuote(bodyJson.Get(config.AnswerValueFrom.ResponseBody).Raw)
		if value == "" {
			log.Warnf("parse value from response body failded, body:%s", body)
			return chunk
		}

	} else {
		// 情况 2.2: 流式响应的最后一个包到达
		if len(chunk) > 0 {
			// 如果这最后一包里还有残留的字节数据
			var lastMessage []byte
			partialMessageI := ctx.GetContext(PartialMessageContextKey)
			if partialMessageI != nil {
				lastMessage = append(partialMessageI.([]byte), chunk...)
			} else {
				lastMessage = chunk
			}
			// 校验 SSE 流正常结束(必须以双换行结尾)
			if !strings.HasSuffix(string(lastMessage), "\n\n") {
				log.Warnf("invalid lastMessage:%s", lastMessage)
				return chunk
			}
			// 剥除末尾多余的两个换行符 \n\n, 解析并提取出流中最后几个词
			lastMessage = lastMessage[:len(lastMessage)-2]
			value = processSSEMessage(ctx, config, string(lastMessage), log)
		} else {
			// 如果这最后一个包没有内容(通常只是一个标记结束的空 Body 片段),
			// 则直接从上下文中取出此前早已实时累加拼接好的完整回答文本
			tempContentI := ctx.GetContext(AnswerContentContextKey)
			if tempContentI == nil {
				return chunk
			}
			value = tempContentI.(string)
		}
	}

	// 4. [终极目标达成]: 将提取完毕的完整问答对, 写入 Redis 存储历史.
	//    其中 questionI 为用户提问, value 为 AI 的回答内容.
	saveChatHistory(ctx, config, questionI, value, log)

	// 5. 必须返回原始 chunk 字节, 保障大模型返回给客户端的真实数据流不受任何阻塞和损耗.
	return chunk
}
```

---

### 二、核心机制解析

这段代码巧妙地解决了 Wasm 插件处理流式传输时的三大世界级设计难点:

1. TCP 分片重组(Network Slicing Reassembly)

   * 难点: TCP 是面向字节流的, 并没有"包边界". 流式 SSE 报文(如 `data: {"content": "Hello"}\n\n`)经常被网络拆包在任意位置.

   * 解决方案: 引入 `PartialMessageContextKey` 缓冲池, 结合 `strings.Split(..., "\n\n")`. 对不带 `\n\n` 后缀的末尾分片直接不予解析, 而是存入上下文, 等下一次 `chunk` 来临时重组. 这保证了即使网络传输非常不稳定, 插件依然可以百分之百准确地提取每一条 JSON 的内容.

2. 区分"流式 (Stream)"与"非流式 (Non-Stream)"的提取路线

   * 非流式: 不实时解析, 将所有分片累加在一起. 直到最后一包 `isLastChunk == true` 到达后, 合并一次性用 `gjson` 提取.

   * 流式: 不阻塞客户端, 每一个分片来临时, 提取里面的文字并实时存入上下文变量. 最后一包到达时, 直接取出来即可.

3. 双重校验拦截

   * 通过 `ToolCallsContextKey` 保证如果大模型正在进行 Agent 工具调用, 插件立刻放弃拦截和记录, 保障了对话记录的纯净和高性能.

---

### 三、函数的执行时机

1. 执行阶段:

   在响应阶段(Response Path) 触发.

2. 具体的业务时点与触发频次:

   * 非流式请求: 通常触发 1 ~ 2 次(视上游 AI 服务回包的大小而定).

   * 流式(SSE)请求: 在整个对话流传输期间会被触发数十次甚至上百次. 每次 Higress 网关在底层连接中收到一包来自大模型的 TCP 响应包时, 都会触发一次该函数, 直至收到大模型的 `data: [DONE]` 等流式终止包、且 `isLastChunk` 标志位由 Envoy 网关置为 `true`.


------------------------------------------------------------------------------------------------------------------------------

Q: 假设你是一位精通 go 语言, 并且精通 higress 生态的高级开发专家, 我看了 higress 插件 ai-history 的源码, 其中在 main.go 里面有如下代码
```go
func saveChatHistory(ctx wrapper.HttpContext, config PluginConfig, questionI any, value string, log log.Log) {
	question := questionI.(string)
	identityKey := ctx.GetStringContext(IdentityKey, "")
	var chat []ChatHistory
	chatHistories := ctx.GetStringContext(ChatHistories, "")
	if chatHistories != "" {
		err := json.Unmarshal([]byte(chatHistories), &chat)
		if err != nil {
			log.Errorf("unmarshal chatHistories:%s failed, err:%v", chatHistories, err)
			return
		}
	}
	chat = append(chat, ChatHistory{Role: "user", Content: question})
	chat = append(chat, ChatHistory{Role: "assistant", Content: value})
	if len(chat) > config.FillHistoryCnt*2 {
		chat = chat[len(chat)-config.FillHistoryCnt*2:]
	}
	str, err := json.Marshal(chat)
	if err != nil {
		log.Errorf("marshal chat:%v failed, err:%v", chat, err)
		return
	}
	log.Infof("start to Set history, identityKey:%s, chat:%s", identityKey, string(str))
	_ = config.redisClient.Set(config.CacheKeyPrefix+identityKey, string(str), nil)
	if config.CacheTTL != 0 {
		_ = config.redisClient.Expire(config.CacheKeyPrefix+identityKey, config.CacheTTL, nil)
	}
}
```
请为这个代码加上详细的注释, 说明代码的含义, 作用和执行的时机

这段 Go 语言代码是 `ai-history` 插件中用于归档与持久化对话记录的核心函数 `saveChatHistory`.

它的核心职责是: 当大模型最终回包完毕(最后一个分片接收并解析完成)后, 将本次新产生的"用户提问"与"大模型回答"打包为一组新的问答对, 追加到原本的历史记录中. 同时, 通过滑动窗口算法限制存储的历史记录总轮数, 防止 Redis 内存无限膨胀, 最后通过网关封装的 Redis 客户端将最新历史异步写回 Redis 缓存中, 并设置生存时间(TTL).

以下是为该函数添加的详细中文注释以及对其含义、作用和执行时机的深度解析:

### 代码中文注释

```go
// saveChatHistory 用于将最新的一轮问答记录追加合并, 并写回 Redis 缓存中.
// [入参]:
//   - ctx: HTTP 上下文, 用来提取请求和响应阶段暂存的数据.
//   - config: 插件全局配置(用于读取缓存 key 前缀、回填轮数、过期时间等).
//   - questionI: 暂存在上下文中的用户最新提问(any 类型, 使用时需类型断言为 string).
//   - value: 经过上游解析组装后, 大模型最终输出的完整文本回答.
//   - log: 日志工具.
func saveChatHistory(ctx wrapper.HttpContext, config PluginConfig, questionI any, value string, log log.Log) {
	// 1. 类型断言: 将 interface{} 类型的提问转换为 string
	question := questionI.(string)

	// 2. 从上下文中取出用户身份标识(用作 Redis 的 Key 区分不同的用户会话)
	identityKey := ctx.GetStringContext(IdentityKey, "")

	var chat []ChatHistory

	// 3. 从上下文中提取原有的历史对话记录(这是在请求阶段 'onHttpRequestBody' 异步从 Redis 查出来并暂存的)
	chatHistories := ctx.GetStringContext(ChatHistories, "")
	if chatHistories != "" {
		// 如果原有历史不为空, 将其反序列化为 Go 的 ChatHistory 结构体切片
		err := json.Unmarshal([]byte(chatHistories), &chat)
		if err != nil {
			log.Errorf("unmarshal chatHistories:%s failed, err:%v", chatHistories, err)
			return // 反序列化失败则直接中断, 避免写入损坏的历史数据
		}
	}

	// 4. 将本次最新的对话回合追加到切片末尾:
	// - 先追加用户的最新提问 (Role: "user")
	chat = append(chat, ChatHistory{Role: "user", Content: question})
	// - 后追加大模型的最新回答 (Role: "assistant")
	chat = append(chat, ChatHistory{Role: "assistant", Content: value})

	// 5. [滑动窗口限制]: 限制 Redis 中缓存的历史对话轮数.
	//    一轮完整的对话包含 1 次 user 提问和 1 次 assistant 回答, 因此最大消息条数为 FillHistoryCnt * 2.
	//    如果追加后超出该限制, 则利用切片操作丢弃掉最旧的消息, 只保留最近的消息片段, 实现缓存轮数的自动更替.
	if len(chat) > config.FillHistoryCnt*2 {
		chat = chat[len(chat)-config.FillHistoryCnt*2:]
	}

	// 6. 将合并及裁剪后的最新历史记录序列化为 JSON 字符串
	str, err := json.Marshal(chat)
	if err != nil {
		log.Errorf("marshal chat:%v failed, err:%v", chat, err)
		return
	}

	log.Infof("start to Set history, identityKey:%s, chat:%s", identityKey, string(str))

	// 7. 将更新后的历史记录异步写入 Redis
	//    使用自定义前缀加上用户标识构成唯一的 Cache Key.
	//    第三个参数为回调函数, 此处传 nil 表示采用"发射后不管"(Fire-and-Forget)的模式异步执行.
	_ = config.redisClient.Set(config.CacheKeyPrefix+identityKey, string(str), nil)

	// 8. 刷新该 Cache Key 的生存时间 (TTL)
	//    若配置中指定了缓存生存时间(非 0), 则向 Redis 发送 EXPIRE 指令, 确保无活动会话能够按时自动清理, 释放 Redis 空间.
	if config.CacheTTL != 0 {
		_ = config.redisClient.Expire(config.CacheKeyPrefix+identityKey, config.CacheTTL, nil)
	}
}
```

---

### 二、核心作用解析

1. 会话生命周期闭环(Dialog Lifecycle Sink):

   该函数是整套"多轮对话"逻辑的最后一公里. 它将请求阶段提取的 User Prompt、响应阶段实时拼接的 Assistant Answer 以及之前缓存的 History 重新聚合在一起, 使之成为完整连贯的历史上下文链.

2. 主动式内存规整(Sliding Window Truncation):

   * 痛点: 如果不加以限制, 随着用户聊天频次的增加, 存储在 Redis 中的 JSON 数组会无限变大. 这不仅会导致 Redis 内存耗尽(OOM), 还会在下一次提取时对网关反序列化性能和网络带宽造成巨大负担.

   * 作用: 通过 `len(chat) - config.FillHistoryCnt * 2` 滑动窗口设计, 强行截断旧对话, 确保每个用户在 Redis 里的缓存大小保持恒定.

3. 异步连接性能保障(Async I/O Non-blocking):

   通过 `redisClient.Set` 和 `Expire` 异步执行且不配置 Callback(第三参数传 `nil`), 使得插件把写操作投递给网关的网络事件循环后即可立刻返回, 完全不阻塞网关向客户端吐回最后一个字符包, 保证了用户的极佳交互体验.

---

### 三、函数的执行时机

1. 执行阶段:

   在响应阶段(Response Path) 执行.

2. 具体的业务时点:

   该函数是在 `onHttpStreamResponseBody` 回调函数的逻辑终点处被调用 [0]. 其触发时点精密控制在:

   * 网关已经接收到上游大模型的最后一个数据包(`isLastChunk == true`), 即标志着本轮会话已彻底结束.

   * 插件已经将前面分散在各个网络包里的文本内容组装完毕, 拿到了大模型最终、最完整的回答(`value`).

   * 在网关即将完成请求清理并向客户端结束本轮 HTTP 响应之前, 执行异步归档操作 [0].

------------------------------------------------------------------------------------------------------------------------------


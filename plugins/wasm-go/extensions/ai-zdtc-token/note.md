
---------------------------------------------------------------------------------------------

假设你是一位精通 higress 及其插件开发的高级开发人员, 我开发了一个简单的 higress 插件, 使用如下配置部署插件
```
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  annotations:
    higress.io/wasm-plugin-description:
      Implement token-based rate limiting according
      to assigned fixed token, while also supporting token management capabilities,
      including querying, refreshing, and adjusting token.
    higress.io/wasm-plugin-icon: https://img.alicdn.com/imgextra/i1/O1CN018iKKih1iVx287RltL_!!6000000004419-2-tps-42-42.png
    higress.io/wasm-plugin-title: AI Token
  labels:
    higress.io/resource-definer: higress
    higress.io/wasm-plugin-built-in: "true"
    higress.io/wasm-plugin-category: ai
    higress.io/wasm-plugin-name: ai-token
    higress.io/wasm-plugin-version: 1.0.0
  name: ai-token
  namespace: higress-system
spec:
  defaultConfig:
    admin_consumer: consumer3
    admin_path: /quota
    redis:
      service_name: redis-service.default.svc.cluster.local
      service_port: 6379
      timeout: 2000
    redis_key_prefix: "chat_token:"
  defaultConfigDisable: true
  imagePullPolicy: IfNotPresent
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-token:20260707-193617-35df208a

```
我在 higress console 上查看插件时, 有如下报错
```
RequestURL: http://172.16.10.98:32333/v1/wasm-plugins/ai-token/config?ts=1783475815791
Headers: {"Accept":"application/json, text/plain, */*"}
```
请问这是什么原因, 应该如何解决

在 Higress 中, 当你在控制台(Higress Console)查看或配置 WASM 插件时, 控制台前端会向后端发送请求(如请求 `/v1/wasm-plugins/{plugin-name}/config`). 这个请求的目的是获取该插件的配置 Schema(即 OpenAPI V3 格式的数据模版), 以便在前端渲染出可视化的配置表单或进行配置项的合法性校验.

你遇到的报错, 其核心原因通常与插件类型声明以及配置 Schema 缺失有关.

---

### 原因分析

#### 1. 错误的内置属性声明(主要原因)

在你的 YAML 配置中, 含有如下 Label:

```yaml
labels:
  higress.io/wasm-plugin-built-in: "true"
```

* 原理解析: 当此标签设置为 `"true"` 时, Higress Console 会认为 `ai-token` 是一个 Higress 官方内置插件. 控制台后端会尝试从其内部的静态目录(例如 `/opt/data/defaultConfig` 或打包在 Console 镜像内部的内置配置文件夹)去读取该插件的 `spec.yaml` 规格文件.

* 问题所在: 由于 `ai-token` 是你自主开发的自定义插件, 控制台的内置插件库里并没有它. 因此, 控制台后端在查询配置 Schema 时会因为找不到内置规格文件而直接返回 `404 Not Found` 或 `500` 错误, 导致前端请求 `/config` 失败.

#### 2. 自定义 OCI 镜像缺少 `spec.yaml` 元数据(次要原因)

如果将 `built-in` 设置为 `"false"`, 控制台会尝试从你配置的 OCI 镜像 `oci://higress-registry...` 中提取元数据.

* 如果在构建该 Wasm 镜像时, 只简单地将 `main.wasm` 复制到了镜像(例如通过 `FROM scratch COPY plugin.wasm` 这种两行 Dockerfile), 那么该镜像中是不包含 `spec.yaml`(即配置 Schema 文件)的.

* 此时控制台无法从镜像中解析出配置字段, 同样会报错或退化为无法渲染表单.

---

### 解决方案

针对上述原因, 建议通过以下两种方式之一进行解决:

#### 方案一: 修改 Label 并在控制台重新注册为"自定义插件"(推荐, 最快速)

如果不想重新打包镜像, 可以将其标记为自定义插件, 利用控制台的手动配置能力规避报错.

1. 修改 WasmPlugin 的 Labels:

   将 `higress.io/wasm-plugin-built-in` 改为 `"false"`, 并将分类改为 `custom`:

   ```yaml
   labels:
     higress.io/resource-definer: higress
     higress.io/wasm-plugin-built-in: "false" # 改为 false
     higress.io/wasm-plugin-category: custom   # 改为 custom
     higress.io/wasm-plugin-name: ai-token
     higress.io/wasm-plugin-version: 1.0.0
   ```

2. 在 Higress 控制台注册:

   不要直接用 `kubectl apply` 强行部署未在控制台备案的插件.

   * 登录 Higress 控制台, 进入 插件市场 (Plugin Market).
   * 点击 自定义插件 / 创建 按钮.
   * 填写插件名称为 `ai-token`, 镜像地址填写 `oci://...`.
   * 提供 Schema 校验(可选): 在控制台创建时, 它会提供一个 Schema 编辑器. 你可以使用标准的 JSON Schema 声明你的配置字段(如 `redis`、`admin_consumer` 等), 或者直接留空, 留空后控制台会自动提供原生的 YAML 文本编辑器来让你编辑配置, 从而不会报错.

---

#### 方案二: 规范化打包 OCI 镜像(适合生产标准发布)

如果你希望保留内置式的体验, 让控制台能够自动解析你的插件配置并生成完美的表单, 需要按照 Higress WASM 插件镜像规范 重新打包你的 OCI 镜像.

1. 编写 `spec.yaml` 文件:

   在你的插件项目根目录下创建一个 `spec.yaml` 文件, 定义好前端渲染所需的 OpenAPI V3 格式的 Schema:

   ```yaml
   apiVersion: 1.0.0
   info:
     category: ai
     name: ai-token
     title: AI Token
     description: Implement token-based rate limiting...
   spec:
     phase: AUTHN
     priority: 100
     configSchema:
       openAPIV3Schema:
         type: object
         properties:
           admin_consumer:
             type: string
           admin_path:
             type: string
           redis:
             type: object
             properties:
               service_name:
                 type: string
               service_port:
                 type: integer
               timeout:
                 type: integer
           redis_key_prefix:
             type: string
   ```

2. 准备 `README.md`:

   至少提供一个简单的 `README.md` 描述插件如何使用.

3. 使用 `oras` 命令行工具进行多层打包推送:

   不能使用普通的单层 `docker build`. 需要使用 `oras` 工具, 将 `spec.yaml`、`README.md` 和编译好的 `plugin.wasm` 打包成多层格式的符合 Higress 规范的 OCI 镜像:

   ```bash
   # 打包 wasm 文件
   tar czvf plugin.tar.gz main.wasm

   # 使用 oras 工具推送到镜像仓库
   oras push oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-token:20260707-193617-35df208a \
     ./spec.yaml:application/vnd.module.wasm.spec.v1+yaml \
     ./README.md:application/vnd.module.wasm.doc.v1+markdown \
     ./plugin.tar.gz:application/vnd.oci.image.layer.v1.tar+gzip
   ```

打包好并重新部署后, Higress Console 再次请求 `/v1/wasm-plugins/ai-token/config` 时, 就能成功从 OCI 镜像中拉取并解析出 `spec.yaml`, 报错便会自然消失.

---------------------------------------------------------------------------------------------

Q: 假设你是一位精通 higress 及其插件开发的高级开发人员, 我现在想开发如下功能的插件:
1、higress 的上游服务是 vllm 部署的模型服务
2、用户经过 higress gateway 请求上游的模型服务, 在我的插件中将用户信息, 最终的模型输出的 输入 token 数量和输出 token 数量记录在 redis 中
代码的大致结构如下:
```go
```
第一, 请帮我检查一下这个大概的代码框架是否合理, 如果不合理, 请修改这个代码框架,
第二, 请帮我在每个函数中输出详细的请求参数的日志, 我想看看每个函数的输入到底是什么, 暂时不用实现具体的功能

你好! 作为一名长期从事 Higress 插件开发的开发人员, 我非常高兴能与你探讨这个实用的 AI 网关插件.

以下是对你提供的 Go 代码框架进行的专业评估, 以及为你改造后的、带有完整日志输出的代码实现.

---

### 第一部分: 代码框架合理性评估与修改建议

你提供的大体框架非常清晰, 已经具备了 Higress Go 插件的基本雏形. 但在针对 vLLM(大模型服务) 的对接场景下, 有以下几个关键问题需要优化和调整:

#### 1. 响应体处理机制: 避免同时注册非流式与流式 Hook

* 问题所在: 在你的 `init()` 函数中, 同时注册了 `wrapper.ProcessResponseBodyBy`(非流式)和 `wrapper.ProcessStreamingResponseBodyBy`(流式).
* 潜在风险: vLLM 的模型服务通常默认支持流式输出(Server-Sent Events, SSE, 例如 `stream: true`). 如果在插件中启用了非流式的 `ProcessResponseBodyBy`, Higress 会强制缓存(Buffer)整个响应体, 直到全部接收完毕才回调. 这会导致流式响应(SSE)完全失效, 客户端产生极高延迟, 大模型无法逐字输出.
* 解决方案: 在 LLM/vLLM 场景下, 强烈建议仅保留并注册流式响应 Hook `ProcessStreamingResponseBodyBy`. 该 Hook 非常灵活, 不仅能处理 SSE 流式响应, 对于非流式响应(`stream: false`), 它也会将其作为"单 chunk(且 `isLastChunk = true`)"进行正常处理.

#### 2. 结构体字段导出(Exported Fields)

* 问题所在: `PluginConfig` 结构体中的 `redisInfo` 和 `redisClient` 首字母为小写, 属于私有字段.
* 解决方案: 在 Go 语言中, 私有字段会限制外部包(如反射、JSON/YAML 解析库)对其进行读写或反序列化. 建议将它们改为大写: `RedisInfo` 和 `RedisClient`.

#### 3. 依赖引入

* 为了在 Hook 函数中读取 HTTP 请求/响应的 Header, 我们需要引入 `"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"` 包, 这样才能调用 `proxywasm.GetHttpRequestHeaders()` 等底层 WASM API.

---

### 第二部分: 优化后的完整代码(包含详细日志输出)

下面的代码修复了上述框架问题, 并在每个生命周期函数中实现了精细的参数与上下文日志打印. 你可以直接拿去编译, 并在 gateway 运行后观察各阶段的入参输出:

```go
```

### 调试该插件时的观察要点:

1. 对于非流式请求 (`"stream": false`): 你会在日志中看到 `onHttpStreamResponseBody` 只被调用一次, 且 `isLastChunk` 为 `true`, 此时 `chunk` 即为完整的响应 JSON. 你可以直接解析其中的 `usage.prompt_tokens` 和 `usage.completion_tokens`.

2. 对于流式请求 (`"stream": true`): 你会在日志中观察到 `onHttpStreamResponseBody` 被多次调用. 由于 vLLM 符合 OpenAI 规范, 它通常会在最后一个 chunk(即 `isLastChunk` 为 `true`, 或者内容包含 `[DONE]` 之前的一帧)中返回包含完整的 `usage` 统计信息. 通过日志排查这一帧的格式, 能非常方便你后续编写具体的 Token 提取和 Redis 存储逻辑.

---------------------------------------------------------------------------------------------

Q:
```go
type PluginConfig struct {
	// 首字母大写, 确保字段可导出, 便于反序列化与初始化
	RedisInfo   RedisInfo           `yaml:"redis" json:"redis"`
	RedisClient wrapper.RedisClient `yaml:"-" json:"-"`
}
```
其中 RedisClient 应该是私有字段, 不需要导出, 因为它只用于最后操作 redis, 不需要导出

您的考虑非常周到, 这个设计确实更加严谨.

确实, `redisClient` 仅作为插件运行期间与 Redis 交互的内部句柄, 它不需要外部包访问, 更不需要参与任何配置的反序列化过程(我们已经通过 `yaml:"-" json:"-"` 显式忽略了它).

由于我们的插件代码(包括配置解析和所有的 Hook 回调函数)都处于同一个 `package main` 中, Go 语言的包内可见性足够让我们在私有字段上进行赋值和调用. 将其保持为私有字段(首字母小写)有助于更好地隐藏实现细节, 使配置结构体的定义更加干净和内聚.

按照您的建议, 我们可以将结构体以及 `parseConfig` 中的初始化代码调整如下:

### 1. 修改后的结构体定义

```go
```

### 2. 对应的 `parseConfig` 赋值调整

在 `parseConfig` 中, 直接使用小写的 `config.redisClient` 进行初始化和配置:

```go
```

这样修改后, 代码结构更加符合 Golang 的最佳实践和封装原则. 后续在任何 Hook 阶段(如流式响应处理时), 你都可以直接通过 `config.redisClient` 来调用其方法(例如 `config.redisClient.IncrBy`、`config.redisClient.HSet` 等)来实现具体的 Token 计数功能.

---------------------------------------------------------------------------------------------

Q: 在上面打印日志的代码中, 请在每条日志的前面添加插件名称的标识, 这样有利于在复杂的 higress gateway 日志里面进行查找, 请完整输出所有的代码

在复杂的 Higress 网关和 Envoy 日志环境中, 为日志添加独一无二的标识(如插件名称前缀)能极大地提高排查问题的效率. 通过简单的过滤命令(如 `kubectl logs -l app=higress-gateway | grep ai-zdtc-token`), 您可以迅速定位到本插件的所有运行轨迹.

根据您的要求, 已在所有日志输出的前面加上了 `[ai-zdtc-token]` 统一前缀, 并将 `redisClient` 保持为私有字段.

以下是完整的 Go 源代码:

```go
```

---------------------------------------------------------------------------------------------

Q: 我编译上述代码并构建了镜像, 在 Higress 控制台中创建了相应的插件, 查看 higress gateway 日志, 发现有如下报错
```
2026-07-08T05:39:54.697249Z	info	wasm	fetching image plugins/ai-zdtc-token from registry higress-registry.cn-hangzhou.cr.aliyuncs.com with tag 20260708-133037-35df208a

2026-07-08T05:39:55.069825Z	error	wasm	error in applying failopen rbac config: cannot fetch Wasm module oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-zdtc-token:20260708-133037-35df208a: could not fetch Wasm OCI image: could not fetch manifest: GET https://higress-registry.cn-hangzhou.cr.aliyuncs.com/v2/plugins/ai-zdtc-token/manifests/20260708-133037-35df208a: UNAUTHORIZED: authentication required; [map[Action:pull Class: Name:plugins/ai-zdtc-token Type:repository]]
```
我没有将镜像推送到远程镜像仓库, 我可以直接使用本地的镜像吗? 在我的机器上是有这个镜像的
```
root@ubuntu-mass:~/huzhi# crictl images | grep ai-zdtc-token
higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-zdtc-token                 20260708-133037-35df208a       2147a3b24070b       5.58MB
root@ubuntu-mass:~/huzhi#
```
并且我的镜像拉取策略是本地不存在时拉取
```
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  annotations:
    higress.io/wasm-plugin-title: ai-zdtc-token
  creationTimestamp: "2026-07-08T05:38:14Z"
  generation: 4
  labels:
    higress.io/resource-definer: higress
    higress.io/wasm-plugin-built-in: "false"
    higress.io/wasm-plugin-category: custom
    higress.io/wasm-plugin-name: ai-zdtc-token
    higress.io/wasm-plugin-version: 1.0.0
  name: ai-zdtc-token-1.0.0
  namespace: higress-system
  resourceVersion: "10909881"
  uid: 516feb28-8d7b-429a-b927-bf430d99e0d3
spec:
  defaultConfig:
    redis:
      service_name: redis-service.default.svc.cluster.local
      service_port: 6379
      timeout: 2000
  defaultConfigDisable: false
  failStrategy: FAIL_OPEN
  imagePullPolicy: IfNotPresent
  phase: UNSPECIFIED_PHASE
  priority: 95
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-zdtc-token:20260708-133037-35df208a

```
我应该如何解决这个问题, 如何使用本地镜像

这个问题是 Higress 插件本地开发和调试中非常经典的一个痛点.

### 为什么会报错? (原理解析)

虽然您通过 `crictl images` 能在宿主机(Node)上看到这个镜像, 但 Higress 依然报错并尝试向远端拉取, 原因在于 Wasm 插件镜像的下载机制与普通的 K8s Pod 容器镜像拉取是完全隔离的:

1. 执行下载的主体不同:

   * 普通 Pod 的镜像是宿主机上的 Kubelet 调用本地容器运行时(containerd/docker)去拉取的, 因此宿主机上有镜像就可以直接使用.
   * 而 Wasm 插件的 OCI 镜像, 是由运行在 `higress-gateway` 容器内部的 `istio-agent`(Envoy 边车代理) 独立通过网络请求去拉取的. 它无法直接读取或共享宿主机上的 `containerd` 镜像列表.

2. `imagePullPolicy: IfNotPresent` 的含义不同:

   * 在 `WasmPlugin` 中, `IfNotPresent` 指的是 `istio-agent` 容器内部的本地文件/内存缓存中是否存在该插件. 当网关 Pod 首次启动或重新加载配置时, 其内部缓存为空, 它必然会发起网络请求去 OCI 地址下载.
   * 此时, 因为您没有将镜像推送到阿里云的远端私有仓库(或者该仓库需要账号密码验证), `istio-agent` 直接去 `higress-registry.cn-hangzhou.cr.aliyuncs.com` 拖取镜像时, 就会触发 `UNAUTHORIZED: authentication required` 的鉴权失败报错.

---

### 如何解决?

针对本地开发和调试, 有以下两种主流的解决方案:

#### 方案一: 使用 `file://` 本地文件挂载方式(本地开发、免推镜像, 最为推荐)

这种方式完全绕过了 OCI 镜像拉取和网络传输. 直接将编译好的 `.wasm` 二进制文件挂载到 `higress-gateway` 的 Pod 内部, 并使用 `file://` 协议去读取.

##### 步骤 1: 利用 ConfigMap 导入 Wasm 二进制文件

假设您本地编译出了 `main.wasm` 文件(大小一般在 2MB~3MB 左右), 直接将其创建为 Kubernetes ConfigMap:

```bash
kubectl create configmap ai-zdtc-token-wasm --from-file=main.wasm=./main.wasm -n higress-system
```

##### 步骤 2: 修改 `higress-gateway` 部署, 挂载该 ConfigMap

您需要修改 `higress-gateway` 的 Deployment(或 DaemonSet), 将刚刚创建的 ConfigMap 挂载进容器中.
执行修改命令:
```bash
kubectl edit deployment higress-gateway -n higress-system
```
在 YAML 的对应位置添加 `volumes` 和 `volumeMounts` 配置:
```yaml
spec:
  template:
    spec:
      containers:
        - name: higress-gateway
          # ... 其他配置 ...
          volumeMounts:
            # 添加以下内容, 将 Wasm 文件挂载到容器的 /var/local/plugins 目录下
            - name: wasm-volume
              mountPath: /var/local/plugins
      volumes:
        # 添加以下内容, 引用上面创建的 ConfigMap
        - name: wasm-volume
          configMap:
            name: ai-zdtc-token-wasm
```

##### 步骤 3: 修改 `WasmPlugin` 配置
将您的 `WasmPlugin` YAML 中的 `url` 修改为 `file://` 协议:
```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-zdtc-token-1.0.0
  namespace: higress-system
spec:
  # ... 其他不变 ...
  # 修改为本地文件路径
  url: file:///var/local/plugins/main.wasm
```
提示: 通过 `file://` 部署后, 每次您修改了代码重新编译 `main.wasm`, 只需重新通过 `kubectl create configmap ... --dry-run=client -o yaml | kubectl apply -f -` 替换 ConfigMap, 网关便能自动热加载更新.

---

#### 方案二: 推送到集群可访问的镜像仓库(云原生标准流程)

如果您还是希望走 OCI 镜像的流程, 您不能只让镜像停留在宿主机上.

##### 1. 推送到公共免密仓库(如 Docker Hub、GitHub Packages 或阿里云个人版公开仓库)

如果您将镜像推送到一个无需登录即可 Pull 的公开 OCI 仓库, 那么 `WasmPlugin` 配置保持原样, 网关的 `istio-agent` 便能直接通过外网将其下载并正常运行.

##### 2. 推送到私有仓库(如您报错中的阿里云私有仓库)并配置凭证

如果您已经将镜像推送到了阿里云私有仓库:

1. 在 `higress-system` 命名空间下创建凭证 Secret:

   ```bash
   kubectl create secret docker-registry aliyun-registry-key \
     --docker-server=higress-registry.cn-hangzhou.cr.aliyuncs.com \
     --docker-username=<您的阿里云镜像仓库用户名> \
     --docker-password=<您的阿里云镜像仓库密码/独立访问凭证> \
     -n higress-system
   ```

2. 在 `WasmPlugin` 中引用该 Secret:
   修改您的 YAML, 在 `spec` 下增加 `imagePullSecret` 配置:
   ```yaml
   spec:
     # ... 其他配置 ...
     imagePullPolicy: IfNotPresent
     imagePullSecret: aliyun-registry-key # 引用刚刚创建的凭证 Secret
     url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-zdtc-token:20260708-133037-35df208a
   ```

如果是本地单机快速开发与测试阶段, 建议优先采用 方案一(file 挂载), 效率最高; 当开发完毕准备上生产环境交付时, 再采用 方案二(OCI 镜像 + imagePullSecret 凭证) 进行分发.

---------------------------------------------------------------------------------------------

Q: 我想将镜像 lanzhiwang/ai-zdtc-token:20260708-133037-35df208a 推送到公共免密仓库 Docker Hub 上, 我的 WasmPlugin url 应该如何配置

如果您想将镜像推送到公共免密仓库 Docker Hub, 并将 `WasmPlugin` 指向该镜像, 可以按照以下步骤进行操作:

### 第一步: 将镜像重新打标签(Tag)并推送到 Docker Hub

因为您当前的本地镜像是带有阿里云私有域名标签的, 您需要先将其重新命名为 Docker Hub 的格式, 然后再推送:

```bash
# 1. 登录您的 Docker Hub 账号(如尚未登录)
docker login

# 2. 将您本地的镜像重新打标(格式为: 用户名/仓库名:版本号)
docker tag higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-zdtc-token:20260708-133037-35df208a lanzhiwang/ai-zdtc-token:20260708-133037-35df208a

# 3. 将镜像推送到 Docker Hub
docker push lanzhiwang/ai-zdtc-token:20260708-133037-35df208a
```

> 重要提示: 在推送完毕后, 请登录您的 Docker Hub 网页端控制台, 确认 `lanzhiwang/ai-zdtc-token` 仓库的 Visibility(可见性) 设置为 Public(公开). Docker Hub 默认新建的仓库通常是公开的, 但如果是公开仓库, Higress 网关便可以直接免密拉取.

---

### 第二步: 配置 `WasmPlugin` 的 `url`

当镜像成功推送到 Docker Hub 的公共仓库后, 您的 `WasmPlugin` 配置应该修改为使用 `oci://docker.io` 前缀.

更新后的完整 YAML 配置如下:

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
      service_name: redis-service.default.svc.cluster.local
      service_port: 6379
      timeout: 2000
  defaultConfigDisable: false
  failStrategy: FAIL_OPEN
  imagePullPolicy: IfNotPresent
  phase: UNSPECIFIED_PHASE
  priority: 95
  # 关键修改点: 指向 Docker Hub 的公共 OCI 镜像地址
  url: oci://docker.io/lanzhiwang/ai-zdtc-token:20260708-133037-35df208a
```

### 💡 针对国内网络环境的友好提示:

由于您之前使用了阿里云杭州(`cn-hangzhou`)的镜像服务, 推测您的 Higress 网关及 Kubernetes 集群可能运行在国内的云服务器或本地网络环境中.

国内网络在拉取 Docker Hub 官方镜像(`docker.io`)时, 可能会遇到网络超时或连接失败的情况. 如果您发现更新后 Higress 依然报错无法连接 `docker.io`, 建议采用如下替代方案:

* 更稳妥的方案: 直接使用阿里云容器镜像服务(ACR)个人版. 在阿里云上创建一个公开(Public)的命名空间和镜像仓库(也是免费的), 推送至: `registry.cn-hangzhou.aliyuncs.com/您的命名空间/ai-zdtc-token:20260708-133037-35df208a`.
* 这样在 `WasmPlugin` 中配置 `oci://registry.cn-hangzhou.aliyuncs.com/...`, 不仅能省去配置 PullSecret 的麻烦, 在国内网络拉取时也会非常稳定.

```bash
电信测试环境:
registry-dx.wair.ac.cn
zsy/FIO0eOcrhC8=

docker tag \
higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-zdtc-token:20260731-112833-e6b3b314 \
registry-dx.wair.ac.cn/taichu-studio/higress-plugin:ai-zdtc-token-20260731-112833-e6b3b314

kubectl create secret docker-registry registry-dx-wair-ac-cn \
--docker-server=registry-dx.wair.ac.cn \
--docker-username=zsy \
--docker-password=FIO0eOcrhC8= \
-n higress-system

curl -sv http://172.16.10.98:8080/v1/chat/completions \
-X POST \
-H 'Content-Type: application/json' \
-H 'Authorization: Bearer a0s4asfkits7wgvs6jhr5vc5' \
-d '{
    "model": "claude",
    "stream": false,
    "messages": [
        {
            "role": "user",
            "content": "您好!"
        }
    ]
}'

curl -sv http://172.16.10.98:8080/v1/chat/completions \
-X POST \
-H 'Content-Type: application/json' \
-H 'Authorization: Bearer a0s4asfkits7wgvs6jhr5vc5' \
-d '{
    "model": "claude",
    "stream": true,
    "messages": [
        {
            "role": "user",
            "content": "什么是redis?"
        }
    ]
}'

```

---------------------------------------------------------------------------------------------

Q: onHttpStreamResponseBody 方法实现如下
```go
func onHttpStreamResponseBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isLastChunk bool, log log.Log) []byte {
	log.Infof("[ai-zdtc-token] === [OnHttpStreamResponseBody] 阶段触发 ===")
	log.Infof("[ai-zdtc-token] 当前数据块大小: %d 字节, 是否为最后一个数据块: %t", len(chunk), isLastChunk)

	if len(chunk) > 0 {
		// 打印当前接收到的数据块
		log.Infof("[ai-zdtc-token] 数据块内容: %s", string(chunk))
	}

	log.Infof("[ai-zdtc-token] === [OnHttpStreamResponseBody] 阶段结束 ===")
	// 务必返回原 chunk, 避免数据截断
	return chunk
}
```
在 onHttpStreamResponseBody 方法中, 但 isLastChunk 为 true 时, 也就是最后一个数据块时, 使用 github.com/higress-group/wasm-go/pkg/tokenusage 中的类似 usage := tokenusage.GetTokenUsage(ctx, chunk) 方法获取相关信息, 请在 onHttpStreamResponseBody 方法中该逻辑, 在日志中详细输出 usage 的内容

为了在 `onHttpStreamResponseBody` 中顺利调用 `GetTokenUsage` 方法, 我们需要引入 Higress 内置的 `"github.com/higress-group/wasm-go/pkg/tokenusage"` 包.

该包能够根据 OpenAI/vLLM 兼容的协议规范, 从响应流数据(包括 SSE 格式的文本流数据)中自动解析出本次请求消耗的 Token 数量及所使用的模型信息.

### 完整优化的 Go 代码实现

以下是包含了 `tokenusage` 依赖引入、防空指针(nil)防御性校验、以及详细 Token 字段打印的完整代码:

```go
```

---

### 💡 专业开发提示:

在实际测试时, 如果您发现日志输出 `GetTokenUsage` 结果全部为 `0` 或 `nil`, 通常有以下两个需要注意的机制原因:

1. vLLM 请求体中的 `stream_options` 配置:
   对于流式输出(SSE), 大多数 OpenAI 兼容接口(包括 vLLM)在默认情况下, 流式的中间块和最后一个块是不包含 `usage` 数据的.
   * 解决办法: 客户端发起请求时, 需要在 Body 中显式指定:
     ```json
     "stream_options": {
         "include_usage": true
     }
     ```
   * 如果该参数为 `true`, vLLM 就会在流的最后一帧(即 `isLastChunk == true` 时)返回一个包含 `usage` 结构的 JSON 包, 此时 `GetTokenUsage` 就能完美捕捉到该信息.

2. WasmPlugin 的执行优先级 (`priority`):
   在 Higress 默认配置中, 有一些前置插件可能已经消耗并移除了 SSE 流的某些特定部分. 当前的配置 `priority: 95` 是一个相对合理的执行顺序. 如果在后续逻辑中需要配合 `ai-statistics`(AI统计插件)或 `ai-proxy`(AI代理插件), 可以根据具体的业务流适当微调该优先级.

---------------------------------------------------------------------------------------------

Q: onHttpRequestHeaders 函数实现如下:
```go
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {
	log.Infof("[ai-zdtc-token] === [OnHttpRequestHeaders] 阶段开始 ===")

	// 打印基础请求元数据
	path, _ := proxywasm.GetHttpRequestHeader(":path")
	method, _ := proxywasm.GetHttpRequestHeader(":method")
	log.Infof("[ai-zdtc-token] 当前请求 - 方法: %s, 路径: %s", method, path)

	// 获取并打印所有请求头
	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Errorf("[ai-zdtc-token] 无法获取请求头: %v", err)
	} else {
		log.Infof("[ai-zdtc-token] 请求头总数: %d", len(headers))
		for _, h := range headers {
			log.Infof("[ai-zdtc-token]   Header -> %s: %s", h[0], h[1])
		}
	}

	log.Infof("[ai-zdtc-token] === [OnHttpRequestHeaders] 阶段结束 ===")
	return types.ActionContinue
}
```
如果获取到的 header key 是 x-request-id, x-higress-llm-model, x-higress-llm-model-final, x-hi-original-auth, 那么将这四个相关的头信息写入 ctx wrapper.HttpContext, 以便后续使用

在 Higress/WASM Go 插件开发中, `ctx wrapper.HttpContext` 提供了 `SetContext(key string, value interface{})` 和 `GetContext(key string) interface{}` 方法. 我们可以非常方便地利用这两个 API, 在请求头部解析阶段(`onHttpRequestHeaders`)暂存信息, 并在后续的阶段(例如流式响应处理 `onHttpStreamResponseBody`)将其读取出来.

需要注意的是, 在 Envoy 内核及 WASM 协议栈内部, HTTP Header 的 Key 默认会统一转换为小写(例如 `X-Request-Id` 会变成 `x-request-id`). 为了保证检索的鲁棒性, 我们在代码中引入了 `strings` 包, 并在判断时对 key 做了一次小写转换.

以下是为您修改后的完整代码. 除了将匹配的头信息写入 `ctx` 以外, 我还在最后的 `onHttpStreamResponseBody` 函数中, 为您添加了如何从 `ctx` 中安全读取并打印这四个变量的代码演示:

```go
```

---------------------------------------------------------------------------------------------

Q: 如下代码中的 "x-request-id", "x-higress-llm-model", "x-higress-llm-model-final", "x-hi-original-auth" 最好使用字符串常量表示, 我认为这样可读性更强
```go
			switch lowerKey {
			case "x-request-id", "x-higress-llm-model", "x-higress-llm-model-final", "x-hi-original-auth":
				// 将匹配到的 key-value 写入 ctx 上下文中, 以便在之后的 onHttpStreamResponseBody 等生命周期中使用
				ctx.SetContext(lowerKey, val)
				log.Infof("[ai-zdtc-token]   [Context写入成功] -> Key: %s, Value: %s", lowerKey, val)
			}
```
请使用常量表示, 输出所有的代码

使用常量(Constants)来替换魔法字符串(Magic Strings)是非常优秀的工程实践. 这不仅能显著提高代码的可读性, 还能有效防止在后续多次读取(`SetContext` / `GetContext`)时因为拼写错误而引入难以排查的 Bug.

我已为您定义了相关的字符串常量, 并替换了代码中所有的硬编码部分. 以下是更新后的完整源代码:

```go
```

---------------------------------------------------------------------------------------------

Q: 在请求头处理阶段(将特定头信息保存至上下文 ctx), 将当前时间的时间戳也保存到上下文 ctx 当中

在 `onHttpRequestHeaders` 阶段将请求开始的时间戳保存至 `ctx` 是一个非常典型的生产实践设计. 这样在流式结束(`isLastChunk == true`)解析出 Token 数量时, 可以用"当前时间减去请求开始时间", 从而计算出整个大模型推理请求的端到端耗时(Latency).

我们同样在全局常量中定义了该时间戳的 Key: `ctxKeyRequestStartTime`.

以下是为您实现该逻辑后的完整源代码(并在流结束阶段为您自动计算并输出了请求耗时):

```go
```

---------------------------------------------------------------------------------------------

Q: onHttpStreamResponseBody 函数实现如下:
```go
func onHttpStreamResponseBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isLastChunk bool, log log.Log) []byte {
	log.Infof("[ai-zdtc-token onHttpStreamResponseBody] === [OnHttpStreamResponseBody] 阶段触发 ===")
	log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 当前数据块大小: %d 字节, 是否为最后一个数据块: %t", len(chunk), isLastChunk)

	if len(chunk) > 0 {
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 数据块内容: %s", string(chunk))
	}
	if usage := tokenusage.GetTokenUsage(ctx, chunk); usage.TotalToken > 0 {
		ctx.SetContext(tokenusage.CtxKeyInputToken, usage.InputToken)
		ctx.SetContext(tokenusage.CtxKeyOutputToken, usage.OutputToken)
		ctx.SetContext(tokenusage.CtxKeyTotalToken, usage.TotalToken)
		ctx.SetContext(tokenusage.CtxKeyModel, usage.Model)
	}

	if isLastChunk {
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 收到最后一个数据块, 尝试解析 Token 消耗数据")

		// 从 HttpContext 中读取暂存的请求头信息, 并使用类型断言 .(string) 转换回字符串
		var requestID, llmModel, llmModelFinal, originalAuth, mseConsumer, responseID string
		var startTimeMilli int64

		if v := ctx.GetContext(headerXRequestID); v != nil {
			requestID, _ = v.(string)
		}
		if v := ctx.GetContext(headerXHigressLLMModel); v != nil {
			llmModel, _ = v.(string)
		}
		if v := ctx.GetContext(headerXHigressLLMModelFin); v != nil {
			llmModelFinal, _ = v.(string)
		}
		if v := ctx.GetContext(headerXHiOriginalAuth); v != nil {
			originalAuth, _ = v.(string)
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

		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 读取到暂存的 Context 字段值:")
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXRequestID, requestID)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXHigressLLMModel, llmModel)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXHigressLLMModelFin, llmModelFinal)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXHiOriginalAuth, originalAuth)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerMseConsumer, mseConsumer)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXResponseID, responseID)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %d (毫秒级时间戳)", ctxKeyRequestStartTime, startTimeMilli)


		inputToken := ctx.GetContext(tokenusage.CtxKeyInputToken).(int64)
		outputToken := ctx.GetContext(tokenusage.CtxKeyOutputToken).(int64)
		totalToken := ctx.GetContext(tokenusage.CtxKeyTotalToken).(int64)
		model := ctx.GetContext(tokenusage.CtxKeyModel)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 模型名称 (Model): %s", model)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 输入 Token 数量 (InputToken): %d", inputToken)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 输出 Token 数量 (OutputToken): %d", outputToken)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 总共 Token 数量 (TotalToken): %d", totalToken)


	}

	log.Infof("[ai-zdtc-token onHttpStreamResponseBody] === [OnHttpStreamResponseBody] 阶段结束 ===")
	// 务必返回原 chunk, 避免数据截断
	return chunk
}

```
在 onHttpStreamResponseBody 中, 如果收到最后一个数据块, 将相关信息写入 redis 中,
其中 key 的规则是 pluginName|mseConsumer|requestID|responseID
value 是一个 json 字符串, 将上述所有信息都添加到 json 字符串中

在对您提供的代码进行 Redis 写入和 JSON 序列化功能扩展时, 有以下几个非常核心的工程安全设计与边界条件需要优先为您指出:

### 1. 核心改进与防崩盘设计(避免 Panic)

* 安全的 Context 类型断言(关键风险点):
  在您提供的代码片段中, 直接使用了 `inputToken := ctx.GetContext(tokenusage.CtxKeyInputToken).(int64)`.
  * 风险: 如果中间没有匹配到任何包含 Token 的 chunk, `GetContext` 将返回 `nil`. 在 Golang 中, 将 `nil` 直接断言为 `int64` 会直接触发 Panic, 从而导致网关处理流程中断并给客户端返回 500.
  * 解决方案: 在下面的代码中, 我对它们进行了安全防御性断言(通过 `nil` 校验与类型推断), 以此保证插件在任何异常流下都不会崩溃.

* `x-response-id` 的捕获时机:
  `responseID`(通常代表响应层面的 ID)往往来自于响应头阶段. 为了保证在流结束时能够通过 `ctx.GetContext(headerXResponseID)` 读取到它, 我已在 `onHttpResponseHeaders` 阶段补充了对其的监听与暂存逻辑, 确保 `responseID` 能够顺利写入.

---

### 完整的 `main.go` 源代码

以下是实现将完整信息异步写入 Redis 并输出详细日志的完整插件代码:

```go
```

---------------------------------------------------------------------------------------------

```bash
kubectl -n default run --rm=true -t -i --restart=Never \
--image-pull-policy=IfNotPresent \
--image=redis \
redis-client --command bash

redis-cli -h redis.infra.svc.cluster.local -p 6379
```

---------------------------------------------------------------------------------------------

Q: 我在验证这个插件时, redis 有如下报错
```
[2026-07-09 03:38:14.164][62][warning][wasm] [external/envoy/source/extensions/common/wasm/context.cc:1516] wasm log: failed to init redis: error status returned by host: bad argument, will retry after

[2026-07-09 03:50:34.807][62][error][wasm] [external/envoy/source/extensions/common/wasm/context.cc:1519] wasm log higress-system.ai-zdtc-token-1.0.0: [ai-zdtc-token] [nil] [ca96ae5a-187b-449c-b2f5-0c060378ed48] [ai-zdtc-token onHttpStreamResponseBody] 异步调用 Redis 接口时发生内部错误: &errors.errorString{s:"error status returned by host: bad argument"}
```
插件的完整代码是
```go
```
请问这是什么原因, 应该如何解决

这个报错 `error status returned by host: bad argument` 是 Higress WASM 插件在与 Redis(或任何外部 HTTP 服务)交互时最经典、最常见的连接初始化失败问题.

---

### 一、报错的根本原因

在 Envoy WASM 宿主环境(Host)中, `bad argument` 的底层含义是: "无法在网关数据面(Envoy Cluster 列表)中找到对应的服务集群".

简单来说, 当插件在代码中调用 `redisClient.Init` 或 `redisClient.Set` 时, Envoy 会拿 `service_name` 去匹配网关内部已有的后端服务集群. 如果匹配不上(Envoy 认为该服务不存在), 就会直接拒绝并返回 `bad argument`.

目前您的配置为:
```yaml
redis:
  service_name: redis-service.default.svc.cluster.local
  service_port: 6379
```
尽管 K8s 集群中确实存在这个 Redis Service, 但 Envoy 无法识别它. 以下是针对不同部署场景的对应解决办法:

---

### 二、解决方案

请根据您部署 Redis 的实际情况, 选择以下对应的解决方案:

#### 方案 A: Redis 部署在 K8s 集群内(您的当前情况)

如果您直接使用 K8s Service 域名 `redis-service.default.svc.cluster.local`:

##### 原因分析:

默认情况下, 为了减轻数据面压力, Higress 的全局参数 `global.onlyPushRouteCluster` 默认被设置为 `true`. 这意味着 Higress 不会自动将 K8s 的所有 Service 发现并下发给 Envoy, 只有真正配置了 Ingress 路由的目标服务才会被下发. 由于没有路由指向 Redis, Envoy 内部没有 Redis 对应的 Cluster 节点, 从而报错 `bad argument`.

##### 解决办法(二选一):

* 方法一(推荐: 无需修改全局网关配置):
  为 Redis 服务创建一个内部虚拟 Ingress 路由. 这会强制 Higress 将 Redis 服务的 Cluster 配置推送到 Envoy 中.
  创建如下 YAML 并应用到 K8s 中(注意与您的 Redis 处于相同 Namespace):
  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    name: redis-internal-route
    namespace: default  # 与您的 Redis Service 相同命名空间
    annotations:
      # 告诉 Higress 目的地址和端口
      higress.io/destination: redis-service.default.svc.cluster.local:6379
      # 标记为内部虚拟服务, 外部无法直接通过公网 path 访问
      higress.io/internal: "true"
  spec:
    ingressClassName: higress
    rules:
      - http:
          paths:
            - path: /redis-dummy-path-placeholder
              pathType: Prefix
              backend:
                service:
                  name: redis-service
                  port:
                    number: 6379
  ```
  创建该 Ingress 后, 网关内部便会生成对应的 Cluster, 您插件中的 `service_name: redis-service.default.svc.cluster.local` 便可正常完成初始化, 不会再报 `bad argument`.

* 方法二(影响范围大, 适合测试环境):
  如果您拥有 Higress 的安装控制权, 可以通过修改 Helm Value, 将全局参数 `global.onlyPushRouteCluster` 改为 `false`.
  修改后, Higress 会自动把集群内所有的 K8s Service 全量同步给 Envoy, 此时插件可以直接无缝使用 `redis-service.default.svc.cluster.local`.

---

#### 方案 B: Redis 是通过"静态 IP/域名"在 Higress 控制台注册的

如果您在 Higress 控制台的"服务来源/服务列表"中, 将外部 Redis 手动注册为服务:

##### 解决办法:
1. 服务名称必须携带类型后缀:
   如果您在控制台注册的服务名称叫 `local-redis`, 那么在 WasmPlugin 插件配置中, 必须带上服务后缀(例如 `.static` 或 `.dns`).
   * 静态 IP 注册的服务: 配置中填 `local-redis.static`.
   * 外部域名解析的服务: 配置中填 `local-redis.dns`.
2. ⚠️ 极为反直觉的端口限制(非常关键):
   在 Higress 架构中, 所有后缀为 `.static`(静态服务) 的外部服务, 其在网关数据面的虚拟映射端口必须指定为 `80`(即使您的真实 Redis 部署在 6379 端口).
   * 错误的静态服务配置:
     ```yaml
     redis:
       service_name: local-redis.static
       service_port: 6379 # 会报错 bad argument
     ```
   * 正确的静态服务配置:
     ```yaml
     redis:
       service_name: local-redis.static
       service_port: 80 # 必须指定为虚拟端口 80
     ```

### 总结

请优先尝试方案 A 的方法一(在 K8s 中为 Redis 部署一个带 `higress.io/internal: "true"` 注解的内部虚拟 Ingress 路由), 这通常能立竿见影地解决本地和容器内开发测试时的 `bad argument` 连接问题.

---------------------------------------------------------------------------------------------

Q: 假设你是一位精通 higress 及其插件开发的高级开发人员, 我现在想开发如下功能的插件:
1、higress 的上游服务是 vllm 部署的模型服务
2、用户经过 higress gateway 请求上游的模型服务, 在我的插件中将用户信息, 最终的模型输出的 输入 token 数量和输出 token 数量记录在 redis 中
代码如下:
```go
```
请根据描述和代码实现, 帮我生成 README.md 文件内容和 spec.yaml 配置内容

为您生成了针对该插件的 `README.md`(使用说明文档)和 `spec.yaml`(元数据及配置约束规范). 这两个文件在打包 WASM 插件 OCI 镜像或在 Higress 控制台进行私有部署集成时是必须提供的标准文件.

---

### 一、`spec.yaml` 配置文件

该文件用于定义插件的基础元数据、执行阶段、执行优先级, 以及控制台所需的配置可视化约束架构(基于 OpenAPI V3 Schema).

```yaml
```

---

### 二、`README.md` 说明文档

该文档包含了插件的设计架构、工作原理、参数说明、部署约束(包括核心的 `bad argument` 避坑指南)以及 Redis 数据结构格式说明.

```markdown
```

---------------------------------------------------------------------------------------------

Q: 假设你是一位精通 golang 以及 higress 插件开发的高级开发人员, 我现在实现了如下的插件功能, 代码如下:
```
```
请修改上述代码, 实现下面两个功能,
1、在 onHttpRequestHeaders 函数中生成一个 uuid 写入 ctx.SetContext, 后续将这个 uuid 一并写入 redis, 作为每一次请求的唯一id
2、在最后写入 redis 中时, 如果 input_token 或者 output_token 为零, 就不写入 redis

这里是为您修改后的 Higress WASM-Go 插件代码.

修改主要包括以下两点:

1. 引入了 `"github.com/google/uuid"` 库: 在 `onHttpRequestHeaders` 中调用 `uuid.New().String()` 生成一个唯一的 UUID, 并通过 `ctx.SetContext` 存入上下文. 最后将其反序列化至 `TokenAuditLog` 结构体中的 `uuid` 字段.

2. 零 Token 拦截逻辑: 在 `onHttpStreamResponseBody` 函数的 `isLastChunk` 处理逻辑中, 获取到 `inputToken` 和 `outputToken` 后进行判断, 若其中任意一个为 `0`, 则直接通过日志记录并跳过后续的 Redis 写入操作, 避免不必要的网络开销.

### 修改后的完整代码

```go
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
	headerXHiOriginalAuth     = "x-hi-original-auth"
	headerMseConsumer         = "x-mse-consumer"

	headerXResponseID = "x-response-id"

	ctxKeyRequestStartTime = "request_start_time"
	ctxKeyRequestUUID      = "request_uuid" // 新增: 保存 UUID 的 Context Key
)

type TokenAuditLog struct {
	UUID           string `json:"uuid"` // 新增: 单次请求的唯一 UUID
	RequestID      string `json:"request_id"`
	LLMModel       string `json:"llm_model"`
	LLMModelFinal  string `json:"llm_model_final"`
	OriginalAuth   string `json:"original_auth"`
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

	// 新增功能 1: 在请求头处理阶段生成一个 UUID 并写入 Context
	reqUUID := uuid.New().String()
	ctx.SetContext(ctxKeyRequestUUID, reqUUID)
	log.Infof("[ai-zdtc-token onHttpRequestHeaders] 生成请求唯一 UUID: %s 并写入 Context", reqUUID)

	// 将当前请求开始的时间戳(毫秒级)保存到上下文 ctx 中
	nowMilli := time.Now().UnixMilli()
	ctx.SetContext(ctxKeyRequestStartTime, nowMilli)
	log.Infof("[ai-zdtc-token onHttpRequestHeaders]   [Context写入成功] -> Key: %s, Value: %d (毫秒级时间戳)", ctxKeyRequestStartTime, nowMilli)

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
			log.Infof("[ai-zdtc-token onHttpRequestHeaders]   Header -> %s: %s", key, val)

			lowerKey := strings.ToLower(key)

			switch lowerKey {
			case headerXHigressLLMModel, headerXHigressLLMModelFin, headerXHiOriginalAuth, headerMseConsumer:
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

	if len(body) > 0 {
		log.Infof("[ai-zdtc-token onHttpRequestBody] 请求体内容: %s", string(body))
	} else {
		log.Infof("[ai-zdtc-token onHttpRequestBody] 请求体为空")
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
			log.Infof("[ai-zdtc-token onHttpResponseHeaders]   Header -> %s: %s", key, val)

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

	if len(chunk) > 0 {
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 数据块内容: %s", string(chunk))
	}
	if usage := tokenusage.GetTokenUsage(ctx, chunk); usage.TotalToken > 0 {
		ctx.SetContext(tokenusage.CtxKeyInputToken, usage.InputToken)
		ctx.SetContext(tokenusage.CtxKeyOutputToken, usage.OutputToken)
		ctx.SetContext(tokenusage.CtxKeyTotalToken, usage.TotalToken)
		ctx.SetContext(tokenusage.CtxKeyModel, usage.Model)
	}

	if isLastChunk {
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 收到最后一个数据块, 尝试解析 Token 消耗数据")

		var requestUUID, requestID, llmModel, llmModelFinal, originalAuth, mseConsumer, responseID string
		var startTimeMilli int64

		// 获取在 onHttpRequestHeaders 中保存的 UUID
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
		if v := ctx.GetContext(headerXHiOriginalAuth); v != nil {
			originalAuth, _ = v.(string)
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

		log.Infof("[ai-zdtc-token onHttpStreamResponseBody] 读取到暂存的 Context 字段值:")
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - UUID: %s", requestUUID)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXRequestID, requestID)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXHigressLLMModel, llmModel)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXHigressLLMModelFin, llmModelFinal)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXHiOriginalAuth, originalAuth)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerMseConsumer, mseConsumer)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXResponseID, responseID)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %d (毫秒级时间戳)", ctxKeyRequestStartTime, startTimeMilli)

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

		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 模型名称 (Model): %s", modelStr)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 输入 Token 数量 (InputToken): %d", inputToken)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 输出 Token 数量 (OutputToken): %d", outputToken)
		log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - 总共 Token 数量 (TotalToken): %d", totalToken)

		// 新增功能 2: 如果 input_token 或者 output_token 为零, 就不写入 Redis
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
			// JSON Value 示例:
			// 	{
			// 		"uuid": "3e46c7d1-e634-4bc3-9fa7-6b2bf784a961",
			// 		"request_id": "10c3b60c-0598-4c09-bc85-49bbb876dc8c",
			// 		"llm_model": "claude",
			// 		"llm_model_final": "claude-opus-4-7",
			// 		"original_auth": "Bearer a0s4asfkits7wgvs6jhr5vc5",
			// 		"mse_consumer": "1-13833583",
			// 		"response_id": "6972de9fe61f85f7888ac9a35445f6d5",
			// 		"start_time_milli": 1783567458952,
			// 		"end_time_milli": 1783567460981,
			// 		"duration_ms": 2029,
			// 		"input_token": 3,
			// 		"output_token": 17,
			// 		"total_token": 20,
			// 		"model": "claude-opus-4-7"
			// 	}

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
```

### 注意事项:

* 依赖引入: `github.com/google/uuid` 已包含在 `github.com/higress-group/wasm-go` 的核心依赖中, 因此在使用 tinygo 编译打包时通常能直接解析. 若在本地编译出现依赖缺失报错, 可在网关对应的 `go.mod` 文件夹下执行 `go get github.com/google/uuid`.

---------------------------------------------------------------------------------------------


---------------------------------------------------------------------------------------------


---------------------------------------------------------------------------------------------

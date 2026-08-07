https://10.122.1.120/vminstance
admin
admin@123

ssh root@172.16.10.98
67R5^WSCyz9uY+
/usr/local/bin/sshpass -p '67R5^WSCyz9uY+' ssh root@172.16.10.98

higress
http://172.16.10.98:32333
admin
admin

```bash
docker run -ti --rm --name golang_1_24_1 \
-v /Users/huzhi/work/code/py_code/higress:/higress \
-w /higress golang:1.24.1 bash

kubectl -n default run --rm=true -t -i --restart=Never \
--image-pull-policy=IfNotPresent \
--image=redis \
redis-client --command bash

redis-cli -h redis.infra.svc.cluster.local -p 6379
```

---------------------------------------------------------------------------------------------

Q: 假设你是一位精通 higress 及其插件开发的高级开发人员, 我开发了一个简单的 higress 插件, 使用如下配置部署插件
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
higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-zdtc-token:20260806-113154-9e577662 \
registry-dx.wair.ac.cn/taichu-studio/higress-plugin:ai-zdtc-token-20260806-113154-9e577662

kubectl create secret docker-registry registry-dx-wair-ac-cn \
--docker-server=registry-dx.wair.ac.cn \
--docker-username=zsy \
--docker-password=FIO0eOcrhC8= \
-n higress-system

curl -sv http://172.16.10.98:8080/v1/chat/completions \
-X POST \
-H 'Content-Type: application/json' \
-H 'Authorization: Bearer hpqmjrj9wauys7lsajs5fyes' \
-d '{
    "model": "volces",
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

Q: 假设你是一位精通 golang 的高级开发人员, 同时也十分精通 higress 插件的开发, 我实现了如下插件, 代码如下:
```go
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
	headerMseConsumer         = "x-mse-consumer"
	headerXRequestID          = "x-request-id"

	headerXResponseID = "x-response-id"
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
	// {
	// 	"debug": true,
	// 	"pay": {
	// 		"balance_cache_ttl": 30,
	// 		"service_name": "financial-pay.static",
	// 		"service_port": 80,
	// 		"timeout": 20000
	// 	},
	// 	"redis": {
	// 		"service_name": "redis.dns",
	// 		"service_port": 6379,
	// 		"timeout": 2000
	// 	}
	// }

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
			case headerXHigressLLMModel, headerXHigressLLMModelFin, headerAuthorization, headerMseConsumer:
				ctx.SetContext(lowerKey, val)
				log.Infof("[ai-zdtc-token onHttpRequestHeaders]   [Context写入成功] -> Key: %s, Value: %s", lowerKey, val)
				if lowerKey == headerMseConsumer {
					mseConsumerVal = val
				}
				if lowerKey == headerAuthorization {
					authHeaderVal = val
				}
			case "x-request-id":
				ctx.SetContext(headerXRequestID, val)
				log.Infof("[ai-zdtc-token onHttpRequestHeaders]   [Context写入成功] -> Key: %s, Value: %s", headerXRequestID, val)
			}
		}
	}

	// 如果 authorization 的值以 Bearer sp 开头, 那么就直接放行, 跳过后面的所有逻辑
	if authHeaderVal != "" {
		trimmedAuth := strings.TrimSpace(authHeaderVal)
		if strings.HasPrefix(trimmedAuth, "Bearer sp") {
			log.Infof("[ai-zdtc-token onHttpRequestHeaders] 校验到 authorization 以 'Bearer sp' 开头 (%s), 直接放行", trimmedAuth)
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
			UUID:           requestUUID,
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
		err = config.redisClient.Set(redisKey, dataRaw, func(respVal resp.Value) {
			if respVal.Error() != nil {
				log.Errorf("[ai-zdtc-token callPayService] Redis 写入失败: %#v", respVal.Error())
			} else {
				_ = config.redisClient.Expire(redisKey, int(cacheTTL), nil)
			}
		})
		if err != nil {
			log.Errorf("[ai-zdtc-token callPayService] 调用 Redis Set 接口异常: %#v", err)
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

```
其中我请求了用户的账号信息, 将相关信息写入了redis, 并且设置了过期时间, 但是我在redis中查看发现redis并没有正确设置过期时间, 如下所示
```bash
redis.infra.svc.cluster.local:6379> get 450287008:balance:ai-zdtc-token
"{\"cash_balance\":\"-0.0001\",\"total_recharge\":\"11.0000\",\"free_balance\":\"-0.0001\",\"total_outcome\":\"221.7002\",\"free_income\":\"200.0000\"}"
redis.infra.svc.cluster.local:6379> ttl 450287008:balance:ai-zdtc-token
(integer) -1
```
请问这是什么原因, 应该如何解决

作为精通 Go 语言和 Higress WASM 插件开发的开发人员, 我为你精准诊断这个问题.

### 为什么 Redis 中 Key 的 TTL 是 `-1`?

在 `ttl` 返回 `-1` 时, 代表该 Key 在 Redis 中是永不过期的. 造成这个现象的核心原因是: Higress WASM 插件中的 Redis 操作是完全异步的, 且代码中存在"请求生命周期与嵌套异步回调冲突"的问题.

分析你原先的代码:

```go
// 1. 发起异步 SET 操作
err = config.redisClient.Set(redisKey, dataRaw, func(respVal resp.Value) {
    if respVal.Error() != nil {
        log.Errorf("[ai-zdtc-token callPayService] Redis 写入失败: %#v", respVal.Error())
    } else {
        // 2. 在 SET 回调成功后, 再发起异步 EXPIRE 操作
        _ = config.redisClient.Expire(redisKey, int(cacheTTL), nil)
    }
})

// 3. 紧接着在主流程(未等待 SET/EXPIRE 回调完成)直接 Resume 了 HTTP 请求!
if cashBalance <= 0 && freeBalance <= 0 {
    sendInsufficientBalanceResponse()
    return
}
proxywasm.ResumeHttpRequest()
```

这里发生了以下时序过程:
1. `config.redisClient.Set` 向 Redis 发送 `SET` 命令(非阻塞, 立即返回).
2. 代码继续向下执行, 调用了 `proxywasm.ResumeHttpRequest()` 或 `sendInsufficientBalanceResponse()`, 主请求流程立刻被恢复/终止了.
3. 当 Redis 响应 `SET` 成功并触发 Go 回调闭包时, 当前 HTTP 请求在 Envoy 侧的上下文(Request Context)可能已经完成或销毁, 导致闭包内嵌套调用的 `config.redisClient.Expire` 无法成功发送到 Redis, 或者被 Envoy 忽略/取消.
4. 结果就是: `SET` 命令成功执行了(数据存进去了), 但后续回调里的 `EXPIRE` 命令根本没生效, Key 就变成了永久 Key(TTL 为 -1).

---

### 正确解决方案

在 Higress `wasm-go` 框架中, 严禁使用"嵌套异步回调"(Set 里面套 Expire), 这不仅增加了网络 RTT 延迟, 还极易触发 WASM 生命周期失效问题.

正确的做法是使用原子的 `SetEx` 指令, 将"写入"和"设置过期时间"合二为一(对应 Redis 的 `SETEX key seconds value` 命令), 只需一次 RTT 即可完成.

#### 修改后的 `callPayService` 代码

将 `callPayService` 中写入 Redis 的部分修改为:

```go
		// 1. 将 data 的所有信息写入 Redis (使用 SetEx 原子操作完成 SET + EXPIRE)
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
			return
		}

		log.Infof("[ai-zdtc-token callPayService] 租户 %s 资金充裕 (cash_balance: %.4f, free_balance: %.4f), 余额校验通过",
			tenantID, cashBalance, freeBalance)
		proxywasm.ResumeHttpRequest()
```

> 提示: Higress 的 `wasm-go/pkg/wrapper` 库中 `SetEx` 的入参顺序一般为 `(key string, value string, sec int, callback RedisResponseCallback)`, 如果你的 SDK 版本提示参数不匹配, 请根据 IDE 提示微调 `dataRaw` 和 `cacheTTL` 的位置即可.

---

### 补充代码优化建议(生产级健壮性)

既然帮你排查了代码, 顺带提几个在 Envoy WASM 插件开发中容易踩坑的地方:

1. Context 类型断言防御 (Prevent Panic):
   在 `onHttpStreamResponseBody` 中, 你使用了断言 `startTimeMilli, _ = v.(int64)`. 如果某个 Request 没有设置过或者类型出现偏差(比如被转为了 `float64`), 直接断言会导致 WASM VM Panic(使得网关返回 500). 建议写成安全断言:
   ```go
   if v := ctx.GetContext(ctxKeyRequestStartTime); v != nil {
       if val, ok := v.(int64); ok {
           startTimeMilli = val
       }
   }
   ```

2. 请求暂停与恢复的配对 (ActionPause / StopAllIterationAndWatermark):
   - 官方 Go 1.24 原生编译方案中, 在 Header 阶段调用外部异步服务(HTTP / Redis)暂停请求时, 建议在 `onHttpRequestHeaders` 尾部返回 `types.HeaderStopAllIterationAndWatermark`(旧 TinyGo ABI 中为 `types.ActionPause`). 你代码中的逻辑是正确的.
   - 注意任何分支下如果不拦截, 必须保证有且仅有一次 `proxywasm.ResumeHttpRequest()` 调用；如果拦截, 必须调用 `sendInsufficientBalanceResponse()` 并及时 `return`, 这一点你当前的代码处理得很好.

改用 `SetEx` 后重新编译部署, 再次在 Redis 中使用 `ttl 450287008:balance:ai-zdtc-token` 查看, 就能看到正常的倒计时秒数(例如 `120` 或 `30`)了.

---------------------------------------------------------------------------------------------

Q: 假设你是一位精通 golang 的高级开发人员, 同时也十分精通 higress 插件的开发, 我实现了如下插件, 代码如下:
```go
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
	UUID            string `json:"uuid"`
	RequestID       string `json:"request_id"`
	LLMModel        string `json:"llm_model"`
	LLMModelFinal   string `json:"llm_model_final"`
	Authorization   string `json:"authorization"`
	XHiOriginalAuth string `json:"x_hi_original_auth"`
	MseConsumer     string `json:"mse_consumer"`
	ResponseID      string `json:"response_id"`
	StartTimeMilli  int64  `json:"start_time_milli"`
	EndTimeMilli    int64  `json:"end_time_milli"`
	DurationMs      int64  `json:"duration_ms"`
	InputToken      int64  `json:"input_token"`
	OutputToken     int64  `json:"output_token"`
	TotalToken      int64  `json:"total_token"`
	Model           string `json:"model"`
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

		var requestUUID, requestID, llmModel, llmModelFinal, authorization, xHiOriginalAuth, mseConsumer, responseID string
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
			xHiOriginalAuth, _ = v.(string)
			xHiOriginalAuth = strings.TrimPrefix(xHiOriginalAuth, "Bearer ")
			xHiOriginalAuth = strings.TrimSpace(xHiOriginalAuth)
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
			log.Infof("[ai-zdtc-token onHttpStreamResponseBody]   - %s: %s", headerXHiOriginalAuth, xHiOriginalAuth)
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
			UUID:            requestUUID,
			RequestID:       requestID,
			LLMModel:        llmModel,
			LLMModelFinal:   llmModelFinal,
			Authorization:   authorization,
			XHiOriginalAuth: xHiOriginalAuth,
			MseConsumer:     mseConsumer,
			ResponseID:      responseID,
			StartTimeMilli:  startTimeMilli,
			EndTimeMilli:    endTimeMilli,
			DurationMs:      durationMs,
			InputToken:      inputToken,
			OutputToken:     outputToken,
			TotalToken:      totalToken,
			Model:           modelStr,
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

```
其中有如下逻辑
```go
	// 如果 x-hi-original-auth 的值以 Bearer sp 开头, 那么就直接放行, 跳过后面的所有逻辑
	if authHeaderVal != "" {
		trimmedAuth := strings.TrimSpace(authHeaderVal)
		if strings.HasPrefix(trimmedAuth, "Bearer sp") {
			log.Infof("[ai-zdtc-token onHttpRequestHeaders] 校验到 x-hi-original-auth 以 'Bearer sp' 开头 (%s), 直接放行", trimmedAuth)
			log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段结束 ===")
			return types.ActionContinue
		}
	}
```
我查看日志之后发现如果 x-hi-original-auth 的值以 Bearer sp 开头, 确实直接返回了 return types.ActionContinue,
后续 onHttpRequestBody, onHttpResponseHeaders, onHttpStreamResponseBody 还是执行了, 最终也写入 redis 了,
这不是我想要的逻辑, 我想的是如果 x-hi-original-auth 的值以 Bearer sp 开头, 后续 onHttpRequestBody, onHttpResponseHeaders, onHttpStreamResponseBody 都不执行, 让 higress gateway 自己去返回客户端
这个要如何修改

在 Envoy / Higress 的 Wasm 插件架构中, 当你在 `onHttpRequestHeaders` 中返回 `types.ActionContinue` 时, 它的含义是: "当前插件对请求头的处理已完成, 通知 Envoy 继续后续的 HTTP 处理流程".

它不会取消或注销该插件对后续生命周期阶段(`onHttpRequestBody`、`onHttpResponseHeaders`、`onHttpStreamResponseBody`)的监听. 因此 Envoy 在后续各个阶段依然会触发对应的回调函数.

### 解决方案

在 Higress Wasm 插件的开发规范中, 若要彻底跳过该请求后续所有阶段的插件逻辑, 最推荐且标准的做法是:
1. 在 `onHttpRequestHeaders` 中判定需要跳过时, 在请求上下文 `ctx` 中写入一个标志位(例如 `skip_plugin = true`).
2. 在 `onHttpRequestBody`、`onHttpResponseHeaders` 和 `onHttpStreamResponseBody` 的最顶端增加检查, 如果读取到该标志位为 `true`, 直接快速 `return` 退出, 不执行任何日志解析和 Redis 操作.

---

### 具体代码修改步骤

#### 1. 定义 Context 常量
在常量定义区域增加一个 key:
```go
const (
	pluginName = "ai-zdtc-token"

	ctxKeyRequestUUID      = "request_uuid"
	ctxKeyRequestStartTime = "request_start_time"
	ctxKeySkipPlugin       = "skip_plugin" // 增加: 跳过插件标志位

	// ... 其他 header 常量保持不变 ...
)
```

#### 2. 修改 `onHttpRequestHeaders`
当检测到 `Bearer sp` 开头时, 写入 `ctxKeySkipPlugin` 标志位:

```go
	// 如果 x-hi-original-auth 的值以 Bearer sp 开头, 那么就直接放行, 跳过后面的所有逻辑
	if authHeaderVal != "" {
		trimmedAuth := strings.TrimSpace(authHeaderVal)
		if strings.HasPrefix(trimmedAuth, "Bearer sp") {
			log.Infof("[ai-zdtc-token onHttpRequestHeaders] 校验到 x-hi-original-auth 以 'Bearer sp' 开头 (%s), 标记 skip_plugin 并放行", trimmedAuth)
			ctx.SetContext(ctxKeySkipPlugin, true) // 写入跳过标记
			log.Infof("[ai-zdtc-token onHttpRequestHeaders] === [OnHttpRequestHeaders] 阶段结束 ===")
			return types.ActionContinue
		}
	}
```

#### 3. 修改 `onHttpRequestBody`
在入口处加入跳过检查:

```go
func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log log.Log) types.Action {
	// 如果设置了跳过标记, 直接放行
	if skip, ok := ctx.GetContext(ctxKeySkipPlugin).(bool); ok && skip {
		return types.ActionContinue
	}

	log.Infof("[ai-zdtc-token onHttpRequestBody] === [OnHttpRequestBody] 阶段开始 ===")
	// ... 原有逻辑 ...
	log.Infof("[ai-zdtc-token onHttpRequestBody] === [OnHttpRequestBody] 阶段结束 ===")
	return types.ActionContinue
}
```

#### 4. 修改 `onHttpResponseHeaders`
在入口处加入跳过检查:

```go
func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log log.Log) types.Action {
	// 如果设置了跳过标记, 直接放行
	if skip, ok := ctx.GetContext(ctxKeySkipPlugin).(bool); ok && skip {
		return types.ActionContinue
	}

	log.Infof("[ai-zdtc-token onHttpResponseHeaders] === [OnHttpResponseHeaders] 阶段开始 ===")
	// ... 原有逻辑 ...
	log.Infof("[ai-zdtc-token onHttpResponseHeaders] === [OnHttpResponseHeaders] 阶段结束 ===")
	return types.ActionContinue
}
```

#### 5. 修改 `onHttpStreamResponseBody`
在入口处加入跳过检查(注意: 该函数返回值是数据块 `[]byte`, 跳过时必须原样返回传入的 `chunk`):

```go
func onHttpStreamResponseBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isLastChunk bool, log log.Log) []byte {
	// 如果设置了跳过标记, 直接返回原始数据块, 不做任何 Token 计算与 Redis 写入操作
	if skip, ok := ctx.GetContext(ctxKeySkipPlugin).(bool); ok && skip {
		return chunk
	}

	log.Infof("[ai-zdtc-token onHttpStreamResponseBody] === [OnHttpStreamResponseBody] 阶段触发 ===")
	// ... 原有逻辑 ...
	log.Infof("[ai-zdtc-token onHttpStreamResponseBody] === [OnHttpStreamResponseBody] 阶段结束 ===")
	return chunk
}
```

---

### 原理解析

* 为什么 Envoy 会继续触发回调？
  Envoy C++ 层面的 `Http::StreamFilter` 是基于链式调用的, 在请求阶段挂载了该 Filter 后, 后续的 Body/Response 阶段默认都会挨个回调 Filter 对应的方法.
* 为什么使用 Context 标记是最佳做法？
  Proxy-Wasm 规范中并没有"动态在请求中途注销自身 Filter"的 ABI 指令. 因此在 `onHttpRequestHeaders` 阶段向请求级别打上上下文标记(`ctx.SetContext`), 并在后续阶段进行拦截早退, 是执行效率最高、对性能影响最小的标配做法.



---------------------------------------------------------------------------------------------

Q: 假设你是一位精通 golang 的高级开发人员, 也十分精通 higress 插件的开发, 我实现了一个 higress 插件, 代码如下:
```go
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

```
使用如下配置启动插件
```yaml
debug: true
pay:
  balance_cache_ttl: 30
  service_name: "financial-pay.static"
  service_port: 80
  timeout: 20000
redis:
  database: 10
  service_name: "redis.dns"
  service_port: 6379
  timeout: 2000
```
我在配置中定义使用 redis db 10, 但是我发现插件并没有把数据写入 10 号数据库, 有如下日志
```
[2026-08-06 08:35:29.976][62][info][wasm] [external/envoy/source/extensions/common/wasm/context.cc:1513] wasm log higress-system.ai-zdtc-token-1.0.0: [ai-zdtc-token] [nil] [c677602e-2eb1-42c8-93d2-fc577ad9698b] [ai-zdtc-token onHttpStreamResponseBody] 异步向 Redis 写入. Key: {ai-zdtc-token}-|450287008-51585633|c677602e-2eb1-42c8-93d2-fc577ad9698b|, Value: {"uuid":"00a5ddf6-38b7-4be5-bf8d-fdef3e43acad","request_id":"c677602e-2eb1-42c8-93d2-fc577ad9698b","llm_model":"taichu_llm","llm_model_final":"taichu","original_auth":"co9ks8yldh4d6bgellw8otsu","authorization":"admin","mse_consumer":"450287008-51585633","response_id":"","start_time_milli":1786005328922,"end_time_milli":1786005329975,"duration_ms":1053,"input_token":57,"output_token":128,"total_token":185,"model":"taichu"}
```
我检查reids
```bash
redis.infra.svc.cluster.local:6379[10]> select 0
OK
redis.infra.svc.cluster.local:6379> get {ai-zdtc-token}-|450287008-51585633|c677602e-2eb1-42c8-93d2-fc577ad9698b|
"{\"uuid\":\"00a5ddf6-38b7-4be5-bf8d-fdef3e43acad\",\"request_id\":\"c677602e-2eb1-42c8-93d2-fc577ad9698b\",\"llm_model\":\"taichu_llm\",\"llm_model_final\":\"taichu\",\"original_auth\":\"co9ks8yldh4d6bgellw8otsu\",\"authorization\":\"admin\",\"mse_consumer\":\"450287008-51585633\",\"response_id\":\"\",\"start_time_milli\":1786005328922,\"end_time_milli\":1786005329975,\"duration_ms\":1053,\"input_token\":57,\"output_token\":128,\"total_token\":185,\"model\":\"taichu\"}"
redis.infra.svc.cluster.local:6379>
redis.infra.svc.cluster.local:6379> select 10
OK
redis.infra.svc.cluster.local:6379[10]> get {ai-zdtc-token}-|450287008-51585633|c677602e-2eb1-42c8-93d2-fc577ad9698b|
(nil)
redis.infra.svc.cluster.local:6379[10]>
```
请问这是什么原因, 应该如何解决

这个问题的根本原因出在 Redis 集群模式(Redis Cluster)的协议限制 以及 Higress SDK / Envoy Redis 代理的实现机制.

---

### 核心原因分析

#### 1. Redis 集群协议(Redis Cluster)不支持多数据库(Multi-DB)

根据 Redis 官方规范, Redis Cluster 模式下只支持 0 号数据库(`db0`), 禁用 `SELECT` 命令.
在 Redis 集群架构中, 数据是通过 Hash Slot 散列在各个节点上的, 多数据库概念在 Cluster 模式下是被彻底废弃的.

#### 2. `wrapper.NewRedisClusterClient` 会强行使用集群代理逻辑

在你的插件 `parseConfig` 中, 使用了:

```go
config.redisClient = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
    FQDN: config.RedisInfo.ServiceName,
    Port: int64(config.RedisInfo.ServicePort),
})
```
`wrapper.NewRedisClusterClient` 告诉 Higress/Envoy 底层: 按 Redis Cluster 模式建立和管理该 Redis 连接.
在这种模式下:
- Envoy 的 Redis 代理层(`redis_proxy`)会按照集群协议处理命令.
- 即使你在 `Init` 中传入了 `wrapper.WithDataBase(10)`, Envoy Redis Cluster 客户端也会直接静默忽略 `database` 参数(因为无法向集群发送 `SELECT 10` 指令).
- 最终, 所有读写指令(`GET`/`SET`/`SETEX`)都会直接落入 Redis 的 DB 0.

---

### 解决方案

在云原生网关和分布式架构中, 针对这种情况有两种解决思路:

---

#### 方案一: 采用 Key 前缀隔离(强烈推荐 🌟)

在分布式环境或 Redis 集群场景下, 行业最佳实践是使用 Key 前缀(Prefix)做业务/数据库隔离, 而不是依赖 Redis `SELECT db`.

修改方法:
1. 保持使用 `db 0`(或者直接不传 `WithDataBase`).
2. 在构造 Redis Key 时, 加入数据库或命名空间前缀:

```go
// 例如: 原本的 Key 拼接逻辑
// redisKey := fmt.Sprintf("{%s}-|%s|%s|%s", pluginName, mseConsumer, requestID, responseID)

// 改为在 Key 中显式带上逻辑 DB 标识前缀:
redisKey := fmt.Sprintf("db10:{%s}-|%s|%s|%s", pluginName, mseConsumer, requestID, responseID)
```

> 提示: 如果使用 Redis Cluster 模式, Key 中的 `{...}` 是 Hash Tag. 写成 `db10:{ai-zdtc-token}-...` 可以确保同一个租户/插件的 Key 仍按 Hash Tag 进行 Slot 分发, 同时实现了逻辑库隔离.

---

#### 方案二: 若必须使用 DB 10(需 Redis 为单机/主从模式)

如果你部署的 Redis 确实是 Standalone(单机)或 Master-Replica(主从)架构(非 Redis Cluster), 且业务强要求必须写入 DB 10:

1. 更换 SDK 客户端初始化方法:
   在 Higress `wasm-go` SDK 中, 如果需要单机且支持 `SELECT db` 的连接, 需使用单机客户端构造函数(如 `wrapper.NewRedisClient`)代替 `NewRedisClusterClient`.
   (注意: 具体方法名视使用的 `wasm-go` SDK 版本决定)

2. 检查 Higress 中的服务定义:
   确保在 Higress 网关后台/控制台中注册的 `redis.dns` 服务没有勾选或配置为 Cluster 类型的 Redis upstream.

---

### 总结建议

| 场景 | 推荐做法 |
| :--- | :--- |
| 使用 Redis 集群 / 云厂商 Redis 集群版 | 方案一: 放弃 `database` 配置, 统一在 Key 上加前缀(如 `db10:xxxx`) |
| 使用 Redis 单机/主从, 且硬性要求切库  | 方案二: 改为使用单机 Redis Client 代理, 开启 `SELECT db` |

对于大模型 API 网关(Higress)插件开发, 最稳定且扩展性最好的方式是方案一, 未来即便 Redis 架构由单机升级到分布式集群, 插件代码也无需再次修改.

---------------------------------------------------------------------------------------------

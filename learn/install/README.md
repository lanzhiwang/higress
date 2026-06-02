# 安装部署

## hgctl install

* https://higress.ai/docs/latest/ops/hgctl/

通过 curl 命令下载最新版本的 hgctl 到本地

```bash
curl -Ls https://raw.githubusercontent.com/alibaba/higress/main/tools/hack/get-hgctl.sh | bash
```

1. `get-hgctl.sh` 脚本的作用就是从 `https://github.com/higress-group/higress/releases` 下载 `hgctl_v2.2.1_linux_amd64.tar.gz`

2. `hgctl_v2.2.1_linux_amd64.tar.gz` 解压之后就是二进制文件 `hgctl`

```bash
$ tree -a .
.
└── hgctl_v2.2.1_linux_amd64.tar.gz

1 directory, 1 file
$ tar -zxvf hgctl_v2.2.1_linux_amd64.tar.gz
x out/linux_amd64/
x out/linux_amd64/hgctl
$ tree -a .
.
├── hgctl_v2.2.1_linux_amd64.tar.gz
└── out
    └── linux_amd64
        └── hgctl

3 directories, 2 files
$
```

3. 二进制文件 `hgctl` 需要在 `node` 环境中运行

```bash
docker pull node:20.18.1-bullseye

docker run -ti --rm --name hgctl \
-v ~/work/code/py_code/higress/learn/install:/install \
-w /install \
node:20.18.1-bullseye \
bash

################################################################################

root@9c2f240ee9e2:/install# hgctl --help
A command line utility for operating Higress

Usage:
  hgctl [command]

Available Commands:
  agent          Start the interactive agent window
  code-debug     Start or stop code debug
  completion     generate autocompletion scripts for the specified shell
  dashboard      Access to Higress web UIs
  gateway-config Retrieve Higress Gateway configuration.
  help           Help about any command
  install        Applies an higress manifest, installing or reconfiguring higress on a cluster.
  manifest       Generate higress manifests.
  mcp            for the mcp management
  plugin         For the Golang WASM plugin
  profile        Commands related to higress configuration profiles
  uninstall      Uninstall higress from a cluster
  upgrade        Upgrade Higress in-place
  version        Show version

Flags:
  -h, --help   help for hgctl

Use "hgctl [command] --help" for more information about a command.

################################################################################

root@9c2f240ee9e2:/install# hgctl profile list
higress configuration profiles:
    k8s
    local-docker
    local-k8s

################################################################################

root@9c2f240ee9e2:/install# hgctl profile dump local-k8s -o ./local-k8s.yaml
root@9c2f240ee9e2:/install# hgctl profile dump k8s -o ./k8s.yaml
root@9c2f240ee9e2:/install# hgctl profile dump local-docker -o ./local-docker.yaml

################################################################################

root@9c2f240ee9e2:/install# hgctl install --help
The install command generates an higress install manifest and applies it to a cluster.

Usage:
  hgctl install [flags]

Examples:
  # Apply a default higress installation
  hgctl install

  # Install higress on local kubernetes cluster
  hgctl install --set profile=local-k8s

  # Install higress on local docker environment with specific gateway port
  hgctl install --set profile=local-docker --set gateway.httpPort=80 --set gateway.httpsPort=443

  # To override profile setting
  hgctl install --set profile=local-k8s  --set global.enableIstioAPI=true --set gateway.replicas=2"

  # To override helm setting
  hgctl install --set profile=local-k8s  --set values.global.proxy.resources.requests.cpu=500m"




Flags:
      --context string      The name of the kubeconfig context to use.
      --devel               use development versions (alpha, beta, and release candidate releases), If version is set, this is ignored
  -f, --filename strings    Path to file containing helm custom values
  -h, --help                help for install
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests.
  -d, --manifests string    Specify a path to a directory of profiles
                            (e.g. ~/Downloads/higress/manifests).
  -s, --set stringArray     Override an higress profile value, e.g. to choose a profile
                            (--set profile=local-k8s), or override profile values (--set gateway.replicas=2), or override helm values (--set values.global.proxy.resources.requests.cpu=500m).
root@9c2f240ee9e2:/install#

```

# helm install

```bash
helm repo add higress.io https://higress.io/helm-charts

helm repo update

helm search repo higress.io
NAME                      	CHART VERSION	APP VERSION	DESCRIPTION
higress.io/hiclaw         	1.1.2        	1.1.2      	HiClaw - AI Agent Teams with Matrix-based colla...
higress.io/higress        	2.2.2        	2.2.2      	Helm chart for deploying Higress gateways
higress.io/higress-console	2.2.2        	2.2.2      	Management console for Higress
higress.io/higress-core   	2.2.2        	2.2.2      	Helm chart for deploying higress gateways
higress.io/higress-local  	0.6.1        	0.6.1      	Helm chart for deploying higress gateways
higress.io/istio          	1.12.4       	1.12.4     	Helm chart for deploying higress istio
higress.io/istio-local    	1.12.4       	1.12.4     	Helm chart for deploying higress istio

helm pull higress.io/higress --version 2.2.2 --destination ./learn/install/helm/

helm install --debug --dry-run=client -n higress-system --create-namespace higress ./learn/install/helm/higress
helm install --debug --dry-run=client -n higress-system --create-namespace higress ./learn/install/helm/higress > ./learn/install/helm/helm_install.yaml 2>&1

# higress/charts/higress-console
ServiceAccount
  higress-console
  higress-console-promtail

Secret
  higress-console

ConfigMap
  higress-console

ClusterRole
  higress-console-higress-system

ClusterRoleBinding
  higress-console-higress-system

Role
  higress-console
  higress-console-promtail

RoleBinding
  higress-console
  higress-console-promtail

Service
  higress-console

Deployment
  higress-console higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/console:2.2.2

# higress/charts/higress-core
ServiceAccount
  higress-controller
  higress-gateway

ConfigMap
  higress-config

ClusterRole
  higress-gateway-higress-system
  higress-controller-higress-system

ClusterRoleBinding
  higress-gateway-higress-system
  higress-controller-higress-system

Role
  higress-controller
  higress-gateway

RoleBinding
  higress-controller
  higress-gateway

Service
  higress-controller
  higress-gateway

Deployment
  higress-controller higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress:2.2.2
  higress-gateway higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway:2.2.2

IngressClass
  higress

EnvoyFilter
  higress-gateway-global-custom-response


helm show crds ./learn/install/helm/higress > ./learn/install/helm/higress_crd.yaml 2>&1

CustomResourceDefinition
  wasmplugins.extensions.higress.io
  http2rpcs.networking.higress.io
  mcpbridges.networking.higress.io
  envoyfilters.networking.istio.io

helm template higress ./learn/install/helm/higress -n higress-system --include-crds

```


### Supporting Istio CRD (Optional)

```bash
helm repo add istio https://istio-release.storage.googleapis.com/charts

$ helm search repo istio
NAME                  	CHART VERSION	APP VERSION	DESCRIPTION
higress.io/istio      	1.12.4       	1.12.4     	Helm chart for deploying higress istio
higress.io/istio-local	1.12.4       	1.12.4     	Helm chart for deploying higress istio
istio/istiod          	1.30.2       	1.30.2     	Helm chart for istio control plane
istio/istiod-remote   	1.23.6       	1.23.6     	Helm chart for a remote cluster using an extern...
istio/ambient         	1.30.2       	1.30.2     	Helm umbrella chart for ambient
istio/base            	1.30.2       	1.30.2     	Helm chart for deploying Istio cluster resource...
istio/cni             	1.30.2       	1.30.2     	Helm chart for istio-cni components
istio/gateway         	1.30.2       	1.30.2     	Helm chart for deploying Istio gateways
istio/ztunnel         	1.30.2       	1.30.2     	Helm chart for istio ztunnel components
$

# helm install istio-base istio/base -n istio-system --create-namespace
helm pull istio/base --version 1.30.2 --destination ./learn/install/helm/

helm template -n istio-system --create-namespace --include-crds istio-base ./learn/install/helm/base > ./learn/install/helm/helmistio-base_all_manifests.yaml 2>&1

ServiceAccount
  istio-reader-service-account

CustomResourceDefinition
  gateways.networking.istio.io
  virtualservices.networking.istio.io
  destinationrules.networking.istio.io
  envoyfilters.networking.istio.io
  proxyconfigs.networking.istio.io
  serviceentries.networking.istio.io
  sidecars.networking.istio.io
  workloadentries.networking.istio.io
  workloadgroups.networking.istio.io
  authorizationpolicies.security.istio.io
  peerauthentications.security.istio.io
  requestauthentications.security.istio.io
  trafficextensions.extensions.istio.io
  wasmplugins.extensions.istio.io
  telemetries.telemetry.istio.io

ValidatingWebhookConfiguration
  istiod-default-validator

helm upgrade higress -n higress-system --set global.enableIstioAPI=true higress.io/higress --reuse-values

```


### Supporting Gateway API CRD (Optional)

```bash

kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.0/experimental-install.yaml

CustomResourceDefinition
  backendtlspolicies.gateway.networking.k8s.io
  gatewayclasses.gateway.networking.k8s.io
  gateways.gateway.networking.k8s.io
  grpcroutes.gateway.networking.k8s.io
  httproutes.gateway.networking.k8s.io
  referencegrants.gateway.networking.k8s.io
  tcproutes.gateway.networking.k8s.io
  tlsroutes.gateway.networking.k8s.io
  udproutes.gateway.networking.k8s.io
  xbackendtrafficpolicies.gateway.networking.x-k8s.io
  xlistenersets.gateway.networking.x-k8s.io
  xmeshes.gateway.networking.x-k8s.io

# --set global.enableGatewayAPI=true
helm install --debug --dry-run=client -n higress-system --create-namespace --set global.enableGatewayAPI=true higress ./learn/install/helm/higress

```

## Docker Compose

```bash
curl -fsSL https://higress.io/standalone/get-higress.sh | bash -s -- [DESTINATION] [OPTIONS...]
```

1. `get-higress.sh` 脚本的作用就是从 `https://github.com/higress-group/higress-standalone/archive/refs/tags/v2.2.2.tar.gz` 下载 `higress-standalone-2.2.2.tar.gz`


## 基于 Docker 进行独立部署

```bash
docker pull higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one:2.2.2

# 创建一个工作目录
mkdir higress;
cd higress
# 启动 higress, 配置文件会写到工作目录下
docker run -d --rm --name higress-ai \
-v ~/work/code/py_code/higress/learn/install/docker/higress:/data \
-p 8001:8001 \
-p 8080:8080 \
-p 8443:8443 \
higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one:2.2.2

# 通过 http://localhost:8001/ 访问 Higress Console 进行配置管理
# 8080: Gateway HTTP 服务端口
# 8443: Gateway HTTPS 服务端口

```

------------------------------------------------------------------------------

Q: 假设你是一位精通 docker 的高级开发人员, 我看到有一个镜像是 `higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one:latest`, 我不想使用这个镜像 tag 是 latest 的版本, 我想使用一个特定的 tag, 但是我使用浏览器看不到这个镜像有哪些 tag, 我有其他办法, 比如使用 API 请求获取到这个镜像有哪些 tag 吗?

在不使用浏览器的情况下, 获取远程镜像仓库中的特定 Tag 是一个非常常见的需求. 由于 Docker CLI 并没有直接提供列出远程仓库 Tag 的原生命令(`docker search` 只能搜索镜像, 无法列出 Tag), 我们可以通过 调用 Docker Registry V2 API 或使用专门的命令行工具来解决这个问题.

以下是为您整理的几种方法, 您可以根据实际环境选择最合适的一种:

---

### 方法一: 使用 API 请求(通过 Shell 脚本和 `curl`)

该镜像托管在阿里云容器镜像服务(ACR)上. 标准 Docker Registry V2 规范在访问 API 时通常需要一个临时的 Bearer Token(即使是公开镜像).

我们可以编写一个 Bash 脚本, 自动获取临时 Token 并查询接口:

```bash
#!/usr/bin/env bash
REGISTRY="higress-registry.cn-hangzhou.cr.aliyuncs.com"
REPO="higress/all-in-one"

# 1. 访问 v2 基础接口, 获取 401 响应头中的认证服务器 URL (realm) 和服务名称 (service)
WWW_AUTH=$(curl -sI "https://${REGISTRY}/v2/" | grep -i "www-authenticate")

REALM=$(echo "$WWW_AUTH" | grep -oE 'realm="[^"]+"' | cut -d'"' -f2)
SERVICE=$(echo "$WWW_AUTH" | grep -oE 'service="[^"]+"' | cut -d'"' -f2)

# 2. 匿名请求临时 Bearer Token
TOKEN=$(curl -s "${REALM}?service=${SERVICE}&scope=repository:${REPO}:pull" | jq -r '.token')

# 3. 携带 Token 请求 Tag 列表并使用 jq 格式化输出
curl -s -H "Authorization: Bearer ${TOKEN}" "https://${REGISTRY}/v2/${REPO}/tags/list" | jq -r '.tags[]'
```

使用前提: 系统需要安装 `curl` 和 `jq`.
执行逻辑: 脚本会自动解析阿里云仓库返回的鉴权地址, 获取到针对 `higress/all-in-one` 的 `pull` 权限 Token, 然后调用 `/tags/list` 接口并输出所有 Tag.

---

### 方法二: 使用 OCI 命令行工具(最推荐, 免去写脚本)

如果您不想编写脚本, 可以利用社区中专门用来处理镜像和 Registry 交互的轻量级工具. 这些工具不需要运行 Docker 守护进程, 也不会真正下载镜像.

#### 1. 使用 `skopeo`

`skopeo` 是红帽(Red Hat)维护的一个专门用于检查、复制和传输镜像的工具.

* 安装:
    * macOS: `brew install skopeo`
    * Ubuntu/Debian: `sudo apt-get install skopeo`
    * CentOS/RHEL: `sudo dnf install skopeo`

* 查询命令:
    ```bash
    skopeo list-tags docker://higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one
    ```
    您也可以结合 `jq` 仅提取 Tag 列表:
    ```bash
    skopeo list-tags docker://higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one | jq -r '.Tags[]'
    ```

```bash
$ skopeo list-tags docker://higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one
{
    "Repository": "higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one",
    "Tags": [
        "1.3.5",
        "1.3.6-rc.1",
        "1.3.6-rc.2",
        "1.3.6-test",
        "1.3.6-test2",
        "1.3.6",
        "1.4.0-rc.1",
        "1.4.0",
        "1.4.1-2",
        "1.4.1-test",
        "1.4.1",
        "1.4.2",
        "2.0.0",
        "2.0.1",
        "2.0.2",
        "2.0.3",
        "2.0.4",
        "2.0.6",
        "2.0.7",
        "2.1.0",
        "2.1.1",
        "2.1.10",
        "2.1.11",
        "2.1.2",
        "2.1.3",
        "2.1.4",
        "2.1.5",
        "2.1.6",
        "2.1.7",
        "2.1.8",
        "2.1.9",
        "2.2.0-o11y",
        "2.2.0",
        "2.2.1-o11y",
        "2.2.1",
        "2.2.2-o11y",
        "2.2.2-rc.1-o11y",
        "2.2.2-rc.1",
        "2.2.2-rc.2-o11y",
        "2.2.2-rc.2",
        "2.2.2",
        "20241029",
        "20241218",
        "20241219-2",
        "20241219",
        "20241225",
        "20241227",
        "20250102",
        "latest-o11y",
        "latest",
        "sha-05d80ef",
        "sha-0c7bef2",
        "sha-1066915",
        "sha-15ec208",
        "sha-25bb17d",
        "sha-276b732",
        "sha-292c975",
        "sha-295ea3c",
        "sha-2bcac50",
        "sha-2fac025",
        "sha-315dd1c",
        "sha-33fae63-o11y",
        "sha-33fae63",
        "sha-3cb15c3",
        "sha-4066dea",
        "sha-487f6a2",
        "sha-4ad5fc8",
        "sha-4e37c5a",
        "sha-5d222ac",
        "sha-5f1e899",
        "sha-68846f5-o11y",
        "sha-68846f5",
        "sha-70f8e5c",
        "sha-730f060",
        "sha-73bab41",
        "sha-74c589c-o11y",
        "sha-74c589c",
        "sha-7b89b57",
        "sha-7ece76c",
        "sha-8161bc8",
        "sha-846be7f-o11y",
        "sha-846be7f",
        "sha-9c23986",
        "sha-9cfd03a",
        "sha-9dedde8",
        "sha-9faa2f2",
        "sha-a3dc0b4",
        "sha-a67f011",
        "sha-a742768-o11y",
        "sha-a742768",
        "sha-a77773c",
        "sha-ae282b4",
        "sha-b320509",
        "sha-b365c06",
        "sha-cba23a9",
        "sha-cd9f72d",
        "sha-d2c8a69",
        "sha-d32debd",
        "sha-d606b1c",
        "sha-e7162db",
        "sha-f5202b5",
        "test-0210",
        "test"
    ]
}
$
```

#### 2. 使用 `regctl`

`regctl`(来自 `regclient` 项目)是另一个非常优秀的 Registry 客户端工具.

* 安装(通过 curl 下载二进制文件):
    ```bash
    curl -L https://github.com/regclient/regclient/releases/latest/download/regctl-linux-amd64 > regctl
    chmod +x regctl
    mv regctl /usr/local/bin/
    ```

* 查询命令:
    ```bash
    regctl tag ls higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one
    ```

---

### 方法三: 通过关联的 GitHub Release 确认(便捷的替代方案)

作为一个开源项目, Higress 的 Docker 镜像 Tag 与其在 GitHub 上的发行版本是严格绑定的.

如果您不想运行任何脚本或安装新工具, 可以直接查看 Higress 官方仓库的 GitHub Releases 或项目的发布分支:
* GitHub Releases 地址: [https://github.com/higress-group/higress/releases](https://github.com/higress-group/higress/releases)

在这个页面上看到的每一个 Release 版本号(例如 `2.2.2`, `2.4.0` 等), 在镜像仓库中一般都有一个同名 Tag. 例如, 若想使用特定版本, 可以直接指定标签:
`higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one:2.2.2`

------------------------------------------------------------------------------

Q: 假设你是一位精通 kubernetes 以及 helm 的高级开发人员, 现在我有如下目录结构的 chart:
```
$ tree -a .
.
├── Chart.lock
├── Chart.yaml
├── LICENSE
├── README.md
├── README.md.gotmpl
├── README.zh.md
└── charts
    ├── higress-console
    │   ├── .helmignore
    │   ├── Chart.yaml
    │   ├── LICENSE
    │   ├── README.md
    │   ├── templates
    │   │   ├── NOTES.txt
    │   │   ├── _helpers.tpl
    │   │   ├── certmanager.yaml
    │   │   ├── clusterrole.yaml
    │   │   ├── clusterrolebinding.yaml
    │   │   ├── configmap.yaml
    │   │   ├── deployment.yaml
    │   │   ├── grafana.yaml
    │   │   ├── ingress.yaml
    │   │   ├── loki.yaml
    │   │   ├── prometheus.yaml
    │   │   ├── role.yaml
    │   │   ├── rolebinding.yaml
    │   │   ├── secret.yaml
    │   │   ├── service.yaml
    │   │   └── serviceaccount.yaml
    │   └── values.yaml
    └── higress-core
        ├── .helmignore
        ├── Chart.yaml
        ├── LICENSE
        ├── README.md
        ├── charts
        │   └── redis
        │       ├── .helmignore
        │       ├── Chart.yaml
        │       ├── templates
        │       │   ├── _helpers.tpl
        │       │   ├── configmap.yaml
        │       │   ├── pvc.yaml
        │       │   ├── service.yaml
        │       │   └── statefulset.yaml
        │       └── values.yaml
        ├── crds
        │   ├── customresourcedefinitions.gen.yaml
        │   └── istio-envoyfilter.yaml
        ├── templates
        │   ├── NOTES.txt
        │   ├── _helpers.tpl
        │   ├── _pod.tpl
        │   ├── clusterrole.yaml
        │   ├── configmap.yaml
        │   ├── controller-clusterrole.yaml
        │   ├── controller-clusterrolebinding.yaml
        │   ├── controller-deployment.yaml
        │   ├── controller-role.yaml
        │   ├── controller-rolebinding.yaml
        │   ├── controller-service.yaml
        │   ├── controller-serviceaccont.yaml
        │   ├── daemonset.yaml
        │   ├── deployment.yaml
        │   ├── fallback-envoyfilter.yaml
        │   ├── hpa.yaml
        │   ├── ingressclass.yaml
        │   ├── plugin-server-deployment.yaml
        │   ├── plugin-server-service.yaml
        │   ├── podmonitor.yaml
        │   ├── promtail.yaml
        │   ├── role.yaml
        │   ├── service.yaml
        │   └── serviceaccount.yaml
        └── values.yaml

10 directories, 66 files
$
```
我使用命令 `helm install --debug --dry-run=client -n higress-system --create-namespace higress ./helm/higress` 调试该 chart 时, 发现并没有输出 crd 里面的内容, 我认为这是不正常的, 请问这是什么原因, 应该如何输出所有即将安装到 kubernetes 里面的内容

针对您在使用 `helm install --dry-run` 时无法输出 `crds/` 目录中内容的问题, 这在 Helm 3 中是完全正常的行为, 并不是您的 Chart 目录结构或者命令有误.

### 为什么 `helm install --dry-run` 不会输出 CRD?

这源于 Helm 3 对自定义资源(CRD)的特殊管理机制:

1. 非模板化设计: 在 Helm 3 中, 放置在 `crds/` 目录下的 YAML 文件被视作静态资源. Helm 不会通过 Go 模板引擎去渲染它们.
2. `--dry-run` 校验局限性: `--dry-run` 的目的是为了在不真实更改集群状态的前提下, 模拟和验证即将部署的 Manifest.
   * 如果 Helm 在 Dry Run 阶段输出并校验 `crds/`, 由于这些 CRD 并不会真正安装进 Kubernetes 的 API Server 中, 这就导致紧随其后的自定义资源(CR, 例如您的 `higress-core` 下依赖这些 CRD 的特定实例)在客户端或服务端校验时, 会因为"找不到对应的 API 资源"而抛出校验错误.
   * 为了规避这种由于缺少 CRD 注册而导致的 Dry Run 阻断, Helm 默认不会在 `install --dry-run` 的渲染输出中包含 `crds/` 目录的内容.

---

### 如何输出所有即将安装到 Kubernetes 的内容(包含 CRD)?

为了能在本地调试并完整查看包括 CRD 在内的所有部署清单, 可以采用以下几种方法:

#### 方法一: 使用 `helm template` 并附带 `--include-crds` 参数(推荐)

这是最常用且标准的方法. `helm template` 命令专门提供了一个参数, 用来显式将 `crds/` 目录的内容合并进最终的 YAML 输出中:

```bash
helm template higress ./helm/higress -n higress-system --include-crds > all_manifests.yaml
```

特点:
* 它会递归遍历您的父 Chart 及所有子 Chart(包括 `higress-core`)中的 `crds/` 目录, 将所有的 CRD 与 `templates/` 下渲染后的普通资源拼接在同一个输出中.
* 由于是本地渲染, 不需要真正连接集群, 非常适合用于审计、GitOps 静态 Manifest 生成或本地调试.

#### 方法二: 使用 `helm show crds` 单独查看

如果您只想确认当前 Chart(包含其子 Chart)究竟包含哪些 CRD 定义, 可以直接运行:

```bash
helm show crds ./helm/higress
```

这会过滤掉其他的 Deployment、Service 等资源, 仅输出 `crds/` 目录中定义的静态 CRD 文本.

---

### 高级开发建议: 在实际生产中如何更好地管理 CRD?

在维护类似于 Higress 这样包含复杂 CRD 的 Chart 时, Helm 的 `crds/` 目录虽然简单(能保证安装顺序在普通资源之前), 但也存在无法通过 Helm 升级和删除的局限(Helm 官方出于数据安全考虑, 默认不对 `crds/` 下的资源做更新和删除).

通常, 业内高级开发人员会采用以下两种进阶方案来解决 CRD 的生命周期管理(以及解决 Dry Run 无法输出的问题):

1. 方案 A: 将 CRD 移入 `templates/` 并配合 `installCRDs` 开关控制(如 `cert-manager` 的做法):
   * 将 CRD 移至 `templates/crds/` 目录下(使其可以被模板化, 并能通过 `install --dry-run` 正常渲染).
   * 编写模板控制逻辑(例如 `{{- if .Values.installCRDs }}`), 并为 CRD 的 Metadata 加上注解以防被误删:
     ```yaml
     metadata:
       annotations:
         helm.sh/resource-policy: keep  # 卸载 Chart 时保留 CRD, 防止关联数据丢失
     ```
2. 方案 B: 将 CRD 拆分为独立的 `higress-crds` 子 Chart(如 `prometheus-operator` 的做法):
   * 将所有的 CRD 独立成一个不含普通资源的 Chart, 在主 Chart 部署前, 引导用户或通过 GitOps 工具先安装该 CRD Chart. 这样不仅可以正常对 CRD 进行 Dry Run 调试, 也使 CRD 的版本生命周期更加清晰.

------------------------------------------------------------------------------

------------------------------------------------------------------------------

------------------------------------------------------------------------------

------------------------------------------------------------------------------

------------------------------------------------------------------------------


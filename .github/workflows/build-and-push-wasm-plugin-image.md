你好! 很高兴再次以高级 GitHub Actions 开发者的身份来为你深度解析这段脚本.

如果说上一份脚本是"标准的现代容器构建教科书", 那么这份脚本则是"极其硬核的 OCI 制品(Artifacts)与多语言 Wasm 编译的高级实践".

这段代码并没有构建传统的 Docker 镜像(即跑服务的镜像), 而是将 WebAssembly (Wasm) 插件作为 OCI 制品(OCI Artifacts) 推送到镜像仓库. 它巧妙地使用了 bash 字符串截取、后台构建容器(Builder Container)模式以及 `oras` 工具.

我依然按照两种方式为你解答:
1. 带详细注释的代码(重点讲语法和目的).
2. 高级开发者视角的深度总结(重点讲为什么要这么设计架构).

---

### 一、带详细注释的 Workflow 脚本

```yaml
# workflow 的名称
name: Build and Push Wasm Plugin Image

on:
  # 触发条件 1: 根据特定的 Tag 格式自动触发
  push:
    tags:
      # 语法: 使用通配符匹配 Wasm 插件发布的特定命名规范.
      # 例如: wasm-go-ratelimit-v1.0.0 或 wasm-rust-cors-v2.1.3
      # 目的: 在一个大仓库(Monorepo)中, 有多个插件、不同语言, 通过 Tag 前缀精确区分发布的是哪个插件.
      - "wasm-*-*-v*.*.*"

  # 触发条件 2: 网页端手动触发
  workflow_dispatch:
    inputs:
      plugin_type:
        description: "Type of the plugin"
        required: true
        type: choice # 语法: 提供一个下拉菜单供用户选择, 防止拼写错误
        options:
          - go
          - rust
      plugin_name:
        description: "Name of the plugin"
        required: true
        type: string
      version:
        description: "Version of the plugin (optional, without leading v)"
        required: false
        type: string

jobs:
  build-and-push-wasm-plugin-image:
    runs-on: ubuntu-latest
    # 绑定审批和环境变量上下文
    environment:
      name: image-registry-msg
    env:
      # 语法: 读取后台配置 Variables, 若无则使用默认值.
      IMAGE_REGISTRY_SERVICE: ${{ vars.IMAGE_REGISTRY || 'higress-registry.cn-hangzhou.cr.aliyuncs.com' }}
      IMAGE_REPOSITORY: ${{ vars.PLUGIN_IMAGE_REPOSITORY || 'plugins' }}
      # 目的: 将编译工具链的版本号抽离成环境变量, 方便后续升级维护, 避免代码中到处硬编码.
      RUST_VERSION: 1.82
      GO_VERSION: 1.24.0
      ORAS_VERSION: 1.0.0

    steps:
      # 步骤 1: 解析入参或 Tag, 提取插件类型、名称和版本, 并计算构建环境
      - name: Set plugin_type, plugin_name and version from inputs or ref_name
        id: set_vars
        run: |
          # 语法: 根据 github.event_name 判断触发来源
          if [[ "${{ github.event_name }}" == "workflow_dispatch" ]]; then
            # 手动触发: 直接读取表单输入
            plugin_type="${{ github.event.inputs.plugin_type }}"
            plugin_name="${{ github.event.inputs.plugin_name }}"
            version="${{ github.event.inputs.version }}"
          else
            # Tag 触发: 使用纯 Bash 的字符串操作(Parameter Expansion)来解析 Tag.
            # 假设 github.ref_name 为 "wasm-go-myplugin-v1.0.0"
            ref_name=${{ github.ref_name }}

            # ${var#pattern} 语法: 从左边开始, 删除最短匹配.
            # "wasm-go-myplugin-v1.0.0" 变成 "go-myplugin-v1.0.0"
            plugin_type=${ref_name#*-}

            # ${var%%pattern} 语法: 从右边开始, 删除最长匹配.
            # "go-myplugin-v1.0.0" 变成 "go"
            plugin_type=${plugin_type%%-*}

            # 删除前两个破折号前的内容: "wasm-go-myplugin-v1.0.0" 变成 "myplugin-v1.0.0"
            plugin_name=${ref_name#*-*-}

            # ${var%pattern} 语法: 从右边开始, 删除最短匹配.
            # "myplugin-v1.0.0" 变成 "myplugin"
            plugin_name=${plugin_name%-*}

            # 使用 awk 以 'v' 作为分隔符提取版本号: "v1.0.0" 提取出 "1.0.0"
            version=$(echo "$ref_name" | awk -F'v' '{print $2}')
          fi

          # 目的: 根据前面提取出来的 plugin_type (go/rust), 动态决定使用哪个基础编译镜像
          if [[ "$plugin_type" == "rust" ]]; then
            builder_image="higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-rust-builder:rust${{ env.RUST_VERSION }}-oras${{ env.ORAS_VERSION }}"
          else
            builder_image="higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go${{ env.GO_VERSION }}-oras${{ env.ORAS_VERSION }}"
          fi

          # 语法: 将计算出的变量写入 $GITHUB_ENV.
          # 目的: 因为不同 Step 之间的 Bash 进程是隔离的, 这样才能在后面的 Step 中使用这些变量.
          echo "PLUGIN_TYPE=$plugin_type" >> $GITHUB_ENV
          echo "PLUGIN_NAME=$plugin_name" >> $GITHUB_ENV
          echo "VERSION=$version" >> $GITHUB_ENV
          echo "BUILDER_IMAGE=$builder_image" >> $GITHUB_ENV

      - name: Checkout code
        uses: actions/checkout@v3

      # 步骤 2: 组装 ORAS push 命令参数
      # 目的: Wasm 插件不是普通的镜像, 通常通过 OCI Artifacts 标准推送.
      # ORAS (OCI Registry As Storage) 允许我们把普通的配置文件、文档和 wasm 二进制包以特定 MediaType 形式推送到镜像仓库.
      - name: File Check
        run: |
          workspace=${{ github.workspace }}/plugins/wasm-${PLUGIN_TYPE}/extensions/${PLUGIN_NAME}
          # 基础推送内容: 编译好的插件压缩包, 指定了标准的 OCI 媒体类型
          push_command="./plugin.tar.gz:application/vnd.oci.image.layer.v1.tar+gzip"

          # 自动发现机制: 检查并打包 spec.yaml 配置文件
          if [ -f "${workspace}/spec.yaml" ]; then
            echo "spec.yaml exists"
            push_command="./spec.yaml:application/vnd.module.wasm.spec.v1+yaml $push_command "
          fi

          # 自动发现机制: 检查并打包主 README 文档
          if [ -f "${workspace}/README.md" ];then
              echo "README.md exists"
              push_command="./README.md:application/vnd.module.wasm.doc.v1+markdown $push_command "
          fi

          # 自动发现机制: 检查并打包多语言 README 文档(如 README_zh.md)
          for file in ${workspace}/README_*.md; do
            if [ -f "$file" ]; then
              file_name=$(basename $file)
              echo "$file_name exists"
              # 提取语言标识
              lang=$(basename $file | sed 's/README_//; s/.md//')
              # 动态生成包含语言标识的媒体类型
              push_command="./$file_name:application/vnd.module.wasm.doc.v1.$lang+markdown $push_command "
            fi
          done

          # 保存组装好的超长参数字符串, 供下文执行
          echo "PUSH_COMMAND=\"$push_command\"" >> $GITHUB_ENV

      # 步骤 3: 启动后台编译容器 (Builder Container)
      # 目的: GitHub 默认的虚拟机可能没有安装我们需要的特定版本的 Go/Rust/Wasm-toolchain/ORAS.
      # 使用 `docker run -itd` 在后台启动一个包含完整依赖的容器, 挂载当前代码.
      # 这种做法比在 Runner 上通过 apt-get 慢慢安装环境快得多, 也更纯净.
      - name: Run a wasm-builder
        env:
          PLUGIN_NAME: ${{ env.PLUGIN_NAME }}
          BUILDER_IMAGE: ${{ env.BUILDER_IMAGE }}
        run: |
          # -i (交互式), -t (伪终端), -d (后台运行)
          # -v 挂载本地 workspace 到容器内, 使得容器编译出的产物可以直接保存在 GitHub runner 的磁盘上
          docker run -itd --name builder -v ${{ github.workspace }}:/workspace -e PLUGIN_NAME=${{ env.PLUGIN_NAME }} --rm ${{ env.BUILDER_IMAGE }} /bin/bash

      # 步骤 4: 在容器内执行具体的编译和推送操作
      - name: Build Image and Push
        run: |
          push_command=${{ env.PUSH_COMMAND }}
          # 语法: 清理掉包裹变量的双引号, 防止 ORAS CLI 解析参数时出错
          push_command=${push_command#\"}
          push_command=${push_command%\"}

          # 定义要推送的远端地址, 包含明确版本号和 latest 版
          target_image="${{ env.IMAGE_REGISTRY_SERVICE }}/${{ env.IMAGE_REPOSITORY}}/${{ env.PLUGIN_NAME }}:${{ env.VERSION }}"
          target_image_latest="${{ env.IMAGE_REGISTRY_SERVICE }}/${{ env.IMAGE_REPOSITORY}}/${{ env.PLUGIN_NAME }}:latest"

          cd ${{ github.workspace }}/plugins/wasm-${PLUGIN_TYPE}/extensions/${PLUGIN_NAME}

          # 提供给开发者自定义构建环境的钩子(Hook)
          # 如果插件目录下有 .buildrc, 则 sourcing 加载它(可能包含特殊的 go proxy、私有库认证等环境变量)
          if [ -f ./.buildrc ]; then
            echo 'Found .buildrc file, sourcing it...'
            . ./.buildrc
          fi

          # 构建脚本生成: 根据不同的语言, 组装一段将在 Builder 容器中执行的 Bash 脚本
          if [ "${PLUGIN_TYPE}" == "go" ]; then
          command="
          set -e # 遇到错误立即退出
          cd /workspace/plugins/wasm-go/extensions/${PLUGIN_NAME}
          go mod tidy
          # 核心编译: GOOS=wasip1 GOARCH=wasm 指定交叉编译目标为 WebAssembly
          # -buildmode=c-shared 用于生成符合规范的 C ABI wasm
          GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm .
          tar czvf plugin.tar.gz plugin.wasm

          # 登录并将构建好的制品推送到镜像仓库
          echo ${{ secrets.REGISTRY_PASSWORD }} | oras login -u ${{ secrets.REGISTRY_USERNAME }} --password-stdin ${{ env.IMAGE_REGISTRY_SERVICE }}
          oras push ${target_image} ${push_command}
          oras push ${target_image_latest} ${push_command}
          "
          elif [ "${PLUGIN_TYPE}" == "rust" ]; then
          command="
          set -e
          cd /workspace/plugins/wasm-rust/extensions/${PLUGIN_NAME}
          # 提供 Rust 专用的预编译 Hook
          if [ -f ./.prebuild ]; then
            echo 'Found .prebuild file, sourcing it...'
            . ./.prebuild
          fi
          # 添加 Wasm 编译目标
          rustup target add wasm32-wasip1
          # 核心编译: Release 模式构建 wasm32-wasip1
          cargo build --target wasm32-wasip1 --release
          # 提取产物并打包
          cp target/wasm32-wasip1/release/*.wasm plugin.wasm
          tar czvf plugin.tar.gz plugin.wasm

          # 使用 ORAS 推送
          echo ${{ secrets.REGISTRY_PASSWORD }} | oras login -u ${{ secrets.REGISTRY_USERNAME }} --password-stdin ${{ env.IMAGE_REGISTRY_SERVICE }}
          oras push ${target_image} ${push_command}
          oras push ${target_image_latest} ${push_command}
          "
          else
          command="echo "unkown type ${PLUGIN_TYPE}""
          fi

          # 语法: 使用 docker exec 进入前一个 Step 启动的名为 'builder' 的后台容器,
          # 将上面拼接好的 $command 作为参数传给 bash -c 执行!
          docker exec builder bash -c "$command"
          # 制品: higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/mcp-server:2.0.0
          # 制品: higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/mcp-server:latest
```

---

### 二、高级开发者视角总结(为什么这么写?)

这是一份为云原生网关(如 Higress/Envoy)自动化分发 Wasm 插件量身定制的顶级 CI 脚本, 体现了几个非常高级的架构设计思想:

#### 1. 极致的大仓库(Monorepo)支持策略
在一个包含几十个不同 Wasm 插件、用不同语言(Go/Rust)编写的大型仓库里, 如何只发布变更的插件?
* 巧妙的 Tag 命名法: 作者规定了 `wasm-<语言>-<插件名>-<版本号>` 的 Tag 规范(例如 `wasm-go-jwt-auth-v1.0.0`).
* 纯 Bash 的解构: 没有使用复杂的 Python 或 JS 脚本, 单纯利用 Bash 内置的 Parameter Expansion (`#`, `%%` 等) 极速解构 Tag. 这使得一次 Release Tag 动作, 就能让 CI 明确知道该用什么语言工具链、去哪个目录编译哪个插件、推送到哪里.

#### 2. "Builder 容器化" 模式 (Docker-in-Runner)
* 痛点: GitHub 官方的 Runner 安装 Go/Rust 很容易, 但 Wasm 交叉编译通常还需要特定的 C 工具链(如 `tinygo`, 或针对 `wasip1` 的特定配置), 还要额外安装非标准 CLI `oras`.
* 解法: 作者并没有在 `ubuntu-latest` 上用 `apt-get` 慢慢装环境, 而是直接启动了一个准备好一切的官方镜像(`wasm-go-builder` / `wasm-rust-builder`), 利用 `docker run -itd` 放后台, 然后用 `docker exec` 把编译命令丢进容器执行. 速度极快、环境极度一致、本地开发复现也很容易.

#### 3. 拥抱 OCI Artifacts 与 ORAS
这是这段代码最硬核的地方.
* 为什么不用 docker build? Wasm 插件不是一个完整的操作系统容器环境(没有 rootfs, 不能被 docker run). 它本质上只是一段 `.wasm` 二进制代码和配套的配置文档.
* 怎么分发呢? 现代云原生选择了 OCI Artifacts 标准. 利用 `oras` (OCI Registry As Storage) 客户端, 镜像仓库(如阿里云 ACR、Harbor)可以直接存储 Wasm 文件.
* 精细的 MediaType(媒体类型)定义: 代码中长长的 `application/vnd.module.wasm.doc.v1.zh+markdown` 不是随便写的. 当把产物推送到仓库后, 网关控制面或插件市场(Plugin Market)就能通过这些特殊的类型声明, 知道哪个文件是 Wasm 二进制, 哪个文件是中文说明, 哪个是配置规范, 从而实现 UI 展示和动态加载.

#### 4. 高扩展性的 "Hooks" 机制
* `.buildrc` 和 `.prebuild`: 作者考虑到个别插件可能需要特殊的依赖(比如 CGO 需要特定的 C 库, 或者内网需要设置 `GOPROXY`), 因此留下了这几个隐藏文件的 Hook. 只要插件目录下有这些文件, CI 就会在编译前执行它们. 这也是开源框架中典型的"约定大于配置"的高级设计.


---------------------------------------------------------------------------


这份 GitHub Action 脚本用于自动构建、打包并推送 Wasm(WebAssembly)插件. 下面为您详细解答什么是 Wasm 编译、为什么能将其推送到镜像仓库, 以及后续如何使用.

---

### 一、什么是 Wasm 编译?

Wasm (WebAssembly) 是一种可移植、体积小、加载快的二进制格式. 它最初设计用于浏览器端以接近原生的速度运行代码, 但如今已广泛应用于服务器端、边缘计算以及网络代理(如 Envoy 和 Higress 网关)的插件扩展系统(通常遵循 Proxy-Wasm 规范).

Wasm 编译指的是将高级语言(如 Go、Rust、C++)编写的源代码, 编译成平台无关的 `.wasm` 字节码文件, 而不是特定操作系统(如 Linux x86_64 或 Windows ARM)的二进制可执行文件.

在您的脚本中, 这一编译过程分别针对两种语言进行:

* 对于 Go 语言:
  ```bash
  GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm .
  ```
  通过设置目标操作系统为 `wasip1`(WebAssembly System Interface), 目标架构为 `wasm`, Go 编译器会生成一个可在任何支持 WASI 规范的虚拟机中运行的 `plugin.wasm` 文件.

* 对于 Rust 语言:
  ```bash
  rustup target add wasm32-wasip1
  cargo build --target wasm32-wasip1 --release
  ```
  首先安装 Wasm 编译目标 `wasm32-wasip1`, 然后通过 Cargo 构建出 Release 版本的 Wasm 字节码.

---

### 二、为什么可以将这个制品推送到镜像仓库?

传统的镜像仓库(如 Docker Registry)主要用来存储由多个 Rootfs 层打包而成的容器镜像. 但在云原生技术的发展过程中, 这一标准被进一步抽象.

1. OCI 规范与 OCI Artifacts
   为了打破"镜像仓库只能存 Docker 镜像"的限制, 开放容器计划(OCI)推出了 OCI Artifacts 概念. 它允许任何类型的文件(如 Helm Charts、Wasm 模块、OPA 策略等)利用现有的镜像分发协议(分发、版本控制、鉴权)存储到符合 OCI 标准的镜像仓库中.

2. ORAS 工具的使用
   脚本中使用的 `oras`(OCI Registry As Storage)工具, 就是实现这一目的的标准客户端.
   ```bash
   oras push ${target_image} ${push_command}
   ```
   这里的 `${push_command}` 实际上定义了以下内容:
   * `./plugin.tar.gz` 对应的 Media Type(媒体类型)是 `application/vnd.oci.image.layer.v1.tar+gzip`.
   * 此外, 根据检测到的文件, 它可能还会附加 `./spec.yaml` 和 `./README.md`, 并打上特定的自定义 OCI Media Type.

镜像仓库会像对待普通 Docker 镜像一样接收这些层(Blobs), 并为它们生成一个 Manifest(清单). 通过这种方式, Wasm 插件便获得了版本标记(Tags)、访问控制(Auth)以及分发加速(Cache/CDN)等传统镜像拥有的全部红利.

---

### 三、推送到仓库之后后续如何使用这个制品?

在 Higress 或 Envoy 这类支持 Wasm 插件的云原生网关中, 该制品通常按以下流程被拉取和执行:

#### 1. 声明式配置(部署)
在 Kubernetes 环境中, 管理员可以通过创建 `WasmPlugin` 资源(自定义 CRD)来声明要使用的插件:
```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: my-custom-plugin
  namespace: higress-system
spec:
  # 引用刚刚推送到镜像仓库的 OCI 镜像地址
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/my-plugin:1.0.0
  defaultConfig:
    # 插件所需的自定义业务配置
    some_key: "some_value"
```

#### 2. 网关动态拉取与解析
* 当 Higress 控制面(Controller)检测到上述 `WasmPlugin` 配置后, 会解析 `url` 中的 OCI 地址.
* 网关组件会使用类似 OCI 客户端的机制连接镜像仓库, 校验权限, 并将 `plugin.tar.gz` 及 `spec.yaml` 等制品下载到本地.
* 解压出其中的 `plugin.wasm` 文件.

#### 3. 热加载与运行
* 沙箱隔离运行: 网关内部(基于 Envoy 的 Proxy-Wasm 机制)会为该插件创建一个独立的 Wasm 虚拟机(VM)沙箱.
* 无损热更新: 该插件可以在不重启网关、不中断网络连接的情况下, 直接热加载到内存中生效.
* 请求拦截处理: 当有外部 HTTP 请求通过网关时, 网关会调用 Wasm 虚拟机中导出的对应钩子函数(例如 `on_http_request_headers`), 执行您在 Go 或 Rust 中编写的拦截、鉴权、改写等逻辑, 完成流量管理.

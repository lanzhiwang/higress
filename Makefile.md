
```bash
docker run -ti --rm --name higress \
--network host \
-v ~/work/code/py_code/higress:/higress \
-w /higress lanzhiwang/python:sha-07d7efa bash

apt-get update && apt-get install -y build-essential

# 调试 Makefile
$ export BUILD_WITH_CONTAINER=1 && \
export GENERATE_API=1 && \
export DOCKER_RUN_OPTIONS="--security-opt seccomp=unconfined" && \
export GOPROXY="https://goproxy.cn,direct" && \
make \
BUILD_WITH_CONTAINER=1 \
GENERATE_API=1 \
DOCKER_RUN_OPTIONS="--security-opt seccomp=unconfined" \
GOPROXY="https://goproxy.cn,direct" \
DEBUG=1 \
-pq -n --dry-run > Makefile_database.txt 2>&1

$ export BUILD_WITH_CONTAINER=1 && \
export GENERATE_API=1 && \
export DOCKER_RUN_OPTIONS="--security-opt seccomp=unconfined" && \
export GOPROXY="https://goproxy.cn,direct" && \
make \
BUILD_WITH_CONTAINER=1 \
GENERATE_API=1 \
DOCKER_RUN_OPTIONS="--security-opt seccomp=unconfined" \
GOPROXY="https://goproxy.cn,direct" \
DEBUG=0 \
-n --dry-run
++ uname -m
+ LOCAL_ARCH=x86_64
+ export LOCAL_ARCH
+ [[ -n '' ]]
+ [[ x86_64 == x86_64 ]]
+ export TARGET_ARCH=amd64
+ TARGET_ARCH=amd64
++ uname
+ LOCAL_OS=Linux
+ export LOCAL_OS
+ [[ -n '' ]]
+ [[ Linux == Linux ]]
+ export TARGET_OS=linux
+ TARGET_OS=linux
+ readlink_flags=-f
+ [[ '' == '' ]]
+ export IMAGE_VERSION=34b06c08ee613a15e08c5888ac269ad22f23d23e
+ IMAGE_VERSION=34b06c08ee613a15e08c5888ac269ad22f23d23e
+ [[ '' == '' ]]
+ export IMAGE_NAME=build-tools
+ IMAGE_NAME=build-tools
+ export UID
++ grep '^docker:' /etc/group
++ cut -f3 -d:
+ DOCKER_GID=1002
+ export DOCKER_GID
++ readlink -f /etc/localtime
++ sed -e 's/^.*zoneinfo\///'
+ TIMEZONE=Asia/Shanghai
+ export TIMEZONE
++ pwd
+ export TARGET_OUT=/root/huzhi/higress/out/linux_amd64
+ TARGET_OUT=/root/huzhi/higress/out/linux_amd64
++ pwd
+ export TARGET_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
+ TARGET_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
++ pwd
+ export AMD64_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
+ AMD64_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
++ pwd
+ export ARM64_OUT_LINUX=/root/huzhi/higress/out/linux_arm64
+ ARM64_OUT_LINUX=/root/huzhi/higress/out/linux_arm64
+ export CONTAINER_TARGET_OUT=/work/out/linux_amd64
+ CONTAINER_TARGET_OUT=/work/out/linux_amd64
+ export CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
+ CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
+ export IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:34b06c08ee613a15e08c5888ac269ad22f23d23e
+ IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:34b06c08ee613a15e08c5888ac269ad22f23d23e
+ export CONTAINER_CLI=docker
+ CONTAINER_CLI=docker
+ export 'ENV_BLOCKLIST=^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
+ ENV_BLOCKLIST='^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
++ declare -F -x
++ cut -d ' ' -f 3
+ export CONDITIONAL_HOST_MOUNTS=
+ CONDITIONAL_HOST_MOUNTS=
+ container_kubeconfig=
+ [[ -d /root/.docker ]]
+ CONDITIONAL_HOST_MOUNTS+='--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
+ [[ -d /root/.config/gcloud ]]
+ [[ -f /root/.gitconfig ]]
+ [[ -f /root/.netrc ]]
+ KUBECONFIG=/root/.kube/config
+ parse_KUBECONFIG /root/.kube/config
+ TMPDIR=
+ [[ /root/.kube/config =~ ([^:]*):(.*) ]]
+ add_KUBECONFIG_if_exists /root/.kube/config
+ [[ -f /root/.kube/config ]]
+ [[ 1 -eq 1 ]]
+ export KUBECONFIG=
+ KUBECONFIG=
+ export BUILD_WITH_CONTAINER=0
+ BUILD_WITH_CONTAINER=0
+ [[ '' == \e\n\v\f\i\l\e ]]
./tools/hack/run.sh make --no-print-directory -e -f Makefile.core.mk


# 调试 Makefile.core.mk
# 1. 修改 ./tools/hack/run.sh 直接在容器中运行
$ export BUILD_WITH_CONTAINER=1 && \
export GENERATE_API=1 && \
export DOCKER_RUN_OPTIONS="--security-opt seccomp=unconfined" && \
export GOPROXY="https://goproxy.cn,direct" && \
make \
BUILD_WITH_CONTAINER=1 \
GENERATE_API=1 \
DOCKER_RUN_OPTIONS="--security-opt seccomp=unconfined" \
GOPROXY="https://goproxy.cn,direct" \
DEBUG=0
++ uname -m
+ LOCAL_ARCH=x86_64
+ export LOCAL_ARCH
+ [[ -n '' ]]
+ [[ x86_64 == x86_64 ]]
+ export TARGET_ARCH=amd64
+ TARGET_ARCH=amd64
++ uname
+ LOCAL_OS=Linux
+ export LOCAL_OS
+ [[ -n '' ]]
+ [[ Linux == Linux ]]
+ export TARGET_OS=linux
+ TARGET_OS=linux
+ readlink_flags=-f
+ [[ '' == '' ]]
+ export IMAGE_VERSION=34b06c08ee613a15e08c5888ac269ad22f23d23e
+ IMAGE_VERSION=34b06c08ee613a15e08c5888ac269ad22f23d23e
+ [[ '' == '' ]]
+ export IMAGE_NAME=build-tools
+ IMAGE_NAME=build-tools
+ export UID
++ grep '^docker:' /etc/group
++ cut -f3 -d:
+ DOCKER_GID=1002
+ export DOCKER_GID
++ readlink -f /etc/localtime
++ sed -e 's/^.*zoneinfo\///'
+ TIMEZONE=Asia/Shanghai
+ export TIMEZONE
++ pwd
+ export TARGET_OUT=/root/huzhi/higress/out/linux_amd64
+ TARGET_OUT=/root/huzhi/higress/out/linux_amd64
++ pwd
+ export TARGET_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
+ TARGET_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
++ pwd
+ export AMD64_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
+ AMD64_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
++ pwd
+ export ARM64_OUT_LINUX=/root/huzhi/higress/out/linux_arm64
+ ARM64_OUT_LINUX=/root/huzhi/higress/out/linux_arm64
+ export CONTAINER_TARGET_OUT=/work/out/linux_amd64
+ CONTAINER_TARGET_OUT=/work/out/linux_amd64
+ export CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
+ CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
+ export IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:34b06c08ee613a15e08c5888ac269ad22f23d23e
+ IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:34b06c08ee613a15e08c5888ac269ad22f23d23e
+ export CONTAINER_CLI=docker
+ CONTAINER_CLI=docker
+ export 'ENV_BLOCKLIST=^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
+ ENV_BLOCKLIST='^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
++ declare -F -x
++ cut -d ' ' -f 3
+ export CONDITIONAL_HOST_MOUNTS=
+ CONDITIONAL_HOST_MOUNTS=
+ container_kubeconfig=
+ [[ -d /root/.docker ]]
+ CONDITIONAL_HOST_MOUNTS+='--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
+ [[ -d /root/.config/gcloud ]]
+ [[ -f /root/.gitconfig ]]
+ [[ -f /root/.netrc ]]
+ KUBECONFIG=/root/.kube/config
+ parse_KUBECONFIG /root/.kube/config
+ TMPDIR=
+ [[ /root/.kube/config =~ ([^:]*):(.*) ]]
+ add_KUBECONFIG_if_exists /root/.kube/config
+ [[ -f /root/.kube/config ]]
+ [[ 1 -eq 1 ]]
+ export KUBECONFIG=
+ KUBECONFIG=
+ export BUILD_WITH_CONTAINER=0
+ BUILD_WITH_CONTAINER=0
+ [[ '' == \e\n\v\f\i\l\e ]]
++ dirname ./tools/hack/run.sh
+ WD=./tools/hack
++ cd ./tools/hack
++ pwd
+ WD=/root/huzhi/higress/tools/hack
+ source /root/huzhi/higress/tools/hack/setup_env.sh
++ set -ex
+++ uname -m
++ LOCAL_ARCH=x86_64
++ export LOCAL_ARCH
++ [[ -n '' ]]
++ [[ x86_64 == x86_64 ]]
++ export TARGET_ARCH=amd64
++ TARGET_ARCH=amd64
+++ uname
++ LOCAL_OS=Linux
++ export LOCAL_OS
++ [[ -n '' ]]
++ [[ Linux == Linux ]]
++ export TARGET_OS=linux
++ TARGET_OS=linux
++ readlink_flags=-f
++ [[ release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613 == '' ]]
++ [[ '' == '' ]]
++ export IMAGE_NAME=build-tools
++ IMAGE_NAME=build-tools
++ export UID
+++ grep '^docker:' /etc/group
+++ cut -f3 -d:
++ DOCKER_GID=1002
++ export DOCKER_GID
+++ readlink -f /etc/localtime
+++ sed -e 's/^.*zoneinfo\///'
++ TIMEZONE=Asia/Shanghai
++ export TIMEZONE
+++ pwd
++ export TARGET_OUT=/root/huzhi/higress/out/linux_amd64
++ TARGET_OUT=/root/huzhi/higress/out/linux_amd64
+++ pwd
++ export TARGET_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
++ TARGET_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
+++ pwd
++ export AMD64_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
++ AMD64_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
+++ pwd
++ export ARM64_OUT_LINUX=/root/huzhi/higress/out/linux_arm64
++ ARM64_OUT_LINUX=/root/huzhi/higress/out/linux_arm64
++ export CONTAINER_TARGET_OUT=/work/out/linux_amd64
++ CONTAINER_TARGET_OUT=/work/out/linux_amd64
++ export CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
++ CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
++ export IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
++ IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
++ export CONTAINER_CLI=docker
++ CONTAINER_CLI=docker
++ export 'ENV_BLOCKLIST=^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
++ ENV_BLOCKLIST='^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
+++ declare -F -x
+++ cut -d ' ' -f 3
++ export CONDITIONAL_HOST_MOUNTS=
++ CONDITIONAL_HOST_MOUNTS=
++ container_kubeconfig=
++ [[ -d /root/.docker ]]
++ CONDITIONAL_HOST_MOUNTS+='--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly '
++ [[ -d /root/.config/gcloud ]]
++ [[ -f /root/.gitconfig ]]
++ [[ -f /root/.netrc ]]
++ KUBECONFIG=/root/.kube/config
++ parse_KUBECONFIG /root/.kube/config
++ TMPDIR=
++ [[ /root/.kube/config =~ ([^:]*):(.*) ]]
++ add_KUBECONFIG_if_exists /root/.kube/config
++ [[ -f /root/.kube/config ]]
++ [[ 1 -eq 1 ]]
++ export KUBECONFIG=
++ KUBECONFIG=
++ export BUILD_WITH_CONTAINER=0
++ BUILD_WITH_CONTAINER=0
++ [[ make == \e\n\v\f\i\l\e ]]
+ export TARGET_OUT=/work/out/linux_amd64
+ TARGET_OUT=/work/out/linux_amd64
+ export TARGET_OUT_LINUX=/work/out/linux_amd64
+ TARGET_OUT_LINUX=/work/out/linux_amd64
+ export REPO_ROOT=/work
+ REPO_ROOT=/work
+ HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress
+ MOUNT_SOURCE=/root/huzhi/higress
+ MOUNT_DEST=/work
+ read -ra DOCKER_RUN_OPTIONS
+ [[ -t 1 ]]
+ DOCKER_RUN_OPTIONS+=("-it")
+ docker run --rm --entrypoint /usr/bin/env --security-opt seccomp=unconfined -it --init --sig-proxy=true -v /var/run/docker.sock:/var/run/docker.sock --mount type=bind,source=/tmp,destination=/tmp --net=host --env-file /dev/fd/63 -e IN_BUILD_CONTAINER=1 -e TZ=Asia/Shanghai -e HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress --mount type=bind,source=/root/huzhi/higress,destination=/work --mount type=volume,source=go,destination=/go --mount type=volume,source=gocache,destination=/gocache --mount type=volume,source=cache,destination=/home/.cache --mount type=bind,source=/root/.docker,destination=/config/.docker,readonly -w /work higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613 bash
++ env
++ grep -v '^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
build-tools:/work#
build-tools:/work#
build-tools:/work# env
CONTAINER_CLI=docker
CONDA_EXE=/root/anaconda3/bin/conda
HOSTNAME=k8s-a40-node02
AMD64_OUT_LINUX=/root/huzhi/higress/out/linux_amd64
TARGET_OS=linux
REPO_ROOT=/work
HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress
CONTAINER_OPTIONS=--mount type=bind,source=/tmp,destination=/tmp --net=host
GNUMAKEFLAGS=
GOBIN=/gobin
PWD=/work
BUILD_WITH_CONTAINER=0
GSETTINGS_SCHEMA_DIR=/root/anaconda3/share/glib-2.0/schemas
LOGNAME=root
XDG_SESSION_TYPE=tty
CONDA_PREFIX=/root/anaconda3
ARM64_OUT_LINUX=/root/huzhi/higress/out/linux_arm64
GSETTINGS_SCHEMA_DIR_CONDA_BACKUP=
MAKEOVERRIDES=${-*-command-variables-*-}
LOCAL_OS=Linux
GORACE=atexit_sleep_ms=0
TZ=Asia/Shanghai
CURDIR=/root/huzhi/higress
MOTD_SHOWN=pam
HOME=/home
LANG=C.UTF-8
DOCKER_GID=1002
LS_COLORS=rs=0:di=01;34:ln=01;36:mh=00:pi=40;33:so=01;35:do=01;35:bd=40;33;01:cd=40;33;01:or=40;31;01:mi=00:su=37;41:sg=30;43:ca=30;41:tw=30;42:ow=34;42:st=37;44:ex=01;32:*.tar=01;31:*.tgz=01;31:*.arc=01;31:*.arj=01;31:*.taz=01;31:*.lha=01;31:*.lz4=01;31:*.lzh=01;31:*.lzma=01;31:*.tlz=01;31:*.txz=01;31:*.tzo=01;31:*.t7z=01;31:*.zip=01;31:*.z=01;31:*.dz=01;31:*.gz=01;31:*.lrz=01;31:*.lz=01;31:*.lzo=01;31:*.xz=01;31:*.zst=01;31:*.tzst=01;31:*.bz2=01;31:*.bz=01;31:*.tbz=01;31:*.tbz2=01;31:*.tz=01;31:*.deb=01;31:*.rpm=01;31:*.jar=01;31:*.war=01;31:*.ear=01;31:*.sar=01;31:*.rar=01;31:*.alz=01;31:*.ace=01;31:*.zoo=01;31:*.cpio=01;31:*.7z=01;31:*.rz=01;31:*.cab=01;31:*.wim=01;31:*.swm=01;31:*.dwm=01;31:*.esd=01;31:*.jpg=01;35:*.jpeg=01;35:*.mjpg=01;35:*.mjpeg=01;35:*.gif=01;35:*.bmp=01;35:*.pbm=01;35:*.pgm=01;35:*.ppm=01;35:*.tga=01;35:*.xbm=01;35:*.xpm=01;35:*.tif=01;35:*.tiff=01;35:*.png=01;35:*.svg=01;35:*.svgz=01;35:*.mng=01;35:*.pcx=01;35:*.mov=01;35:*.mpg=01;35:*.mpeg=01;35:*.m2v=01;35:*.mkv=01;35:*.webm=01;35:*.webp=01;35:*.ogm=01;35:*.mp4=01;35:*.m4v=01;35:*.mp4v=01;35:*.vob=01;35:*.qt=01;35:*.nuv=01;35:*.wmv=01;35:*.asf=01;35:*.rm=01;35:*.rmvb=01;35:*.flc=01;35:*.avi=01;35:*.fli=01;35:*.flv=01;35:*.gl=01;35:*.dl=01;35:*.xcf=01;35:*.xwd=01;35:*.yuv=01;35:*.cgm=01;35:*.emf=01;35:*.ogv=01;35:*.ogx=01;35:*.aac=00;36:*.au=00;36:*.flac=00;36:*.m4a=00;36:*.mid=00;36:*.midi=00;36:*.mka=00;36:*.mp3=00;36:*.mpc=00;36:*.ogg=00;36:*.ra=00;36:*.wav=00;36:*.oga=00;36:*.opus=00;36:*.spx=00;36:*.xspf=00;36:
GOSUMDB=sum.golang.org
CONDITIONAL_HOST_MOUNTS=--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly
CONDA_PROMPT_MODIFIER=(base)
TIMEZONE=Asia/Shanghai
IMAGE_NAME=build-tools
GOROOT=/usr/local/go
RUN=./tools/hack/run.sh
MAKE_DOCKER=./tools/hack/run.sh make --no-print-directory -e -f Makefile.core.mk
MFLAGS=
LESSCLOSE=/usr/bin/lesspipe %s %s
XDG_SESSION_CLASS=user
TERM=xterm
IN_BUILD_CONTAINER=1
LESSOPEN=| /usr/bin/lesspipe %s
CONDA_SHLVL=1
GOPROXY=https://proxy.golang.org
SHLVL=2
LOCAL_ARCH=x86_64
MAKELEVEL=1
XDG_SESSION_ID=29888
GOCACHE=/gocache
KUBECONFIG=
CONDA_PYTHON_EXE=/root/anaconda3/bin/python
XDG_RUNTIME_DIR=/run/user/0
GO111MODULE=on
PS1=build-tools:\w\$ \[\]
CONDA_DEFAULT_ENV=base
TARGET_ARCH=amd64
XDG_DATA_DIRS=/usr/local/share:/usr/share:/var/lib/snapd/desktop
PATH=/usr/local/go/bin:/gobin:/usr/local/google-cloud-sdk/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/0/bus
GENERATE_API=1
IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
TARGET_OUT=/work/out/linux_amd64
CONTAINER_TARGET_OUT=/work/out/linux_amd64
IMAGE_VERSION=release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
DEBUG=0
UID=0
RUBYOPT=-KU -E utf-8:utf-8
TARGET_OUT_LINUX=/work/out/linux_amd64
CONTAINER_TARGET_OUT_LINUX=/work/out/linux_amd64
GOPATH=/go
_=/usr/bin/env
build-tools:/work#

# 2. 生成 Make 自带的数据库
make DEBUG=1 -e -f Makefile.core.mk -pq > Makefile_core_database.txt 2>&1

build-tools:/work# make DEBUG=0 -e -f Makefile.core.mk -n --dry-run
make[1]: Entering directory '/work'
mkdir -p /work/out/linux_amd64
make[1]: Leaving directory '/work'
build-tools:/work#

make lint
make prebuild
make bootstrap
make release
GOPROXY="https://proxy.golang.org,direct" make go.test.coverage
GOPROXY="https://proxy.golang.org,direct" make higress-conformance-test
GOPROXY="https://proxy.golang.org,direct" make build
GOPROXY="https://proxy.golang.org,direct" make build-hgctl-multiarch
GOPROXY="https://proxy.golang.org,direct" make build-hgctl-macos-arm64
GOPROXY="https://proxy.golang.org,direct" make build-hgctl-macos-amd64
GOPROXY="https://proxy.golang.org,direct" IMG_URL="$image" make docker-buildx-push
GOPROXY="https://proxy.golang.org,direct" IMG_URL="$BUILT_IMAGE" make build-istio
GOPROXY="https://proxy.golang.org,direct" IMG_URL="$BUILT_IMAGE" make build-gateway
GOPROXY="https://proxy.golang.org,direct" PLUGIN_TYPE=${{ matrix.wasmPluginType }} make higress-wasmplugin-test

```

这份 Makefile 是 Istio 项目中非常典型且设计精妙的"包装器(Wrapper)"文件. 它的核心目的是屏蔽底层构建环境的差异. 它通过判断一个环境变量, 来决定是在本地宿主机直接执行构建, 还是将构建指令无缝投递到 Docker 容器中执行, 以此保证跨平台构建的一致性.

以下为您对该 Makefile 进行的详细拆解、注释以及深度原理解析.

---

### 一、带详细注释的 Makefile 代码

```makefile
# WARNING: DO NOT EDIT, THIS FILE IS PROBABLY A COPY
# ... (版权与许可声明, 此处略)

# ==============================================================================
# 1. 基础全局配置
# ==============================================================================

# [语法]使用 ':=' 进行立即赋值, 强制 Make 使用 /bin/bash 作为配方(Recipe)的执行 Shell.
# [目的]默认的 Make Shell 通常是 /bin/sh(在许多系统上指向 dash), 显式指定 bash 可以确保我们在 Makefile 内部或外部脚本中使用 Bash 特有语法(如数组、Pipefail 等)时表现一致.
SHELL := /bin/bash

# [语法]'-include' 尝试引入另一个 Makefile. 前缀 '-' 表示如果该文件不存在, Make 会静默忽略, 不会报错终止.
# [目的]允许开发者在本地创建 'Makefile.overrides.mk' 来覆盖或自定义某些构建变量, 而无需修改公共代码.
-include Makefile.overrides.mk

# [语法]'export' 使得变量能传递给子进程. '?=' 是条件赋值运算符, 只有当 BUILD_WITH_CONTAINER 未定义时才赋值为 0.
# [目的]默认在本地宿主机构建(0). 如果开发者在终端执行 `export BUILD_WITH_CONTAINER=1`, 则切换到容器化构建.
export BUILD_WITH_CONTAINER ?= 0


# ==============================================================================
# 2. 分支一: 容器化构建模式 (BUILD_WITH_CONTAINER == 1)
# ==============================================================================
ifeq ($(BUILD_WITH_CONTAINER),1)

# [语法]不带任何参数的单字 'export'.
# [原理]在 GNU Make 中, 单独的 export 会将当前 Makefile 中定义的所有变量都导出到子 Shell 的环境变量中.
# [目的]确保在 Makefile.overrides.mk 或本文件中定义的所有变量, 都能无缝传递给后续在 Docker 容器中运行的构建脚本.
export

# [语法]'$(shell ...)' 会在 Make 解析阶段立即调用系统 Shell 执行命令.
# [原理]双重嵌套的 '$(shell pwd)' 用于获取当前绝对路径, 防范相对路径带来的定位偏差.
# [目的]在解析阶段立即执行环境初始化脚本, 为容器启动准备必要的环境配置文件.
$(shell $(shell pwd)/tools/hack/setup_env.sh)

# [变量定义]定义容器启动器的路径.
RUN = ./tools/hack/run.sh

# [变量定义]定义在 Docker 容器中运行 Make 的命令.
#  - '--no-print-directory': 阻止 Make 输出 "Entering/Leaving directory" 干扰日志.
#  - '-e': 强制让环境变量覆盖 Makefile 内部定义的同名变量(非常关键, 保证外部传入配置生效).
#  - '-f Makefile.core.mk': 容器内实际执行的是核心逻辑文件 Makefile.core.mk.
MAKE_DOCKER = $(RUN) make --no-print-directory -e -f Makefile.core.mk

# [语法]'%' 是通配符目标(Pattern Rule), 匹配任何命令行传入的 Target(例如 make build, make test).
# [原理]'$@' 是自动变量, 代表当前匹配到的目标名称.
# [目的]把用户在宿主机输入的任何 target, 原封不动地打包投递到容器内重新运行.
%:
	@$(MAKE_DOCKER) $@

# [语法]默认目标(当用户直接输入 `make` 时的默认行为).
default:
	@$(MAKE_DOCKER)

# [目的]提供一个快捷入口, 允许开发者直接进入用于构建的容器交互式命令行(Bash).
shell:
	@$(RUN) /bin/bash

# [语法]声明伪目标, 防止当前目录下如果存在名为 'default' 或 'shell' 的文件时导致目标失效.
.PHONY: default shell


# ==============================================================================
# 3. 分支二: 本地构建模式 (BUILD_WITH_CONTAINER == 0)
# ==============================================================================
else

# [语法/原理]这两行在 Make 的"第一阶段"(解析阶段)就会被立即执行.
#  1. 先创建输出目录 'out'.
#  2. 运行 setup_env.sh 并传入 'envfile' 参数, 将其输出(格式通常为 VAR=VAL)重定向写入到 out/.env 文件.
# [目的]动态生成当前本地开发环境所需的环境变量文件.
$(shell mkdir -p out)
$(shell $(shell pwd)/tools/hack/setup_env.sh envfile > out/.env)

# [语法]'include' 关键字将刚才动态生成的环境变量文件载入到当前的 Make 进程中.
# [目的]巧妙地利用文件读入, 解决了 Make 自身难以在运行时动态感知和解析 Shell 脚本中 export 变量的痛点.
include out/.env

# [语法]再次调用无参数的 'export', 将从 out/.env 中加载进来的所有变量导出到后续所有子进程中.
export

# [语法]条件赋值并导出 GOBIN. 如果外部没定义 GOBIN, 则默认指向 Go 工作区的 bin 目录.
export GOBIN ?= $(GOPATH)/bin

# [语法]引入核心构建逻辑文件.
# [目的]既然是在本地构建, 直接把核心逻辑引入进来. 此时用户执行的 `make <target>` 将直接由本地环境的 Make 解析并执行.
include Makefile.core.mk

endif
```

---

### 二、核心语法与技术细节解析

1. `-include` vs `include`

   * 语法区别: `include` 如果找不到文件会直接报 Fatal 错误并中断构建; 而 `-include` 如果找不到文件会静默跳过.

   * 设计目的: `Makefile.overrides.mk` 是供开发者本地定制使用的(可能根本不存在), 所以用 `-include`; 而 `out/.env` 是流程中动态生成的、必须存在的依赖, 所以用 `include`.

2. 为什么写两个 `export`(不带参数)?

   * 在 GNU Make 中, `export` 不带参数是一种全局声明. 它告诉 Make: "在这个 Makefile 内部解析出来的所有变量, 在调用外部 shell 命令或子 Make 时, 全部放到环境变量里传递过去."

   * 设计目的: 在分支一中, 这确保了所有变量能够穿透到 `run.sh`(Docker 启动器)中; 在分支二中, 这确保了从 `out/.env` 读取的本地变量能被核心逻辑中调用的 Go 编译器、Bash 脚本等正确读取.

3. 双重嵌套的 `$(shell $(shell pwd)/...)`

   * 语法解析: 内层的 `$(shell pwd)` 先获取当前绝对路径, 拼接成绝对路径脚本后, 外层的 `$(shell ...)` 负责执行该脚本.

   * 设计目的: 避免因 Make 在不同的子目录、或用户通过 `make -C` 启动时, 相对路径失效导致找不到初始化脚本.

---

### 三、架构设计与"为什么要这么写"

这个 Makefile 展现了大型开源项目(如 Istio、Kubernetes 生态)在工程化构建上的典型设计模式:

#### 1. 痛点: 环境不一致

* 问题: 每个开发者的本地电脑操作系统(macOS, Linux, WSL)、Go 版本、Linter 工具版本、Protobuf 编译器版本都不同. 直接本地编译经常出现"在我的电脑上是好的, CI 上挂了"的问题.

* 解决办法: 引入容器化构建. 将所有工具链打包进一个 Docker 镜像中.

* 将所有工具链打包进一个 Docker 镜像中的 Dockerfile 文件是 https://github.com/istio/tools/blob/master/docker/build-tools/Dockerfile

#### 2. 设计模式: 面向用户的 Wrapper(包装器)

这个 Makefile 其实是个门面(Facade). 它不包含任何具体的编译、测试命令, 具体的编译逻辑全部写在 `Makefile.core.mk` 中. 它的唯一职责是环境分流:

```
                   [ 终端执行: make build ]
                             │
               ┌─────────────┴─────────────┐
    BUILD_WITH_CONTAINER=1        BUILD_WITH_CONTAINER=0
               │                           │
  ┌────────────┴────────────┐     ┌────────┴────────────┐
  │ 启动 ./tools/hack/run.sh │     │ 1. 动态生成本地 .env  │
  │ 将指令投递进 Docker 容器  │     │ 2. include 载入变量  │
  │ 在容器内运行 Makefile.core│     │ 3. 直接运行 Makefile.core│
  └─────────────────────────┘     └─────────────────────┘
```

#### 3. 巧妙的"自举(Bootstrap)"环境同步

在本地模式(`BUILD_WITH_CONTAINER=0`)下, 有一行非常高明的设计:

```makefile
$(shell $(shell pwd)/tools/hack/setup_env.sh envfile > out/.env)
include out/.env
```

* 为什么要大费周折写文件再 include?
  Make 本身是一个声明式的构建工具, 它无法在运行途中直接捕获一个 Shell 脚本内部 `export` 出来的变量(因为 Shell 脚本是在子进程中运行的, 子进程无法改变父进程的环境变量).

* 破解之法: 通过 `$(shell ...)` 让脚本把变量打印成 `KEY=VALUE` 的格式并写入 `out/.env` 文件. 随后, 利用 Make 的 `include` 机制, 在 Make 还没开始执行具体 Target 之前, 把这个文件像代码一样合进来. 这就完美实现了动态环境同步.

---------------------------------------------------------------------------------------------------------

Q: 假设你是一位经过 linux make 和 Makefile 的高级开发人员, 我现在有如下 Makefile 文件
```
# WARNING: DO NOT EDIT, THIS FILE IS PROBABLY A COPY
#
# The original version of this file is located in the https://github.com/istio/common-files repo.
# If you're looking at this file in a different repo and want to make a change, please go to the
# common-files repo, make the change there and check it in. Then come back to this repo and run
# "make update-common".

# Copyright Istio Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SHELL := /bin/bash

# allow optional per-repo overrides
-include Makefile.overrides.mk

# Set the environment variable BUILD_WITH_CONTAINER to use a container
# to build the repo. The only dependencies in this mode are to have make and
# docker. If you'd rather build with a local tool chain instead, you'll need to
# figure out all the tools you need in your environment to make that work.
export BUILD_WITH_CONTAINER ?= 0

ifeq ($(BUILD_WITH_CONTAINER),1)

# An export free of arguments in a Makefile places all variables in the Makefile into the
# environment. This is needed to allow overrides from Makefile.overrides.mk.
export

$(shell env)

$(shell $(shell pwd)/tools/hack/setup_env.sh)

RUN = ./tools/hack/run.sh

MAKE_DOCKER = $(RUN) make --no-print-directory -e -f Makefile.core.mk

print:
	@echo "SHELL: $(SHELL)"
	@echo "BUILD_WITH_CONTAINER: $(BUILD_WITH_CONTAINER)"
	@echo "CONTAINER_OPTIONS: $(CONTAINER_OPTIONS)"
	@echo "GENERATE_API: $(GENERATE_API)"
	@echo "IMAGE_VERSION: $(IMAGE_VERSION)"
	@echo "PHONYS: $(PHONYS)"
	@echo "RUN: $(RUN)"
	@echo "MAKE_DOCKER: $(MAKE_DOCKER)"

%:
	@$(MAKE_DOCKER) $@

default:
	@$(MAKE_DOCKER)

shell:
	@$(RUN) /bin/bash

.PHONY: default shell

else

# If we are not in build container, we need a workaround to get environment properly set
# Write to file, then include
$(shell mkdir -p out)
$(shell $(shell pwd)/tools/hack/setup_env.sh envfile > out/.env)
include out/.env
# An export free of arguments in a Makefile places all variables in the Makefile into the
# environment. This behavior may be surprising to many that use shell often, which simply
# displays the existing environment
export

export GOBIN ?= $(GOPATH)/bin
include Makefile.core.mk

endif

```
其中 Makefile.overrides.mk 文件内容如下
```
# Copyright 2019 Istio Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.DEFAULT_GOAL := default

# This repository has been enabled for BUILD_WITH_CONTAINER=1. Some
# test cases fail within Docker, and Mac + Docker isn't quite perfect.
# For more information see: https://github.com/istio/istio/pull/19322/

BUILD_WITH_CONTAINER ?= 0
CONTAINER_OPTIONS = --mount type=bind,source=/tmp,destination=/tmp --net=host

GENERATE_API ?= 0

ifeq ($(GENERATE_API),1)
BUILD_WITH_CONTAINER = 1
IMAGE_VERSION=release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613
endif

ifeq ($(BUILD_WITH_CONTAINER),1)
# create phony targets for the top-level items in the repo
PHONYS := $(shell ls | grep -v Makefile)
.PHONY: $(PHONYS)
$(PHONYS):
	@$(MAKE_DOCKER) $@
endif

```

我使用命令 `export BUILD_WITH_CONTAINER=1 && export GENERATE_API=1 && export GOPROXY="https://proxy.golang.org,direct" && make print` 调试时, 发现 `$(shell env)` 并没有打印出环境变量, 请问这是什么原因, 应该如何打印所有的环境变量或者 makefile 中的变量

在使用 GNU Make 时, 你遇到的 `$(shell env)` 没有在终端打印任何内容的问题, 是由 `$(shell ...)` 函数的运行机制决定的.

### 原因分析

在 Makefile 中, `$(shell ...)` 是一个内置函数, 它的行为是:
1. 在后台启动一个子 shell 并执行指定的命令(这里是 `env`).
2. 捕获该命令的标准输出(stdout), 并将其作为字符串返回给 Makefile.
3. 它不会直接将捕获到的输出投递到终端的屏幕上.

当你在 Makefile 中单独写一行 `$(shell env)` 时, Make 实际上执行了 `env`, 拿到了所有的环境变量字符串, 然后把这些字符串替换到了 `$(shell env)` 所在的位置.
* 因为这些输出没有被任何打印函数包裹, 所以终端不会有任何输出.
* 甚至, 如果 `env` 输出的内容不符合 Makefile 的语法(例如某些复杂的环境变量包含空格或特殊字符), 还可能会导致 Make 解析时报错(不过在很多情况下, 它可能只是被 Make 隐式忽略或当作了临时的变量定义).

---

### 解决方案

如果你想打印所有的环境变量或者 Makefile 中的变量, 可以采用以下几种专业且高效的方法:

#### 方法一: 在解析阶段打印(使用 `$(info ...)`)
如果你希望在 Make 读取和解析 Makefile 的阶段(即执行具体 target 之前)就看到环境变量, 可以使用 Make 的内置打印函数 `$(info ...)`、`$(warning ...)` 或 `$(error ...)`. 这些函数会把内容直接输出到终端.

请将 `$(shell env)` 修改为:
```makefile
$(info $(shell env))
```
注: 由于使用了 `export`(无参数版), 这会把 Makefile 里的变量也导出到子 shell 的环境中, 因此通过此方法你能看到非常详尽的环境变量列表.

#### 方法二: 在执行阶段打印(在 `print` 目标中调用 `env`)
如果你希望在运行 `make print` 时才打印环境变量, 应该把 `env` 命令放到 `print` 目标的 Recipe(配方)中执行:

```makefile
print:
	@echo "=== Shell Environment Variables ==="
	@env
	@echo "=== Selected Makefile Variables ==="
	@echo "SHELL: $(SHELL)"
	@echo "BUILD_WITH_CONTAINER: $(BUILD_WITH_CONTAINER)"
	# ... 其他原有的 echo 语句
```
注: Recipe 中的命令是在 Shell 中运行的, 其标准输出会直接显示在终端.

#### 方法三: 打印 Makefile 内部的所有变量
如果你不仅想看系统环境变量, 还想看 Makefile 内部定义的所有变量(包括自动变量、隐式变量等), 可以使用 GNU Make 提供的内置变量 `.VARIABLES`.

你可以通过 `foreach` 循环配合 `info` 来打印它们:

```makefile
# 在 Makefile 的合适位置(例如 ifeq 块内部, 或者 print 目标中)
$(foreach v,$(.VARIABLES),$(info $(v) = $($(v))))
```
> 提示: `.VARIABLES` 包含的内容非常庞大(包括 Make 自带的大量隐式变量如 `CC`、`AR` 等). 如果只想看用户自定义的, 可以使用 `make -p` 进行过滤.

#### 方法四: 使用 Make 自带的数据库打印命令(推荐调试时使用)
如果你不想修改 Makefile 文件, GNU Make 提供了一个非常强大的命令行调试参数 `-p`(或 `--print-data-base`), 它可以输出 Make 解析后的所有变量和规则.

你可以直接在终端运行以下命令, 并通过 `grep` 过滤你关心的变量:
```bash
make -p | grep -A 1 "BUILD_WITH_CONTAINER ="
```
或者将完整的变量和规则数据库输出到文件中查看:
```bash
make -p > make_database.txt
```

---------------------------------------------------------------------------------------------------------

Q: 如何检查 Makefile 中定义了哪些目标 target, 有相应的变量可以看到所有可用的 target 吗

在 GNU Make 中, 并没有类似于 `.VARIABLES` 这样能直接获取所有可用 Target(目标)列表的内置变量(尽管社区曾讨论过引入类似 `.TARGETS` 的变量, 但至今并未实现).

在实际工程中, 解决这个需求通常有以下几种成熟、实用的替代方案:

---

### 方法一: 利用 `make -qp` 命令行过滤(最精准, 支持动态生成的目标)

如果你的 Makefile 包含了很多由 `$(shell ...)` 或 `$(foreach ...)` 动态生成的 Target, 直接用文本搜索 `Makefile` 文件是查不全的.

最准确的方法是让 Make 打印它已经解析好的内部数据库, 然后用脚本过滤. 这也是 Linux 系统中 `make` 命令的 Bash 自动补全(Tab 键补全)背后所使用的原理.

你可以在终端直接运行以下单行命令(无需修改 Makefile):

```bash
make -qp | awk -F':' '/^[a-zA-Z0-9][^$#\/\t=]*:([^=]|$)/ {split($1,A,/ /);for(i in A)print A[i]}' | sort -u
```

参数解释:
* `-p`(`--print-data-base`): 打印 Make 的内部变量和规则数据库.
* `-q`(`--question`): 只提问, 不真正运行任何 Recipe 命令.
* `awk` 部分: 过滤掉 Makefile 内部自带的隐式规则(如 `.c.o:`)以及变量赋值行, 只留下有效的自定义目标.

---

### 方法二: 在 Makefile 中实现自文档化的 `help` 目标(现代工程最推荐)

在日常开发中, 我们通常不需要看那些 Makefile 内部使用的辅助目标, 而只需要看可供开发者调用的公共目标.

目前在 Istio、Kubernetes 以及大量 Go 语言开源项目中, 最流行的做法是为需要公开的目标添加 `##` 注释, 然后编写一个 `help` 目标来解析并漂亮地排版输出.

你可以把以下这段代码加到你的 Makefile 中:

```makefile
.DEFAULT_GOAL := help

.PHONY: help
help: ## 展示帮助信息(列出所有可用目标)
	@echo "可用目标 (Targets):"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

default: ## 默认构建目标
	@echo "building default..."

shell: ## 进入容器内 bash 环境
	@echo "starting shell..."
```

效果:
当你在终端输入 `make` 或 `make help` 时, 它会抓取所有带有 `##` 的行, 并自动排版输出:
```text
可用目标 (Targets):
  help                 展示帮助信息(列出所有可用目标)
  default              默认构建目标
  shell                进入容器内 bash 环境
```

---

### 方法三: 在 Makefile 内部维护 Target 变量(最简单直接)

如果你在编写 Makefile 时, 确实需要在某些逻辑里(比如循环、条件判断)知道有哪些目标, 最稳妥、不易出错的方法依然是手动用一个变量来集中声明和管理它们.

例如, 在你要调试的 Istio 相关的 Makefile 中, 其实已经采用了这种思路:

```makefile
# 定义一组主要的目标
TARGETS := default shell print

.PHONY: $(TARGETS)

$(TARGETS):
	@echo "Executing target: $@"
```

通过显式声明 `TARGETS` 变量, 后续你就可以在 Makefile 内部直接使用 `$(TARGETS)` 变量来代表所有的目标列表, 而不需要依赖 Make 的运行时提取.

---------------------------------------------------------------------------------------------------------

Q: 假设你是一位精通 docker 的高级开发人员, 现在有如下 docker 启动容器的命令
```
docker run \
--rm \
--init \
--sig-proxy=true \
-v /var/run/docker.sock:/var/run/docker.sock \
--mount type=bind,source=/tmp,destination=/tmp \
--net=host \
--env-file /dev/fd/63 \
-e IN_BUILD_CONTAINER=1 \
-e TZ=Asia/Shanghai \
-e HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress \
--mount type=bind,source=/root/huzhi/higress,destination=/work \
--mount type=volume,source=go,destination=/go \
--mount type=volume,source=gocache,destination=/gocache \
--mount type=volume,source=cache,destination=/home/.cache \
--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly \
-w /work \
higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613 \
make --no-print-directory -e -f Makefile.core.mk build
```
请详细说明上述命令中每个选项的含义, 详细说明这些选项的作用, 说明为什么需要这些选项

这行复杂的 `docker run` 命令是典型的容器化构建环境(Build-with-Container)的最佳实践. 它的核心设计理念是: "环境即代码, 开发免配置". 通过启动一个包含所有编译依赖(Go、C++ 编译器、Protobuf 工具链等)的专用构建镜像(`build-tools`), 来在隔离干净的环境中编译 Higress 代码, 同时保留了本地开发的高性能和网络便利性.

下面, 我将从生命周期管理、宿主机通信与 DooD、网络与环境配置、源码挂载与缓存持久化(核心加速)四个维度, 为你逐一拆解每个选项的含义、作用以及引入它的深层原因.

---

### 一、容器生命周期与进程管理

#### 1. `--rm`
* 含义: 容器退出时, 自动清理和删除容器的文件系统.
* 作用: 防止磁盘空间被大量死掉的容器堆积占满.
* 为什么需要: 构建容器属于临时性任务(Ephemeral Task). 我们启动它的唯一目的就是跑完 `make build` 生成二进制文件, 跑完后容器就失去了价值. 如果不加 `--rm`, 每次运行构建都会留下一个处于 `Exited` 状态的死容器, 长期以往会迅速耗尽宿主机的磁盘 inode 和空间.

#### 2. `--init`
* 含义: 在容器内启用一个微型的初始化守护进程(通常是 `tini`)作为 PID 1(1号进程).
* 作用: 负责信号转发和孤儿/僵尸进程的回收.
* 为什么需要: 在构建过程中, `make` 可能会拉起大量的子进程(例如并行编译、执行 shell 脚本). 如果用户突然按下 `Ctrl+C` 想终止编译, 若没有 `--init` 充当 1 号进程, 信号可能无法正确传递给所有的子编译进程, 导致容器无法退出, 或者在容器内残留大量无法回收的"僵尸进程". `--init` 保证了容器内进程生命周期的健壮性.

#### 3. `--sig-proxy=true`
* 含义: 将宿主机(即当前终端)接收到的所有系统信号(如 `SIGINT` / `SIGTERM` / `SIGQUIT`), 100% 代理并转发给容器内部的进程.
* 作用: 确保外部对 `docker run` 的控制(例如 CI/CD 系统取消任务、或者人手动中断)能够即时且安全地传达给容器内的 `make` 进程.
* 为什么需要: 在非交互式或复杂的脚本调用上下文中, 为了防止容器在后台"失控"持续消耗 CPU 编译, 必须确保外界的信号能够强力、直接地制约容器内的实际执行主体.

---

### 二、宿主机通信与 Docker-out-of-Docker (DooD)

#### 4. `-v /var/run/docker.sock:/var/run/docker.sock`
* 含义: 将宿主机的 Docker 守护进程套接字(Socket)挂载到容器内部的相同位置.
* 作用: 允许容器内部的程序直接控制宿主机上的 Docker 引擎(即 DooD, Docker-out-of-Docker 模式).
* 为什么需要: Higress 的构建流程中, 有一步是需要将编译好的二进制打包成 Docker 镜像(例如调用 `docker build`). 因为我们目前已经是在容器里了, 如果要在容器里"套娃"装一个完整的 Docker 引擎(DinD)会非常重且存在安全隐患. 通过挂载 `docker.sock`, 容器内的 `make` 脚本可以直接调用宿主机的 Docker 引擎来打包、标记和推送镜像, 效率最高、也最轻量.

#### 5. `--mount type=bind,source=/tmp,destination=/tmp`
* 含义: 将宿主机的 `/tmp` 目录绑定挂载到容器内的 `/tmp`.
* 作用: 使宿主机和容器共享临时文件夹.
* 为什么需要: 当我们在容器内部通过 `docker.sock` 指挥宿主机 Docker 构建镜像或运行临时测试容器时, 宿主机 Docker 需要能够访问到容器内传递过来的某些临时文件(例如临时证书、编译中间产物). 如果两边的 `/tmp` 路径和内容不一致, 宿主机 Docker 就会找不到文件. 共享 `/tmp` 保证了这种交互的上下文一致.

---

### 三、网络与环境配置

#### 6. `--net=host`
* 含义: 使容器与宿主机共享同一个网络命名空间(Network Namespace).
* 作用: 容器内不需要虚拟网卡和网桥, 直接使用宿主机的 IP、DNS 以及路由表.
* 为什么需要:
  * 网络性能: 避免了 Docker 桥接网卡(bridge)带来的网络地址转换(NAT)开销, 加快依赖下载.
  * 网络连通性: 构建期间需要下载大量的 Go 依赖包、第三方库(可能会走宿主机配置好的本地 VPN/代理、或者是本地搭建的私有镜像源). 使用 `--net=host` 可以直接继承宿主机的代理配置(如 `HTTP_PROXY`)、DNS 设置(如 `/etc/resolv.conf`), 避免了繁琐的容器内网络调试.

#### 7. `--env-file /dev/fd/63`
* 含义: 从文件描述符 `/dev/fd/63`(在 Bash 中通常是由进程替换 ` <(env)` 产生的一个虚拟文件)中读取环境变量.
* 作用: 动态地将宿主机当前 Shell 环境中的大部分环境变量成批导入到容器内, 而不需要在命令行中一条一条地写 `-e`.
* 为什么需要: 这是一种非常优雅且安全的 Shell 高级技巧. 由于我们运行的是 `make print` / `make build`, 宿主机中可能有各种定制的 Go 代理(`GOPROXY`)、私钥或私有凭证. 通过这个虚拟文件描述符, Docker 能直接安全地读取当前最新的环境变量, 同时避免了将敏感密钥暴露在 `docker ps` 的命令行历史中.

#### 8. `-e IN_BUILD_CONTAINER=1`
* 含义: 向容器内传递名为 `IN_BUILD_CONTAINER`、值为 `1` 的环境变量.
* 作用: 这是一个标志位, 告诉容器内部的 `Makefile` 或脚本: "你现在已经处于构建容器内了".
* 为什么需要: 防止递归死循环. 如果 Makefile 检测到 `BUILD_WITH_CONTAINER=1` 却不知道自己已经在容器里, 它可能会在容器内再次执行 `docker run` 试图启动另一个容器. 这个变量能让脚本识别环境, 直接走具体的本地编译逻辑.

#### 9. `-e TZ=Asia/Shanghai`
* 含义: 设置容器的时区为上海时间(东八区).
* 作用: 校准容器内部的系统时间.
* 为什么需要: 默认的 Docker 镜像通常采用 UTC(格林威治标准时间). 在编译和构建过程中, 日志输出的时间戳、编译打包时注入到二进制文件中的 Build Time 等, 都需要符合本地开发者的时区习惯, 以便于排查问题.

#### 10. `-e HUB=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress`
* 含义: 设置镜像仓库的前缀变量为阿里云的 Higress 镜像仓库.
* 作用: 传递给内部构建脚本, 指导后续打包生成的 Docker 镜像该推送到哪个 Hub 地址.
* 为什么需要: 使得镜像打包脚本具有参数化配置的能力.

---

### 四、源码挂载与缓存持久化(核心加速)

构建容器默认是无状态的, 这意味着每次启动都是全新的环境. 如果不做任何挂载, 每次构建都要重新下载几个 G 的 Go 依赖和重新编译标准库, 耗时极长. 以下挂载设计巧妙地解决了这一痛点.

#### 11. `--mount type=bind,source=/root/huzhi/higress,destination=/work`
* 含义: 将宿主机上实际存放 Higress 源码的目录 `/root/huzhi/higress` 绑定挂载到容器内的 `/work`.
* 作用: 这是构建的核心. 容器内的编译器直接对宿主机的代码进行读写.
* 为什么需要: 因为代码在宿主机, 而编译器在容器里. 挂载后, 容器内对源码的修改、以及生成的 `out/` 目录和最终的编译产物, 都会直接实时保留在宿主机的硬盘上, 容器销毁后这些产物依然存在.

#### 12. `--mount type=volume,source=go,destination=/go`
* 含义: 将 Docker 的具名卷(Named Volume)`go` 挂载到容器内的 `/go` 目录(即 Go 的 `GOPATH`).
* 作用: 持久化缓存下载的 Go Module 依赖包.
* 为什么需要: Go 项目(如 Higress)在编译前需要拉取大量的依赖. 通过使用 Docker 具名卷挂载 `/go`, 所有下载过的 `pkg/mod` 依赖包都会被保存在宿主机的高性能本地卷中. 下一次你再启动编译容器时, Go 编译器能直接复用这些缓存, 避免了每次都重新从公网下载依赖.

#### 13. `--mount type=volume,source=gocache,destination=/gocache`
* 含义: 将具名卷 `gocache` 挂载到容器内的 `/gocache`(Go 编译缓存目录).
* 作用: 持久化保存 Go 编译产生的中间临时 `.a` 静态库文件.
* 为什么需要: Go 语言拥有极其强大的增量编译(Incremental Build)缓存机制(由环境变量 `GOCACHE` 指定). 如果不持久化这个目录, 即使你只改动了一行代码, Go 也必须重新编译项目中的所有文件. 持久化此目录后, 后续构建速度会从"分钟级"骤减到"秒级".

#### 14. `--mount type=volume,source=cache,destination=/home/.cache`
* 含义: 将具名卷 `cache` 挂载到容器内的用户缓存目录 `/home/.cache`.
* 作用: 持久化其他构建工具链的缓存.
* 为什么需要: 除了 Go 本身, Higress 的编译链中可能还会用到如 `golangci-lint`(代码风格检查)、`protoc`(Protobuf 编译器插件缓存)或其他工具. 这些工具的缓存通常会存放在当前用户的 `.cache` 下, 持久化它能防止其他辅助工具在每次构建时重新初始化或下载插件.

#### 15. `--mount type=bind,source=/root/.docker,destination=/config/.docker,readonly`
* 含义: 将宿主机当前用户的 Docker 客户端凭证目录 `/root/.docker` 以只读(readonly)方式绑定挂载到容器内的 `/config/.docker`.
* 作用: 将宿主机上已经登录过(`docker login`)的镜像仓库凭证(Credentials)共享给容器内部.
* 为什么需要: 如果构建脚本在最后需要把打包好的镜像推送到私有仓库(阿里云 CR), 容器内的 Docker 客户端必须有推送权限. 直接共享宿主机的 `.docker` 目录, 使得容器内的 Docker 自动继承了宿主机已经登录好的状态, 无需在容器里重复输入账号密码. `readonly` 限制则保证了容器内部无法篡改或损坏宿主机珍贵的本地 Docker 配置文件.

---

### 五、工作目录与启动指令

#### 16. `-w /work`
* 含义: 设定容器启动后的初始工作目录(Working Directory)为 `/work`.
* 作用: 相当于容器一启动, 就在内部自动执行了 `cd /work`.
* 为什么需要: 因为我们的源码已经被挂载在 `/work` 目录下. 我们将工作目录设在这里, 以便紧接着执行的 `make` 命令能够直接在项目的根目录下寻找 `Makefile.core.mk`.

#### 17. 镜像名与后置命令
* 镜像名: `higress-registry.../build-tools:release-1.19...`
  * 这是经过精心预装、包含特定版本构建工具包的定制镜像.
* 运行命令: `make --no-print-directory -e -f Makefile.core.mk build`
  * 最终在容器中执行的编译指令.
  * `--no-print-directory` 减少不必要的日志噪音.
  * `-e`(`--environment-overrides`)指示 `make` 优先采用系统环境变量, 来覆盖 Makefile 内部可能存在的同名变量设定.
  * `-f Makefile.core.mk` 指定使用特定的 Makefile 核心配置文件.
  * `build` 触发编译目标.

---------------------------------------------------------------------------------------------------------

Q: 在上述容器化构建环境中, 使用了多个 --mount 参数, 并且每个 mount 的类型是不一样的, 请问 --mount 参数和直接使用 -v 参数有什么不一样, 为什么使用 --mount, 而不是用 -v, 并且当 type=volume, 是不是要提前将具名卷创建好, 不然是不是会出错

在容器化构建及 CI/CD 流程中, 挂载参数的选择直接影响到脚本的健壮性.

以下将为你详细剖析 `--mount` 与 `-v`(即 `--volume`)的区别、为什么在此场景中推荐 `--mount`, 以及关于具名卷(Named Volume)的自动创建机制.

---

### 一、`--mount` 与 `-v` 的核心区别

虽然它们都能实现文件挂载, 但在语法设计、报错机制以及使用场景上有着本质的不同:

| 特性                            | `-v` / `--volume`                                                        | `--mount`                                                                          |
| :------------------------------ | :----------------------------------------------------------------------- | :--------------------------------------------------------------------------------- |
| 引入版本                        | Docker 早期即存在                                                        | Docker 17.06 引入                                                                  |
| 语法格式                        | 单个字符串, 以冒号 `:` 分割: <br> `[宿主机路径]:[容器内路径]:[读写属性]` | 键值对(Key-Value)格式, 以逗号 `,` 分割: <br> `type=...,source=...,destination=...` |
| 类型区分                        | 隐式声明(通过参数特征自动猜测是 Bind 还是 Volume)                        | 显式声明(必须通过 `type=bind` 或 `type=volume` 明确指定)                           |
| 宿主机路径不存在时的行为 (Bind) | 会自动在宿主机上创建该目录(默认归属 `root` 用户)                         | 直接抛出错误并拒绝启动容器                                                         |
| Swarm 服务支持                  | 不支持在 `docker service` 中使用                                         | 唯一支持的挂载参数                                                                 |

#### 关键区别演示(针对 Bind 挂载):
如果你执行 `docker run -v /nonexistent:/data ...`, 而宿主机的 `/nonexistent` 路径实际上不存在:
* 使用 `-v`: Docker 会默默地在宿主机上帮你创建一个名为 `/nonexistent` 的文件夹(权限归 `root` 所有), 然后启动容器. 这极易导致权限混乱.
* 使用 `--mount`: Docker 会直接报错, 提示 `invalid mount config for type "bind": bind source path does not exist`.

---

### 二、为什么在构建环境中使用 `--mount` 而不是 `-v`?

在这个 Higress 构建命令中, 选用 `--mount` 主要是出于以下三点工程实践上的考量:

#### 1. 明确区分挂载类型(自文档化)
该构建命令混合使用了绑定挂载(Bind Mount)和具名卷(Named Volume):
* 源码、主机凭证等必须用 Bind 挂载(因为需要直接读写宿主机的具体物理文件).
* Go Module 缓存、编译器缓存等适合用 Volume 挂载(因为它们由 Docker 统一管理在 `/var/lib/docker/volumes` 下, 开发人员不需要直接访问).

使用 `--mount` 显式声明了 `type=bind` 和 `type=volume`, 任何开发人员阅读此脚本时, 都能一眼看懂哪些文件是和宿主机强绑定的, 哪些是由 Docker 托管的缓存.

#### 2. 避免"静默创建空目录"的隐患(Fail-Fast 机制)
源码挂载路径 `--mount type=bind,source=/root/huzhi/higress,destination=/work` 极其关键.
如果开发人员在宿主机上不小心删除了该源码目录, 或者拼错了路径:
* 如果用 `-v`, Docker 会在宿主机上悄悄新建一个空的 `/root/huzhi/higress` 目录, 然后容器内 `make` 报错"找不到 Makefile"或"代码为空".
* 用 `--mount`, Docker 启动容器的第一步就会报错拦截, 让错误在最早期暴露出来(Fail-Fast), 便于调试.

#### 3. 语法表现力更强, 不易出错
`-v` 的参数如果包含多个附加选项(如只读 `ro`、一致性 `cached` 等), 其冒号分隔的格式很容易写错或看错. `--mount` 采用结构化的 `key=value`, 可读性极佳, 例如 `,readonly` 的语义非常直观.

---

### 三、当 `type=volume` 时, 需要提前创建好具名卷吗?

不需要提前创建, 也不会出错.

当你执行:
```bash
--mount type=volume,source=go,destination=/go
```
即使宿主机上完全不存在名为 `go` 的 Docker 卷, Docker 也会自动在后台为你创建它, 然后启动容器. 这与 `-v go:/go` 的自动创建行为是一致的.

#### 为什么对 Volume 可以自动创建, 而 Bind 不行?
* Volume 是由 Docker 引擎完全托管的: Docker 在自身存储目录(如 `/var/lib/docker/volumes/`)下创建并管理一个卷是绝对安全的, 符合预期.
* Bind 触碰的是宿主机的文件系统: 如果允许 Docker 随意在宿主机上"无中生有"地创建目录, 不仅会破坏宿主机的目录结构, 还会带来潜在的安全隐患和权限问题(创建出来的目录默认属于 `root`). 因此, 对于 `type=bind`, Docker 坚持要求宿主机路径必须提前存在.

```bash
(base) root@k8s-a40-node02:~/huzhi/higress# docker volume ls
DRIVER    VOLUME NAME
local     cache
local     go
local     gocache
(base) root@k8s-a40-node02:~/huzhi/higress#
(base) root@k8s-a40-node02:~/huzhi/higress# docker volume inspect go
[
    {
        "CreatedAt": "2026-06-25T20:08:32+08:00",
        "Driver": "local",
        "Labels": null,
        "Mountpoint": "/var/lib/docker/volumes/go/_data",
        "Name": "go",
        "Options": null,
        "Scope": "local"
    }
]
(base) root@k8s-a40-node02:~/huzhi/higress# docker volume inspect gocache
[
    {
        "CreatedAt": "2026-06-25T20:08:32+08:00",
        "Driver": "local",
        "Labels": null,
        "Mountpoint": "/var/lib/docker/volumes/gocache/_data",
        "Name": "gocache",
        "Options": null,
        "Scope": "local"
    }
]
(base) root@k8s-a40-node02:~/huzhi/higress# docker volume inspect cache
[
    {
        "CreatedAt": "2026-06-25T20:08:32+08:00",
        "Driver": "local",
        "Labels": null,
        "Mountpoint": "/var/lib/docker/volumes/cache/_data",
        "Name": "cache",
        "Options": null,
        "Scope": "local"
    }
]
(base) root@k8s-a40-node02:~/huzhi/higress#
(base) root@k8s-a40-node02:~/huzhi/higress# docker volume rm cache gocache go
cache
gocache
go
(base) root@k8s-a40-node02:~/huzhi/higress#
```

---------------------------------------------------------------------------------------------------------

Q: 假设你是一位今天 linux, make, Makefile 的高级开发人员, 我调用命令 `make --no-print-directory -e -f Makefile.core.mk build` 时有如下报错
```
make --no-print-directory -e -f Makefile.core.mk build
++ env
++ grep -v '^_\|PATH\|SHELL\|EDITOR\|TMUX\|USER\|HOME\|PWD\|TERM\|GO\|rvm\|SSH\|TMPDIR\|CC\|CXX\|MAKEFILE_LIST'
make[1]: env: Operation not permitted
```
请问这是什么原因, 是因为我使用了 -e 参数吗? `--no-print-directory -e` 这两个参数的含义和作用是什么

这个报错与你是否使用了 `-e` 参数完全没有关系. `-e` 是 Make 工具内部的逻辑控制参数, 绝不会导致操作系统底层报出 `Operation not permitted`(操作不允许 / EPERM)这种系统调用级别的权限错误.

以下是关于该报错原因的深度剖析, 以及这两个参数的详细含义.

---

### 一、为什么会报 `env: Operation not permitted`?

这是一个典型的宿主机容器沙箱安全策略限制导致的错误.

当 Make 试图在容器内部执行 shell 命令(这里是 `env` 并通过 `grep` 过滤)时, 它会通过底层的 `fork` 和 `execve` 系统调用来拉起新进程. 如果该操作被宿主机的内核安全机制拦截, 就会向容器内返回 `EPERM` 错误, 表现为 `Operation not permitted`.

在 Docker/容器化构建环境中, 最常见的诱因有以下几种:

#### 1. 宿主机 `libseccomp` 库过旧(最常见原因)
新版的容器镜像(包括你正在使用的 `build-tools`)内部使用了较新版本的 GLIBC 或 C 标准库. 这些新库在执行命令时, 会调用一些较新的系统调用(例如 `clone3` 或 `faccessat2`).
如果你的宿主机操作系统较老(例如 Ubuntu 18.04/20.04), 其 `libseccomp`(安全计算模式过滤库)可能无法识别这些较新的系统调用, 从而将其作为"不安全操作"直接拦截, 导致任何命令(如 `env`, `bash`, `dirname`)在执行时都报 `Operation not permitted`.

如何验证和修复:
* 临时测试(不安全, 仅用于排查):
  在启动该容器的 `docker run` 命令中, 加入 `--security-opt seccomp=unconfined` 参数(即放开安全限制). 如果加入后 `make` 顺利跑通, 说明 100% 是宿主机 Seccomp 安全策略的问题.
* 根本解决(推荐):
  升级你宿主机上的 `libseccomp2` 库和 Docker 引擎. 例如在 Ubuntu 宿主机上运行:
  ```bash
  sudo apt-get update && sudo apt-get install -y libseccomp2 docker-ce
  ```

#### 2. macOS 上的 Docker Desktop 沙箱限制
如果你是在 macOS 上通过 Docker Desktop 运行此命令, 有时 macOS 自身的安全隐私限制(例如未授予终端"完全磁盘访问权限")或 Docker Desktop 虚拟机的 gRPC-FUSE 文件系统漏洞, 也会引发类似的 EPERM 挂载和执行限制.
* 尝试在 macOS 的"系统设置 -> 隐私与安全性 -> 完全磁盘访问权限"中, 勾选你使用的终端(如 Terminal、iTerm2)以及 Docker.

---

### 二、`--no-print-directory` 与 `-e` 参数的含义和作用

这两个参数是 GNU Make 非常经典且实用的控制开关:

#### 1. `--no-print-directory`
* 含义: 不打印目录切换信息.
* 作用:
  在大型项目中, Makefile 经常会嵌套调用其他子目录的 Makefile(例如主 Makefile 调用了 `Makefile.core.mk` 或进入子目录编译). 默认情况下, 每次进入和离开子目录, Make 都会往屏幕上打印如下日志:
  ```text
  make[1]: Entering directory '/work'
  ...
  make[1]: Leaving directory '/work'
  ```
  加上 `--no-print-directory` 后, Make 会静默这些目录切换提示, 从而让终端输出的日志保持极度的干净、紧凑, 避免非必要的日志噪音.

#### 2. `-e` (或 `--environment-overrides`)
* 含义: 环境变量覆盖(Environment Overrides).
* 作用:
  在默认情况下, Makefile 中显式定义的变量优先级是高于宿主机环境变量的. 例如:
  * 如果 Makefile 中写了: `HUB = local-registry`
  * 即使你在终端执行了: `export HUB=aliyun-registry`
  * 在不加 `-e` 的情况下运行 `make`, `HUB` 的值依然会被强制认定为 Makefile 里的 `local-registry`.

  一旦加上了 `-e` 参数, 规则就会颠倒: 宿主机传入的环境变量将拥有最高优先级, 直接覆盖(Override)Makefile 内部对同名变量的赋值.

* 为什么在这个构建场景中需要 `-e`:
  因为这是一个高度容器化的构建环境, 所有的定制参数(如 `HUB`、`TAG`、`GOPROXY` 等)都是通过宿主机(或者 CI/CD 平台)动态配置并以环境变量的形式(`-e HUB=...` 或 `--env-file`)注入到容器内部的. 加 `-e` 确保了容器内的 `make` 在执行时, 能够切实使用这些通过外部注入的最新参数, 而不是死板地采用 Makefile.core.mk 里的默认硬编码值.

---------------------------------------------------------------------------------------------------------

Q: 假设你是一位今天 linux, make, Makefile 的高级开发人员, 在我的 Makefile 文件中包含了其他的 Makefile, 如下所示
```
include tools/tools.mk
include tools/lint.mk
```
我在调试这个 Makefile 文件时, 使用命令 `make -f Makefile -pq`, 这时我可以看到详细的关于 Makefile 的内容, 其中包含 tools/tools.mk 相关的内容. 我如何单独调试 tools/tools.mk 的内容, 直接 `make -f tools/tools.mk -pq` 不行, 因为 tools/tools.mk 中使用了一些 Makefile 定义的变量

在大型、模块化的 Makefile 设计中(例如 Kubernetes、Istio 等项目), 子 Makefile(如 `tools.mk`)严重依赖主 Makefile 定义的全局变量和上下文, 这是一个非常经典且棘手的调试痛点.

直接运行 `make -f tools/tools.mk -pq` 会因为丢失上下文(未定义的变量、环境配置等)而导致解析错误或结果不准; 而运行 `make -f Makefile -pq` 产生的输出又过于庞大, 混杂了主 Makefile、`lint.mk` 以及大量内置的隐式规则.

作为高级开发人员, 我们可以通过以下几种精准、专业的调试方案来解决这个问题:

---

### 方案一: 利用 `awk` 段落模式精准过滤数据库(最推荐的静态调试法)

GNU Make 的 `-p`(`--print-data-base`)在输出变量和目标(Target)时, 会在每一项的注释中极其精准地标注其来源文件和行号.
例如:
* 变量定义: `# makefile (from 'tools/tools.mk', line 5)`
* 目标规则: `#  Recipe to be executed (from 'tools/tools.mk', line 12):`

利用这个特性, 我们可以配合 `awk` 的段落模式(通过将 `RS` 记录分隔符设为空, 使 `awk` 按空行来分割每一个完整的变量或目标块), 瞬间将 `tools/tools.mk` 相关的定义剥离出来.

在终端中执行以下命令(为了避免 Bash 里的单引号转义混乱, 这里使用了双引号包裹 `awk`):

```bash
make -f Makefile -pq | awk -v RS="" "/from 'tools\/tools.mk'/"
```

#### 原理解析:
1. `make -f Makefile -pq`: 让主 Makefile 带着完整的上下文去解析, 但并不执行任何实际命令.
2. `-v RS=""`: 这是 `awk` 的一个高级技巧(段落模式). 它告诉 `awk` 遇到连续的空行时才算作一条记录(Record). 因为 `make -p` 输出的每个变量和 Target 之间都由空行隔开, 所以它们会被完整地当作一个段落处理.
3. `"/from 'tools\/tools.mk'/"`: 筛选出所有包含 `from 'tools/tools.mk'` 的段落并打印.

通过这个命令, 你既保证了 `tools/tools.mk` 拿到了 `Makefile` 的所有变量上下文, 又在输出中只看得到定义在 `tools/tools.mk` 内部的变量与 Target, 极其干净利落.

---

### 方案二: 设计"条件引入"(企业级 Makefile 的最佳实践)

如果你不仅需要静态查看(`-pq`), 还需要频繁在开发中动态运行或单独调试 `tools.mk` 里的某些目标, 且不希望触发 `lint.mk` 等其他无关模块, 最优雅的架构设计是在主 `Makefile` 中引入调试开关(Toggle).

你可以微调主 `Makefile` 的引入逻辑:

```makefile
# Makefile
# 1. 这里定义核心变量和环境配置
export BUILD_WITH_CONTAINER ?= 0
# ...

# 2. 只有在未开启仅调试 tools 模式时, 才引入 lint.mk
ifndef TOOLS_ONLY
include tools/lint.mk
endif

# 3. 始终引入 tools.mk
include tools/tools.mk
```

#### 如何使用:
在调试时, 你可以通过在命令行传入 `TOOLS_ONLY=1`, 这样主 Makefile 就会带着所有的全局变量, 但排除掉 `lint.mk`(以及其他无关的 include 文件):

```bash
# 静态查看 tools.mk 解析后的数据库
make TOOLS_ONLY=1 -pq

# 实际运行 tools.mk 里的某个具体 target
make TOOLS_ONLY=1 <your-tools-target>
```

这种"条件式 Include"的设计模式在大型 CI/CD 流程中非常普遍, 能让原本臃肿的构建系统具备极高的局部调试柔韧性.

---

### 方案三: 使用临时包装器(Wrapper)进行沙盒调试

如果你不想修改项目已有的 `Makefile` 源码, 你可以临时在本地写一个极简的包装文件(例如命名为 `debug.mk`):

```makefile
# debug.mk
# 1. 引入主 Makefile 拿到所有基础变量
include Makefile

# 2. 如果有些 target 在主 Makefile 或 lint.mk 中冲突了, 可以在这里通过 undefine 掉
# 或者在这里覆盖(override)一些专门用于调试 tools 的变量

# 3. 接下来你就可以在这里专注于 tools.mk 的局部调试
```

调试命令:
```bash
make -f debug.mk <your-tools-target> -pq
```
这种方法将"上下文获取"与"特定模块调试"隔离开来, 不会污染生产环境的代码仓.

---------------------------------------------------------------------------------------------------------

Q: 假设你是一位今天 linux, make, Makefile 的高级开发人员, 现在我有如下 Makefile 文件
```

```
我使用命令 `make DEBUG=1 -e -f Makefile.core.mk -pq` 调试是输出 `go: downloading go1.24.4 (linux/amd64)` 请问我的这个 Makefile 哪里设置需要下载这个

这是一个非常经典且容易让人困惑的问题. 当你看到 `go: downloading go1.24.4 (linux/amd64)` 时, 说明系统已经在底层调用并运行了 `go` 相关的命令.

以下是导致这一现象的底层机制、具体触发位置以及排查方法的详细分析:

---

### 一、为什么使用了 `-q`(问询模式)依然触发了下载?

1. `-q` (或 `-n` / `--dry-run`) 仅作用于 Recipe(配方/执行阶段):
   `-q` 的作用是告诉 Make: "不要去真正执行编译 Target(如 `build:`)下面定义的那些 Shell 命令(即配方)".

2. `$(shell ...)` 是在 Parsing(解析阶段)执行的:
   GNU Make 在构建依赖树并加载变量时, 必须先确定所有变量的值. 对于 Makefile 中所有在解析阶段就需要计算的变量(如使用 `:=` 强制赋值, 或在 `ifeq` 条件判断里使用的 `$(shell ...)`), Make 必定会立即执行它们.

因此, 只要 Makefile 或其引入的子文件中含有 `$(shell go ...)`, 即使你加了 `-pq`, 该 `go` 命令也必然会被执行.

---

### 二、为什么执行 `go` 命令会触发下载 `go1.24.4`?

这是 Go 1.21+ 引入的 Toolchain(工具链)自动管理特性:
* 当你在本地执行任何 `go` 命令(即使只是 `go env` 或 `go version`)时, Go 会去读取当前目录下的 `go.mod`(或 `go.work`).
* 如果 `go.mod` 中声明的 Go 版本(例如 `go 1.24.4`)高于你当前操作系统中安装的 Go 版本, 且你的环境变量 `GOTOOLCHAIN` 处于默认值(`auto`), Go 官方工具链会判定本地版本过低.
* 此时, 它会静默前往官方源下载匹配的 Go 版本到本地 Module 缓存中, 并在终端打印:
  `go: downloading go1.24.4 (linux/amd64)`

---

### 三、在你的 Makefile 中, 具体是在哪里触发的?

虽然你给出的 `Makefile.core.mk` 中没有直接调用 `$(shell go ...)`(只有 `GO ?= go` 变量定义), 但注意看文件的第 `148` 和 `149` 行:

```makefile
include tools/tools.mk
include tools/lint.mk
```

在这些被 `include` 引入的子文件中(尤其是 `tools/tools.mk`), 100% 存在类似于获取 Go 环境变量的语句. 例如:

```makefile
# 常见于 tools.mk 的标准写法
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(shell go env GOBIN)
GOOS ?= $(shell go env GOOS)
```

正是这些隐藏在 `tools.mk` 或 `lint.mk` 内部的 `$(shell go env)` 触发了 Go 命令行工具, 进而由于 `go.mod` 的高版本限制触发了自动下载.

---

### 四、如何排查与解决?

#### 1. 快速定位罪魁祸首
你可以在项目根目录下执行 `grep`, 找出到底是哪个文件里的哪一行代码拉起了 `go` 的 `$(shell)`:

```bash
grep -rn "\$(shell.*go" tools/
```
这会立刻帮你锁死是在 `tools/tools.mk` 还是 `tools/lint.mk`.

#### 2. 解决方法: 禁止 Go 自动下载(推荐)
如果你不希望它私自下载高版本 Go, 可以在运行 `make` 前将 `GOTOOLCHAIN` 环境变量设为 `local`(这会强迫 Go 只使用你本地已经安装的版本, 如果版本不够会报错, 但绝不会触发自动下载):

```bash
export GOTOOLCHAIN=local
make DEBUG=1 -e -f Makefile.core.mk -pq
```

#### 3. 提前覆盖变量(避开 `$(shell)` 的执行)
如果 `tools.mk` 中使用的是 `?=`(弱赋值, 如 `GOPATH ?= $(shell go env GOPATH)`), 结合你已经使用了 `-e` 参数, 你可以提前在宿主机把这些变量导出来.

由于你已经定义了环境变量, Make 会直接跳过 `?=` 后面 `$(shell)` 的求值过程, 从而彻底避免调用 `go`:

```bash
export GOPATH=$(go env GOPATH)
export GOBIN=$(go env GOPATH)/bin
make DEBUG=1 -e -f Makefile.core.mk -pq
```

---------------------------------------------------------------------------------------------------------

---------------------------------------------------------------------------------------------------------

---------------------------------------------------------------------------------------------------------

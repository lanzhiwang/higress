#  Copyright (c) 2022 Alibaba Group Holding Ltd.

#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at

#       http:www.apache.org/licenses/LICENSE-2.0

#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# [语法说明] 这是一个 Shebang (#!). 它告诉操作系统使用哪个解释器来执行这个脚本.
# 使用 `/usr/bin/env bash` 比直接写 `/bin/bash` 更好, 因为它会在系统的 PATH 环境变量中寻找 bash, 兼容性更强(比如 macOS 的新版 bash 可能通过 brew 安装在不同路径).
#!/usr/bin/env bash

# [语法说明] `:` 是 bash 中的空命令(什么都不做, 总是返回 true).
# `${VAR:="default"}` 是 bash 变量展开的一种语法: 如果 VAR 没有被设置或为空, 则将它赋值为 "default".
# [逻辑目的] 这里为了给环境变量设置默认值. 如果用户在执行脚本前没有主动 export 这些变量, 脚本就会使用这里指定的默认值.
: "${BINARY_NAME:="hgctl"}"
: "${BINARY_NAME_WINDOWS:="hgctl.exe"}"
: "${hgctl_INSTALL_DIR:="/usr/local/bin"}"
: "${hgctl_INSTALL_DIR_WINDOWS:="${USERPROFILE}/hgctl/bin"}"

# the lowest node version required
# [语法说明] export 将变量导出到环境变量中, 使其对后续调用的子进程(如 curl, wget 等)也可见.
export VERSION

# [语法说明] `$(...)` 是命令替换语法, 会执行括号里的命令, 并将输出结果拼接到当前位置.
# `type "curl" &>/dev/null`: `type` 用于检查命令是否存在; `&>/dev/null` 将标准输出和标准错误都丢弃(实现静默检查).
# `&& echo true || echo false`: 利用逻辑与(&&)和逻辑或(||)模拟三元运算符. 如果前面的检查成功返回 0, 则输出 true, 否则输出 false.
# [逻辑目的] 检测当前系统环境中是否安装了 curl、wget、git 和 node 工具, 将布尔结果存入变量, 供后续逻辑判断使用.
HAS_CURL="$(type "curl" &>/dev/null && echo true || echo false)"
HAS_WGET="$(type "wget" &>/dev/null && echo true || echo false)"
HAS_GIT="$(type "git" &>/dev/null && echo true || echo false)"
HAS_NODE="$(type "node" &>/dev/null && echo true || echo false)"

# the lowest node version required
# [逻辑目的] 定义脚本所需的最低 Node.js 版本, 因为后续的 hgctl 代理功能强依赖特定版本的 Node 环境.
REQUIRED_NODE_VERSION="20.18.1"

# initArch discovers the architecture for this system.
# [逻辑目的] 检测系统 CPU 架构, 为了后续去下载对应架构的二进制文件(如 amd64, arm64 等).
initArch() {
    # [语法说明] uname -m 返回机器硬件名称(例如 x86_64, aarch64)
    ARCH=$(uname -m)

    # [语法说明] case ... in ... esac 是 Bash 的多分支选择结构. `*` 是通配符, 匹配任意字符.
    # [逻辑目的] 将不同系统输出的各种五花八门的硬件名称, 统一映射为 Go 语言交叉编译标准的架构名称(amd64, arm64, 386).
    case $ARCH in
    armv5*) ARCH="armv5" ;;
    armv6*) ARCH="armv6" ;;
    armv7*) ARCH="arm" ;;
    aarch64) ARCH="arm64" ;;
    x86) ARCH="386" ;;
    x86_64) ARCH="amd64" ;; # 将常见的 x86_64 映射为 amd64
    i686) ARCH="386" ;;
    i386) ARCH="386" ;;
    esac
}

# initOS discovers the operating system for this system.
# [逻辑目的] 检测操作系统类型, 并统一转化为小写, 方便后续拼接下载链接.
initOS() {
    # [语法说明] `|` 是管道符, 将 uname 的输出传给 tr. `tr '[:upper:]' '[:lower:]'` 将所有大写字母转换为小写(例如 Linux 变为 linux, Darwin 变为 darwin).
    OS="$(uname | tr '[:upper:]' '[:lower:]')"

    case "$OS" in
    # Minimalist GNU for Windows
    # [逻辑目的] 兼容 Windows 上的各种 Bash 模拟环境(如 Git Bash, Cygwin, MSYS2), 统一识别为 'windows'.
    mingw* | cygwin*) OS='windows' ;;
    esac
}

# runs the given command as root (detects if we are root already)
# [逻辑目的] 以 root 权限运行命令. 因为安装到 /usr/local/bin 需要高权限.
runAsRoot() {
    # [语法说明] `$EUID` 是 Bash 的内置变量, 代表“有效用户ID”(Effective User ID). root 用户的 EUID 总是 0.
    # `-ne` 表示不等于 (not equal). `[ ... ]` 是 bash 的条件测试语句.
    if [ $EUID -ne 0 ]; then
        # [语法说明] `"${@}"` 代表传递给当前函数的所有参数, 并且会保留原有的参数边界和引号(原样传递).
        sudo "${@}"
    else
        "${@}"
    fi
}

# verifySupported checks that the os/arch combination is supported for
# binary builds, as well whether or not necessary tools are present.
# [逻辑目的] 在正式开始前, 进行前置校验(拦截不支持的系统, 检查缺失的下载工具和依赖).
verifySupported() {
    # [语法说明] `local` 声明局部变量, 防止污染全局作用域. `\n` 是换行符.
    local supported="darwin-amd64\ndarwin-arm64\nlinux-amd64\nlinux-arm64\nwindows-amd64\nwindows-arm64\n"

    # [语法说明] `!` 表示逻辑非. `grep -q` 表示安静模式, 只返回匹配成功(0)或失败(1)的状态码, 不在终端输出文本.
    # [逻辑目的] 检查当前 OS 和 ARCH 的组合是否在官方预编译支持的列表中.
    if ! echo "${supported}" | grep -q "${OS}-${ARCH}"; then
        echo "No prebuilt binary for ${OS}-${ARCH}."
        echo "To build from source, go to https://github.com/alibaba/higress"
        # [语法说明] exit 1 退出脚本, 非 0 状态码表示执行失败.
        exit 1
    fi

    # [逻辑目的] curl 和 wget 至少需要存在一个, 否则无法下载安装包.
    if [ "${HAS_CURL}" != "true" ] && [ "${HAS_WGET}" != "true" ]; then
        echo "Either curl or wget is required"
        exit 1
    fi

    if [ "${HAS_GIT}" != "true" ]; then
        echo "[WARNING] Could not find git. It is required for plugin installation."
    fi

    # [逻辑目的] 检查 Node 依赖. 如果没有, 触发自动安装脚本; 如果有, 检查版本是否达标.
    if [ "${HAS_NODE}" != "true" ]; then
        echo "[ERROR] Could not find node. It is required for hgctl agent support."
        echo "Node.js >= ${REQUIRED_NODE_VERSION} is required."
        echo "Start to install node..."
        installNode
    else
        checkNodeVersion
    fi

}

# [逻辑目的] 检查当前已安装的 Node 版本是否符合要求.
checkNodeVersion() {
    # [语法说明] `sed 's/v//'` 用于替换操作, 将 `node -v` 输出的 'v' 前缀去掉(如 v20.18.1 变成 20.18.1), 方便做纯数字对比.
    local current_version=$(node -v | sed 's/v//')

    # [逻辑目的] 调用下面的 verifyNodeVersion 函数. 如果返回值为非 0 (失败), 则输出提示并退出.
    if ! verifyNodeVersion "$current_version" "$REQUIRED_NODE_VERSION"; then
        echo "[ERROR] Node.js version $current_version is installed, but >= ${REQUIRED_NODE_VERSION} is required."
        echo "Please upgrade Node.js or install a newer version."
        echo "Visit: https://nodejs.org/ or use nvm: https://github.com/nvm-sh/nvm"
        exit 1
    else
        echo "[INFO] Node.js version $current_version meets the requirement (>= ${REQUIRED_NODE_VERSION})"
    fi
}

# [逻辑目的] 比较两个语义化版本号(Semantic Versioning), 判断当前版本是否大于等于所需版本.
verifyNodeVersion() {
    # [语法说明] $1 和 $2 表示传入给此函数的第一个和第二个参数.
    local current=$1
    local required=$2

    # [语法说明] `cut -d. -f1`: `-d.` 指定点(.)为分隔符, `-f1` 提取第一列数据(即大版本号 Major).
    local current_major=$(echo "$current" | cut -d. -f1)
    local current_minor=$(echo "$current" | cut -d. -f2)
    local current_patch=$(echo "$current" | cut -d. -f3)

    local required_major=$(echo "$required" | cut -d. -f1)
    local required_minor=$(echo "$required" | cut -d. -f2)
    local required_patch=$(echo "$required" | cut -d. -f3)

    # [语法说明] `-gt` 是整数大于 (Greater Than), `-lt` 是整数小于 (Less Than), `-ge` 是大于等于 (Greater or Equal).
    # [逻辑目的] 按照 主版本号 -> 次版本号 -> 补丁版本号 的优先级逐级比较大小.
    if [ "$current_major" -gt "$required_major" ]; then
        return 0 # return 0 表示函数执行成功(条件满足)
    elif [ "$current_major" -lt "$required_major" ]; then
        return 1 # return 1 表示函数执行失败(条件不满足)
    fi

    if [ "$current_minor" -gt "$required_minor" ]; then
        return 0
    elif [ "$current_minor" -lt "$required_minor" ]; then
        return 1
    fi

    if [ "$current_patch" -ge "$required_patch" ]; then
        return 0
    else
        return 1
    fi
}

# [逻辑目的] 根据不同操作系统分发 Node.js 的安装逻辑.
installNode() {
    echo "Installing Node.js ${REQUIRED_NODE_VERSION}..."

    case "$OS" in
    darwin)
        installNodeMacOS
        ;;
    linux)
        installNodeLinux
        ;;
    windows)
        installNodeWindows
        ;;
    *)
        echo "[ERROR] Unsupported OS: $OS"
        echo "Please install Node.js manually from https://nodejs.org/"
        exit 1
        ;;
    esac
}

# [逻辑目的] 在 macOS 上通过 Homebrew 安装 Node.
installNodeMacOS() {
    if type "brew" &>/dev/null; then
        echo "Using Homebrew to install Node.js..."
        brew install node@20
    else
        echo "[ERROR] Homebrew not found. Please install Homebrew first:"
        # [语法说明] 这里的转义符 `\` 用于在终端打印出原封不动的安装命令文本.
        echo "  /bin/bash -c \\"\\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\\""
        echo "Or install Node.js manually from https://nodejs.org/"
        exit 1
    fi
}

# [逻辑目的] 在 Linux 体系(主要针对 Debian/Ubuntu 系)上通过 NodeSource 源自动安装 Node.js.
installNodeLinux() {
    echo "Installing Node.js via NodeSource repository..."

    if [ "${HAS_CURL}" == "true" ]; then
        # [语法说明] `sudo -E bash -`: `-E` 表示保留当前用户的环境变量传递给 sudo; `bash -` 表示接收标准输入中的内容作为 bash 脚本执行.
        curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
        sudo apt-get install -y nodejs
    elif [ "${HAS_WGET}" == "true" ]; then
        # [语法说明] `wget -qO-`: `-q` 安静模式, `-O-` 表示把下载内容输出到标准输出(而不是文件), 从而可以被管道传给 bash.
        wget -qO- https://deb.nodesource.com/setup_20.x | sudo -E bash -
        sudo apt-get install -y nodejs
    else
        echo "[ERROR] Neither curl nor wget found. Cannot install Node.js."
        echo "Please install Node.js manually from https://nodejs.org/"
        exit 1
    fi
}

# [逻辑目的] Windows 下无法无脑提供包管理器自动安装, 因此给用户打印详细的手动安装引导.
installNodeWindows() {
    echo "[ERROR] Automatic Node.js installation on Windows is not supported."
    echo "Please download and install Node.js manually from:"
    echo "  https://nodejs.org/dist/v${REQUIRED_NODE_VERSION}/node-v${REQUIRED_NODE_VERSION}-x64.msi"
    echo "Or use a package manager like Chocolatey:"
    echo "  choco install nodejs --version=${REQUIRED_NODE_VERSION}"
    exit 1
}

# checkDesiredVersion checks if the desired version is available.
# [逻辑目的] 如果用户没有指定具体的版本号 (VERSION 变量为空), 则自动去 Github 爬取最新版本号.
checkDesiredVersion() {
    if [ "$VERSION" == "" ]; then
        # Get tag from release URL
        local latest_release_url="https://github.com/alibaba/higress/releases"
        if [ "${HAS_CURL}" == "true" ]; then
            # [语法说明]
            # 1. `curl -Ls`: -L 允许重定向(跟随302等重定向), -s 安静模式.
            # 2. `grep ...`: 抓取包含目标标签形式的行.
            # 3. `sed -E 's/.*\/(v[0-9\.]+).*/\1/g'`: 使用扩展正则(-E)进行替换, 利用捕获组 `\1` 只保留纯粹的版本号(如 v1.2.3).
            # 4. `head -1`: 因为 grep 会匹配到多行历史记录, 这里只取第一行(通常是最新的 release).
            VERSION=$(curl -Ls $latest_release_url | grep 'href="/alibaba/higress/releases/tag/v[0-9]*.[0-9]*.[0-9]*\"' | sed -E 's/.*\/alibaba\/higress\/releases\/tag\/(v[0-9\.]+)".*/\1/g' | head -1)
        elif [ "${HAS_WGET}" == "true" ]; then
            VERSION=$(wget $latest_release_url -O - 2>&1 | grep 'href="/alibaba/higress/releases/tag/v[0-9]*.[0-9]*.[0-9]*\"' | sed -E 's/.*\/alibaba\/higress\/releases\/tag\/(v[0-9\.]+)".*/\1/g' | head -1)
        fi

        if [ "$VERSION" == "" ]; then
            echo "Failed to determine latest version. Please check network or set VERSION manually."
            exit 1
        fi
    fi
}

# checkhgctlInstalledVersion checks which version of hgctl is installed and
# if it needs to be changed.
# [逻辑目的] 检查本地是否已经安装过 hgctl, 且版本是否与我们要安装的版本一致. 如果一致, 就跳过下载和安装.
checkhgctlInstalledVersion() {
    # [语法说明] `[[ -f file ]]`: Bash 的高级测试结构, 检查指定的路径是否为一个存在的常规文件.
    if [[ -f "${hgctl_INSTALL_DIR}/${BINARY_NAME}" ]]; then
        # [语法说明] `grep -Eo`: `-E` 扩展正则, `-o` 表示只输出匹配到的部分(而不是输出整行).
        version=$("${hgctl_INSTALL_DIR}/${BINARY_NAME}" version --client | grep -Eo "v[0-9]+\.[0-9]+.*")
        if [[ "$version" == "$VERSION" ]]; then
            # [语法说明] `${VERSION:-latest}`: 变量展开语法. 如果 VERSION 存在且非空则使用它, 否则打印 "latest".
            echo "hgctl ${version} is already ${VERSION:-latest}"
            return 0 # return 0 表示已安装, 后续 `if ! check...` 判断会识别为 False 从而跳过安装逻辑.
        else
            echo "hgctl ${VERSION} is available. Changing from version ${version}."
            return 1 # 版本不一致, 返回 1 触发后续安装逻辑.
        fi
    else
        return 1 # 文件不存在, 返回 1 触发后续安装逻辑.
    fi
}

# downloadFile downloads the latest binary package
# for that binary.
# [逻辑目的] 拼接 Github Release 的真实下载地址, 并下载二进制压缩包.
downloadFile() {
    hgctl_DIST="hgctl_${VERSION}_${OS}_${ARCH}.tar.gz"
    if [ "${OS}" == "windows" ]; then
        # Windows 的包通常是 zip 格式.
        hgctl_DIST="hgctl_${VERSION}_${OS}_${ARCH}.zip"
    fi
    DOWNLOAD_URL="https://github.com/alibaba/higress/releases/download/$VERSION/$hgctl_DIST"

    # [语法说明] `mktemp -dt prefix-XXXXXX`: 创建一个安全的临时目录. `-d` 表示创建目录, `-t` 表示在系统的临时文件目录下创建. XXXXXX 会被替换为随机字符, 确保唯一性.
    hgctl_TMP_ROOT="$(mktemp -dt hgctl-installer-XXXXXX)"
    hgctl_TMP_FILE="$hgctl_TMP_ROOT/$hgctl_DIST"
    echo "Downloading $DOWNLOAD_URL"
    if [ "${HAS_CURL}" == "true" ]; then
        # [语法说明] -SsL: -S 在遇到错误时显示错误信息, -s 安静模式隐藏进度条, -L 跟随重定向. -o 指定输出文件.
        curl -SsL "$DOWNLOAD_URL" -o "$hgctl_TMP_FILE"
    elif [ "${HAS_WGET}" == "true" ]; then
        # [语法说明] -q 安静模式, -O 指定输出文件.
        wget -q -O "$hgctl_TMP_FILE" "$DOWNLOAD_URL"
    fi
}

# installFile installs the hgctl binary.
# [逻辑目的] 对于 Unix-like 系统, 解压下载的压缩包, 并将二进制文件复制到最终的目的地 (/usr/local/bin).
installFile() {
    hgctl_TMP="$hgctl_TMP_ROOT/$BINARY_NAME"
    mkdir -p "$hgctl_TMP"
    # [语法说明] `tar xf`: x 表示提取(extract), f 表示指定文件(file). -C 用于指定解压目标目录.
    tar xf "$hgctl_TMP_FILE" -C "$hgctl_TMP"
    hgctl_TMP_BIN="$hgctl_TMP/out/${OS}_${ARCH}/hgctl"
    echo "Preparing to install $BINARY_NAME into ${hgctl_INSTALL_DIR}"
    # 因为是往 /usr/local/bin 写文件, 这里调用上面封装的 runAsRoot 自动请求 sudo.
    runAsRoot cp "$hgctl_TMP_BIN" "$hgctl_INSTALL_DIR/$BINARY_NAME"
    echo "$BINARY_NAME installed into $hgctl_INSTALL_DIR/$BINARY_NAME"
}

# installFileWindows installs the hgctl binary for windows.
# [逻辑目的] 专门针对 Windows 的解压和安装逻辑(使用 unzip 处理 .zip 文件).
installFileWindows() {
    hgctl_TMP="$hgctl_TMP_ROOT/$BINARY_NAME"
    mkdir -p "$hgctl_TMP"
    # [语法说明] `unzip -d`: -d 指定解压的目录.
    unzip "$hgctl_TMP_FILE" -d "$hgctl_TMP"
    hgctl_TMP_BIN="$hgctl_TMP/out/${OS}_${ARCH}/hgctl.exe"
    echo "Preparing to install ${BINARY_NAME} into ${hgctl_INSTALL_DIR_WINDOWS}"
    mkdir -p ${hgctl_INSTALL_DIR_WINDOWS}
    # Windows 写入用户主目录通常不需要提升权限, 直接 cp 即可.
    cp "$hgctl_TMP_BIN" "$hgctl_INSTALL_DIR_WINDOWS/$BINARY_NAME_WINDOWS"
    echo "$BINARY_NAME installed into $hgctl_INSTALL_DIR_WINDOWS/$BINARY_NAME_WINDOWS"
}

# fail_trap is executed if an error occurs.
# [逻辑目的] 全局错误处理回调. 如果脚本执行中发生异常退出, 会打印友好的提示信息.
fail_trap() {
    # [语法说明] `$?` 是一个极其重要的 Bash 变量, 保存上一个刚刚执行完毕的命令的退出状态码.
    result=$?
    if [ "$result" != "0" ]; then
        # [语法说明] `-n` 检查变量字符串长度是否不为0(非空).
        if [[ -n "$INPUT_ARGUMENTS" ]]; then
            echo "Failed to install $BINARY_NAME with the arguments provided: $INPUT_ARGUMENTS"
        else
            echo "Failed to install $BINARY_NAME"
        fi
        echo -e "\tFor support, go to https://github.com/alibaba/higress."
    fi
    # 退出前一定要执行清理, 释放临时磁盘空间.
    cleanup
    exit $result
}

# testVersion tests the installed client to make sure it is working.
# [逻辑目的] 检查刚安装的二进制文件是否能够被系统正常调用.
testVersion() {
    dir="$hgctl_INSTALL_DIR"
    if [ "${OS}" == "windows" ]; then
        dir="$hgctl_INSTALL_DIR_WINDOWS"
    fi
    # [语法说明] `set +e`: 暂时关闭“遇到错误立即退出脚本”的机制. 这很重要, 因为下一行我们故意要去捕获失败的情况.
    set +e
    # [语法说明] `command -v` 是查找命令绝对路径的标准 POSIX 做法.
    if ! [ "$(command -v $BINARY_NAME)" ]; then
        echo "$BINARY_NAME not found. Is ${dir} on your PATH?"
        exit 1
    fi
    # [语法说明] 恢复“遇到错误立即退出”机制.
    set -e
}

# cleanup temporary files.
# [逻辑目的] 删除整个临时目录及其所有内容, 避免随着每次安装造成系统垃圾堆积.
cleanup() {
    # [语法说明] `${hgctl_TMP_ROOT:-}` 这是 Bash 防止灾难的技巧. 如果变量为空, 则返回空字符串.
    # 配合 `[[ -d ... ]]` 检查是不是有效目录. 防止因为某种原因变量为空时, 执行了 `rm -rf /` 把整个系统删掉.
    if [[ -d "${hgctl_TMP_ROOT:-}" ]]; then
        rm -rf "$hgctl_TMP_ROOT"
    fi
}

# Execution
# ----------------- Execution ----------------- #
# [逻辑目的] 下面是整个脚本的主执行流程. 上面定义的函数将在下面依次被调用.

#Stop execution on any error
# [语法说明] `trap` 用于捕获信号. `EXIT` 是一种伪信号, 表示脚本由于任何原因(正常结束或由于 set -e 异常结束)退出时, 都要执行 "fail_trap" 函数.
trap "fail_trap" EXIT

# [语法说明] `set -e` 极为重要. 它的意思是: 脚本中任何一个指令只要返回了非 0 的退出码, 整个脚本立即终止执行, 不再往下走. 防止产生不可预料的链式反应.
set -e

# 依次执行流程:
initArch            # 1. 探针: CPU 架构
initOS              # 2. 探针: 系统类型
verifySupported     # 3. 校验: 系统支持度、前置工具、Node 环境
checkDesiredVersion # 4. 业务逻辑: 获取远程想要安装的版本号

# 5. 业务逻辑: 检查如果本地版本已经一致, `!` 会把函数返回的 0(成功) 取反, 从而跳过 if 块内的重新下载流程.
if ! checkhgctlInstalledVersion; then
    downloadFile # 6. 下载安装包到临时目录
    if [ "${OS}" == "windows" ]; then
        installFileWindows # 7a. Windows解压与拷贝
    else
        installFile # 7b. Linux/Mac解压与拷贝
    fi
fi

testVersion # 8. 验证安装结果(验证PATH)
cleanup     # 9. 显式清理临时文件 (如果脚本执行成功, 最后做清理)

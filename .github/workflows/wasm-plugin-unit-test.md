这是一个非常具有生产力且设计巧妙的 GitHub Actions 脚本. 它展示了在中大型单体仓库(Monorepo)或多模块项目中, 如何通过动态矩阵(Dynamic Matrix)、增量检测、覆盖率门禁(Coverage Gate)以及自定义步骤汇总(Step Summary)来实现高效、安全的 CI/CD 流程.

以下是为你准备的两个部分: 第一部分是带有极其详尽注释的脚本代码; 第二部分是高级开发视角对其中 4 个核心高级设计模式的深度拆解.

---

### 第一部分: 带有详细注释的完整脚本

```yaml
# [工作流命名]
# 此工作流负责编译和测试基于 Go 语言的 WebAssembly (Wasm) 插件.
name: Wasm Plugin Unit Tests(GO)

# [触发器定义]
on:
  push:
    branches: [main]
    # [高级过滤器: 精确路径过滤 (Path Filtering)]
    # 目的: 限制只有当相关文件或特定的工作流定义发生修改时才触发.
    # 避免在修改无关文档、前端代码或其它语言的插件时空转流水线, 节省宝贵的 CI 计算资源.
    paths:
      - "plugins/wasm-go/extensions/"
      - ".github/workflows/wasm-plugin-unit-test.yml"
      - "go.mod"
      - "go.sum"
  pull_request:
    branches: ["*"] # 针对任何分支发起的 PR 均生效
    paths:
      - "plugins/wasm-go/extensions/"
      - ".github/workflows/wasm-plugin-unit-test.yml"
      - "go.mod"
      - "go.sum"

# [全局环境变量定义]
env:
  GO111MODULE: on   # 开启 Go Modules 功能
  CGO_ENABLED: 0    # 禁用 CGO, 确保生成纯 Go 的二进制文件, 提升跨平台移植性
  GOOS: linux       # 默认构建目标操作系统
  GOARCH: amd64     # 默认构建目标架构

# [任务定义]
jobs:

  # ================= JOB 1 =================
  # 任务: 增量检测变动了哪些插件
  # 目的: 在 Monorepo 架构中, 如果只改了插件 A, 就不应该去测试插件 B. 该 Job 负责找出本次提交中实际发生变动的插件.
  detect-changed-plugins:
    name: Detect Changed Plugins
    runs-on: ubuntu-latest
    # [核心使用方法: 跨 Job 输出 (Outputs)]
    # 将检测出来的变量暴露给流水线中的其它 Job 引用.
    outputs:
      changed-plugins: ${{ steps.detect.outputs.plugins }} # 包含变动插件列表的 JSON 数组
      has-changes: ${{ steps.detect.outputs.has-changes }}     # 标记是否有实际插件变动的布尔值

    steps:
      # 步骤 1.1: 拉取代码
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          # [关键设置: 拉取完整历史 (Full Clone)]
          # 目的: `git diff` 命令极度依赖 Git 历史记录来对比版本差异. 如果使用默认的浅克隆(fetch-depth: 1),
          # 虚拟机将无法获取分支合并前和合并后的公共祖先, 导致对比报错.
          fetch-depth: 0

      # 步骤 1.2: 增量变更检测逻辑
      - name: Detect changed plugins
        id: detect
        run: |
          # 1. 差异文件提取: 针对不同的触发事件(PR 或普通 Push)执行不同的 git 对比命令
          if [ "${{ github.event_name }}" = "pull_request" ]; then
            # PR模式: 拉取目标基准分支, 比较 PR 的源分支(HEAD)和目标分支(base_ref)的最新差异点
            git fetch origin ${{ github.base_ref }}
            CHANGED_FILES=$(git diff --name-only origin/${{ github.base_ref }}...HEAD)
          else
            # Push模式: 对比当前提交(HEAD)和它的上一个父提交(HEAD~1)
            CHANGED_FILES=$(git diff --name-only HEAD~1 HEAD)
          fi

          echo "Changed files:"
          echo "$CHANGED_FILES"

          # 2. 提取发生改变的插件名称(基于正则匹配特定的目录层级)
          CHANGED_PLUGINS=""
          for file in $CHANGED_FILES; do
            # 匹配路径如 plugins/wasm-go/extensions/my-plugin/main.go, 捕获中间的 "my-plugin"
            if [[ $file =~ ^plugins/wasm-go/extensions/([^/]+)/ ]]; then
              PLUGIN_NAME="${BASH_REMATCH[1]}"
              # 去重逻辑: 防止同一个插件因为修改了多个文件而被重复加入列表
              if [[ ! " $CHANGED_PLUGINS " =~ " $PLUGIN_NAME " ]]; then
                if [ -z "$CHANGED_PLUGINS" ]; then
                  CHANGED_PLUGINS="$PLUGIN_NAME"
                else
                  CHANGED_PLUGINS="$CHANGED_PLUGINS $PLUGIN_NAME"
                fi
              fi
            fi
          done

          # 3. 设置输出变量写入 $GITHUB_OUTPUT
          # 目的: 将 Shell 的空格分隔字符串转换为 GitHub Action 能够解析的 JSON 数组格式
          if [ -z "$CHANGED_PLUGINS" ]; then
            echo "No plugin changes detected, skipping tests"
            echo "has-changes=false" >> $GITHUB_OUTPUT
            echo "plugins=[]" >> $GITHUB_OUTPUT
          else
            echo "Changed plugins: $CHANGED_PLUGINS"
            echo "has-changes=true" >> $GITHUB_OUTPUT
            # 格式转换: 如 "pluginA pluginB" -> ["pluginA","pluginB"]
            PLUGINS_JSON=$(echo "$CHANGED_PLUGINS" | sed 's/ /","/g' | sed 's/^/["/' | sed 's/$/"]/')
            echo "PLUGINS_JSON: $PLUGINS_JSON"
            echo "plugins=$PLUGINS_JSON" >> $GITHUB_OUTPUT
          fi

  # ================= JOB 2 =================
  # 任务: 多 Job 动态矩阵并发测试
  # 目的: 针对上一步识别出来的[变动插件列表], 并行启动多个测试沙箱.
  test:
    name: Test Changed Plugins
    runs-on: ubuntu-latest
    # [执行前置条件控制 (needs 与 if)]
    needs: detect-changed-plugins # 确保增量检测先执行完
    if: needs.detect-changed-plugins.outputs.has-changes == 'true' # 只有存在变动时才运行后续测试

    strategy:
      # fail-fast: false. 如果矩阵中的某个插件编译或测试失败, 不要中断其它正在并行的插件测试.
      fail-fast: false
      # [核心使用方法: 动态构建策略矩阵 (Dynamic Matrix)]
      # fromJSON() 将 Job 1 导出的 JSON 字符串重新解析为 Actions 矩阵对象.
      # 如果有 3 个插件变动, 这里将并发启动 3 个并行的 Job.
      matrix:
        plugin: ${{ fromJSON(needs.detect-changed-plugins.outputs.changed-plugins) }}

    steps:
      # 步骤 2.1: 拉取代码
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      # 步骤 2.2: 搭建 Go 环境(缓存启用)
      - name: Set up Go 1.24
        uses: actions/setup-go@v4 # 提示: setup-go 推荐更新为 @v5 以使用最新的依赖缓存处理.
        with:
          go-version: 1.24
          cache: true # 自动开启 Go 缓存(GOCACHE / GOMODCACHE), 在多 Job 运行或后续运行中极大地缩短依赖拉取时间.

      # 步骤 2.3: 安装测试套件工具
      - name: Install test tools
        run: |
          # 安装 gotestsum, 它可以生成兼容 JUnit XML 格式的测试报告, 便于在 CI 界面清晰展示成功/失败列表
          go install gotest.tools/gotestsum@latest

      # 步骤 2.4: 将当前矩阵中的 Go 插件编译为 WASM 格式
      - name: Build WASM for ${{ matrix.plugin }}
        working-directory: plugins/wasm-go/extensions/${{ matrix.plugin }}
        run: |
          echo "Building WASM for ${{ matrix.plugin }}..."
          # [针对 WebAssembly 的局部交叉编译配置]
          # 目的: 将 Go 代码编译为 Wasm 格式.
          export GOOS=wasip1
          export GOARCH=wasm

          if ! go build -buildmode=c-shared -o main.wasm ./; then
            echo "❌ WASM build failed for ${{ matrix.plugin }}"
            exit 1
          fi

          if [ ! -f "main.wasm" ]; then
            echo "❌ WASM file not generated for ${{ matrix.plugin }}"
            exit 1
          fi
          echo "✅ WASM build successful for ${{ matrix.plugin }}"

      # 步骤 2.5: 设置 WASM_PATH 环境变量
      - name: Set WASM_PATH environment variable
        run: |
          # 目的: 将编译好的 WASM 绝对路径存入环境变量, 以便 Go 测试代码中能正确加载该 WASM 文件进行沙箱测试.
          echo "WASM_PATH=$(pwd)/plugins/wasm-go/extensions/${{ matrix.plugin }}/main.wasm" >> $GITHUB_ENV

      # 步骤 2.6: 运行测试并计算测试覆盖率
      - name: Run tests with coverage for ${{ matrix.plugin }}
        working-directory: plugins/wasm-go/extensions/${{ matrix.plugin }}
        run: |
          if [ -f "main_test.go" ]; then
            echo "Running tests for ${{ matrix.plugin }}..."

            # 使用 gotestsum 运行单元测试
            # --junitfile: 写入标准化测试报告
            # --jsonfile: 保存详细的 JSON 执行历史
            # -coverprofile: 生成 Go 覆盖率报告(.out 文件)
            gotestsum --junitfile ../../../../test-results-${{ matrix.plugin }}.xml \
                     --format standard-verbose \
                     --jsonfile ../../../../test-output-${{ matrix.plugin }}.json \
                     -- -coverprofile=coverage-${{ matrix.plugin }}.out -covermode=atomic -coverpkg=./... ./...

            echo "✅ Tests completed for ${{ matrix.plugin }}"
          else
            echo "No tests found for ${{ matrix.plugin }}, skipping..."
            # 优雅兜底: 如果有些插件确实没有写测试文件, 为了防止后期报告分析程序报错, 在这里生成一个空的 XML 框架文件
            echo '<?xml version="1.0" encoding="UTF-8"?><testsuites><testsuite name="no-tests" tests="0" failures="0" errors="0" time="0"></testsuite></testsuites>' > ../../../../test-results-${{ matrix.plugin }}.xml
          fi

      # 步骤 2.7: 持久化保存测试报告
      - name: Upload test results for ${{ matrix.plugin }}
        uses: actions/upload-artifact@v4
        # [核心技巧: 非阻塞条件 (always())]
        # 即使之前的测试运行报错变红, 该步骤仍会执行.
        # 目的: 确保在测试失败时, 我们依然能够收集到失败的 XML 报告, 以便后续步骤分析和展示.
        if: always()
        with:
          name: test-results-${{ matrix.plugin }}
          path: |
            test-results-${{ matrix.plugin }}.xml
            test-output-${{ matrix.plugin }}.json
          retention-days: 30 # 保留 30 天, 到期自动清理释放存储空间

      # 步骤 2.8: 持久化保存覆盖率文件
      - name: Upload coverage report for ${{ matrix.plugin }}
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: coverage-${{ matrix.plugin }}
          path: plugins/wasm-go/extensions/${{ matrix.plugin }}/coverage-${{ matrix.plugin }}.out
          retention-days: 30

      # 步骤 2.9: 上报覆盖率数据到第三方可视化工具 Codecov
      - name: Upload coverage to Codecov for ${{ matrix.plugin }}
        uses: codecov/codecov-action@v4
        if: always()
        env:
          CODECOV_TOKEN: ${{ secrets.CODECOV_TOKEN }}
        with:
          file: plugins/wasm-go/extensions/${{ matrix.plugin }}/coverage-${{ matrix.plugin }}.out
          flags: wasm-go-plugin-${{ matrix.plugin }}
          name: codecov-${{ matrix.plugin }}
          fail_ci_if_error: false # 即使第三方网络波动导致上报失败, 也不要让整个 CI 任务挂掉.
          verbose: true

  # ================= JOB 3 =================
  # 任务: 汇总所有的并行测试数据并运行"覆盖率安全门禁"
  # 目的: 将并行矩阵中零散的数据聚合在一起, 生成一份极具可读性的 Markdown 汇总报告打印在 GitHub Actions 界面,
  # 并且如果发现有些插件的覆盖率低于阈值(如 30%), 强制拦截并挂掉 CI.
  test-summary:
    name: Test Summary & Coverage
    runs-on: ubuntu-latest
    needs: [detect-changed-plugins, test]
    # always(): 保证在某些并行插件测试报错挂掉的情况下, 该 Job 仍会运行(因为你最需要的就是看到失败报告! ).
    if: always() && needs.detect-changed-plugins.outputs.has-changes == 'true'

    steps:
      # 步骤 3.1: 拉取代码
      - name: Checkout code
        uses: actions/checkout@v4

      # 步骤 3.2: 搭建 Go 环境(1.25)
      - name: Set up Go 1.25
        uses: actions/setup-go@v4
        with:
          go-version: 1.25
          cache: true

      # 步骤 3.3: 安装基础计算工具 bc
      - name: Install required tools
        run: |
          # bc 工具用于在 Bash 下进行带小数点的浮点数运算(Bash 原生只支持整数)
          sudo apt-get update && sudo apt-get install -y bc

      # 步骤 3.4: [核心使用方法: 批量下载产物]
      - name: Download all test results
        uses: actions/download-artifact@v4
        with:
          # pattern: 使用通配符, 将并行测试上传的多个以 `test-results-` 开头的产物打包批量下载.
          pattern: test-results-*
          merge-multiple: true # 将下载下来的文件扁平化放置在当前目录(工作区根目录), 方便下一步遍历分析.
          path: ${{ github.workspace }}

      # 步骤 3.5: 下载所有覆盖率文件
      - name: Download all coverage files
        uses: actions/download-artifact@v4
        with:
          pattern: coverage-*
          merge-multiple: true
          path: ${{ github.workspace }}

      # 步骤 3.6: 报告分析、门禁判断和 Step Summary 的编写
      - name: Generate comprehensive test summary
        run: |
          # [核心使用方法: 向 $GITHUB_STEP_SUMMARY 写入内容]
          # 所有重定向到这个临时变量的 Markdown 文本都会直接展示在 GitHub Actions Job 详情页.
          echo "## 🧪 Go Plugin Test Results" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY

          total_plugins=0
          passed_plugins=0
          failed_plugins=0
          total_tests=0
          total_failures=0
          total_errors=0

          echo "### 📊 Test Results by Plugin" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY

          # 遍历并解析所有的 XML 测试结果
          for result_file in test-results-*.xml; do
            if [ -f "$result_file" ]; then
              plugin_name=$(echo "$result_file" | sed 's/test-results-\(.*\)\.xml/\1/')
              total_plugins=$((total_plugins + 1))

              # 鲁棒的 XML 属性提权与解析
              if grep -q '<testsuite' "$result_file"; then
                tests=$(grep -o 'tests="[0-9]*"' "$result_file" | head -1 | grep -o '[0-9]*' || echo "0")
                failures=$(grep -o 'failures="[0-9]*"' "$result_file" | head -1 | grep -o '[0-9]*' || echo "0")
                errors=$(grep -o 'errors="[0-9]*"' "$result_file" | head -1 | grep -o '[0-9]*' || echo "0")
                time=$(grep -o 'time="[0-9.]*"' "$result_file" | head -1 | grep -o '[0-9.]*' || echo "0")

                tests=${tests:-0}
                failures=${failures:-0}
                errors=${errors:-0}

                total_tests=$((total_tests + tests))
                total_failures=$((total_failures + failures))
                total_errors=$((total_errors + errors))

                if [ "$failures" = "0" ] && [ "$errors" = "0" ]; then
                  echo "✅ $plugin_name: $tests tests passed in ${time}s" >> $GITHUB_STEP_SUMMARY
                  passed_plugins=$((passed_plugins + 1))
                else
                  echo "❌ $plugin_name: $tests tests, $failures failures, $errors errors in ${time}s" >> $GITHUB_STEP_SUMMARY
                  failed_plugins=$((failed_plugins + 1))
                fi
              else
                echo "⚠️ $plugin_name: No tests found" >> $GITHUB_STEP_SUMMARY
              fi
            fi
          done

          echo "" >> $GITHUB_STEP_SUMMARY
          echo "### 📈 Coverage Report" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY

          coverage_failed=false
          coverage_files=$(find ${{ github.workspace }} -name "coverage-*.out")

          if [ -n "$coverage_files" ]; then
            echo "Found coverage files:"
            echo "$coverage_files"
          fi

          # 遍历覆盖率报告
          for coverage_file in $coverage_files; do
            if [ -f "$coverage_file" ]; then
              plugin_name=$(basename "$coverage_file" | sed 's/coverage-\(.*\)\.out/\1/')

              if [ -s "$coverage_file" ]; then
                plugin_dir="plugins/wasm-go/extensions/$plugin_name"
                if [ -d "$plugin_dir" ]; then
                  # [关键工程技巧]: 必须将 `.out` 复制到对应的 Go 模块目录下运行.
                  # 目的: `go tool cover` 工具依赖 Go 模块上下文和源代码来解析, 否则会因为缺失路径关系而报错失败.
                  cp "$coverage_file" "$plugin_dir/"
                  cd "$plugin_dir"

                  # 运行官方的 go tool cover 命令计算总覆盖率数值
                  coverage_stats=$(go tool cover -func="$(basename "$coverage_file")" 2>&1 | tail -1)
                  cd - > /dev/null

                  rm -f "$plugin_dir/$(basename "$coverage_file")"
                else
                  echo "Plugin directory not found: $plugin_dir"
                  coverage_stats=""
                fi

                echo "Coverage stats result: $coverage_stats"

                if [ -n "$coverage_stats" ] && echo "$coverage_stats" | grep -q "%"; then
                  coverage_percent=$(echo "$coverage_stats" | grep -o '[0-9.]*%' | head -1 | sed 's/%//')
                  coverage_percent=${coverage_percent:-0}

                  # 浮点数比较
                  if (( $(echo "$coverage_percent > 0" | bc -l) )); then
                    # 用红、黄、绿三色可视化图标区分覆盖率等级
                    if (( $(echo "$coverage_percent >= 80" | bc -l) )); then
                      coverage_icon="🟢"
                    elif (( $(echo "$coverage_percent >= 30" | bc -l) )); then
                      coverage_icon="🟡"
                    else
                      # 触发覆盖率门禁: 低于 30%
                      coverage_icon="🔴"
                      coverage_failed=true
                    fi

                    echo "$coverage_icon $plugin_name: $coverage_percent%" >> $GITHUB_STEP_SUMMARY

                    if (( $(echo "$coverage_percent < 30" | bc -l) )); then
                      echo "❌ $plugin_name: Coverage below 30% threshold!" >> $GITHUB_STEP_SUMMARY
                    fi
                  else
                    echo "⚪ $plugin_name: No statements to cover" >> $GITHUB_STEP_SUMMARY
                  fi
                else
                  echo "⚪ $plugin_name: Coverage data unavailable" >> $GITHUB_STEP_SUMMARY
                fi
              else
                echo "⚪ $plugin_name: Coverage file is empty or invalid" >> $GITHUB_STEP_SUMMARY
              fi
            fi
          done

          echo "" >> $GITHUB_STEP_SUMMARY
          echo "📊 Coverage reports are now available on Codecov" >> $GITHUB_STEP_SUMMARY
          echo "🔗 This Commit Coverage: https://codecov.io/gh/${{ github.repository }}/commit/${{ github.sha }}" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY

          # ================= 覆盖率门禁中断控制 =================
          # 目的: 如果任意变动插件的测试覆盖率低于 30%, 即使测试全过, 整个 CI 流水线也必须红叉挂掉.
          # 强制约束开发人员提高单元测试编写规范, 保障交付质量.
          if [ "$coverage_failed" = true ]; then
            echo "### ❌ Coverage Gate Failed" >> $GITHUB_STEP_SUMMARY
            echo "" >> $GITHUB_STEP_SUMMARY
            echo "🚫 Coverage threshold not met: Some plugins have coverage below 30%" >> $GITHUB_STEP_SUMMARY
            echo "📋 Please improve test coverage before merging this PR" >> $GITHUB_STEP_SUMMARY
            echo "" >> $GITHUB_STEP_SUMMARY

            echo "Coverage gate failed - some plugins below 30% threshold"
            exit 1 # 用非 0 状态码终止构建, CI 变为红叉
          else
            echo "### ✅ Coverage Gate Passed" >> $GITHUB_STEP_SUMMARY
            echo "" >> $GITHUB_STEP_SUMMARY
            echo "🎉 All plugins meet the 30% coverage threshold" >> $GITHUB_STEP_SUMMARY
            echo "" >> $GITHUB_STEP_SUMMARY
          fi

          echo "### 🎯 Summary" >> $GITHUB_STEP_SUMMARY
          echo "- Total plugins: $total_plugins" >> $GITHUB_STEP_SUMMARY
          echo "- Passed: $passed_plugins ✅" >> $GITHUB_STEP_SUMMARY
          echo "- Failed: $failed_plugins ❌" >> $GITHUB_STEP_SUMMARY
          echo "- Total tests: $total_tests" >> $GITHUB_STEP_SUMMARY
          echo "- Total failures: $total_failures" >> $GITHUB_STEP_SUMMARY
          echo "- Total errors: $total_errors" >> $GITHUB_STEP_SUMMARY

          # 如果存在合并失败的详细信息, 动态拼装 Markdown 报错列表
          if [ $total_failures -gt 0 ] || [ $total_errors -gt 0 ]; then
            echo "" >> $GITHUB_STEP_SUMMARY
            echo "### ❌ Failed Tests Details" >> $GITHUB_STEP_SUMMARY
            echo "" >> $GITHUB_STEP_SUMMARY
            echo "Failed plugins: $failed_plugins" >> $GITHUB_STEP_SUMMARY
            echo "Total failures: $total_failures" >> $GITHUB_STEP_SUMMARY
            echo "Total errors: $total_errors" >> $GITHUB_STEP_SUMMARY
            echo "" >> $GITHUB_STEP_SUMMARY
            echo "📋 View detailed logs: [Click here](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})" >> $GITHUB_STEP_SUMMARY
            echo "" >> $GITHUB_STEP_SUMMARY

            echo "#### 📊 Failed Plugin Details" >> $GITHUB_STEP_SUMMARY
            echo "" >> $GITHUB_STEP_SUMMARY

            for result_file in test-results-*.xml; do
              if [ -f "$result_file" ]; then
                plugin_name=$(echo "$result_file" | sed 's/test-results-\(.*\)\.xml/\1/')

                failures=$(grep -o 'failures="[0-9]*"' "$result_file" | head -1 | grep -o '[0-9]*' || echo "0")
                errors=$(grep -o 'errors="[0-9]*"' "$result_file" | head -1 | grep -o '[0-9]*' || echo "0")
                failures=${failures:-0}
                errors=${errors:-0}

                if [ "$failures" -gt 0 ] || [ "$errors" -gt 0 ]; then
                  echo "$plugin_name:" >> $GITHUB_STEP_SUMMARY
                  echo "- Failures: $failures" >> $GITHUB_STEP_SUMMARY
                  echo "- Errors: $errors" >> $GITHUB_STEP_SUMMARY
                  echo "- [View plugin logs](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }})" >> $GITHUB_STEP_SUMMARY
                  echo "" >> $GITHUB_STEP_SUMMARY
                fi
              fi
            done
          fi
```

---

### 第二部分: 高级开发视角的技术设计剖析(Why)

这套流水线展示了高度专业且具备工业级规范的 CI 实践, 我们可以提炼出以下核心工程学设计:

#### 1. 单体仓库(Monorepo)的微服务化测试隔离
* 在 Wasm 插件这种存在大量子项目的单体仓库中, 如果不管三七二十一, 每次有修改都运行全部上百个插件的编译与测试, 流水线时间往往会在几十分钟以上, 开发者体验极差.
* 如何工作的: Job 1 通过对比 Git 分支公共祖先和当前 HEAD 指针提取变动的文件目录, 并打包导出为 JSON 数组; Job 2 使用 `fromJSON()` 并配合 `strategy.matrix` 将这一数组解构为一组并行的独立 Job 运行 [5].
* 设计目的: 这实现了"局部变更、局部测试"的精准控制, 测试耗时从原本的累计时长直接降低到了最慢的那个单一插件测试时长(通常只需几秒钟至几分钟).

#### 2. "非阻塞聚合"的 Job 运行模型 (`always()` 的妙用)
* 如果使用普通的 Job 依赖, 一旦 Job 2(插件单元测试)有失败, Job 3(测试汇总报告)将直接被 GitHub 跳过(Skipped).
* 如何工作的: Job 3 的声明中写了 `if: always() && needs.detect-changed-plugins.outputs.has-changes == 'true'`.
* 设计目的: 这是一个极为体贴的开发者体验设计. 如果自动化测试挂了, 开发者最渴望的事情, 一定是希望工作流在主页输出一个直观可读的汇总面板: 到底是哪个插件、哪个 case、为什么挂了. `always()` 保证了哪怕有插件测试失败, Job 3 也能稳健运行, 并优雅地把所有的错漏以 Markdown 表格的形式绘制到 GitHub UI 上.

#### 3. 覆盖率门禁检查 (Quality Gates) 与 `bc` 浮点数黑魔法
* 很多团队会发生一种现象: 虽然单测测试通过率保持在 100%, 但新提交的插件里只写了寥寥几条单测, 根本没有覆盖核心逻辑.
* 如何工作的: 脚本在最后一个汇总 Job 3 中整合了覆盖率计算和验证. 在发现有低于 30% 覆盖率的插件时, 执行 `exit 1`.
* 设计目的: 在 CI 中引入硬性的"质量门禁(Quality Gate)".
* 在 Bash 中, 原生计算工具是不支持小数比对的(例如 `29.5 < 30` 运算会抛出语法错). 该脚本通过安装 `bc` 浮点运算工具并使用 `(( $(echo "$coverage_percent < 30" | bc -l) ))` 进行验证, 从而在工程上做到了精确的覆盖率合规控制, 为生产代码构建了极佳的防线.

#### 4. 消除环境差异性: Go Tool Cover 的复制技巧
* 在 Job 3 中有一句看似多余的操作: `cp "$coverage_file" "$plugin_dir/"` 并在运行完命令后清理掉它.
* 如何工作的: Go 官方提供的 `go tool cover` 工具在统计覆盖率时, 必须要获取被统计包(Package)的 `go.mod` 声明以及所有相关的 Go 文件依赖, 否则会提示找不到符号定义.
* 设计目的: 如果直接在根目录下运行统计, 工具会因为当前路径不存在相应的 Go 模块而崩溃. 通过在 Bash 中将生成的临时数据动态移送至其子模块目录下就地解析, 解决了多模块(Go Multi-module)项目无法批量汇总统计的痛点.
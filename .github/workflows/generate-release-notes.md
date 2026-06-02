你好! 这是一份极具前瞻性和技术深度的 GitHub Actions 脚本. 

如果说传统的 CI/CD 脚本是在做"自动化构建和测试", 那么这份脚本则展示了当下最前沿的 "AI Agentic Workflow(AI 智能体工作流)". 它巧妙地结合了大模型(LLM)、MCP(模型上下文协议)以及 GitHub API, 全自动地对多个仓库的 PR 代码变更进行总结, 生成中英双语的 Release Notes, 并自动更新发布页面和提交 PR. 

我依然分为两部分为你详细拆解: 
1. 带详细注释的代码(侧重语法解析、Bash 魔法和工具链). 
2. 高级开发者视角的深度总结(侧重 AI 与 CI/CD 结合的架构设计). 

---

### 一、带详细注释的 Workflow 脚本

```yaml
name: Generate Release Notes

on:
  # 触发条件: 打 Tag 发布新版本时触发, 或手动触发
  push:
    tags:
      - "v*.*.*"
  workflow_dispatch: ~

jobs:
  generate-release-notes:
    runs-on: ubuntu-latest
    # 全局环境变量: 注入大语言模型(LLM)所需的配置
    env:
      # DASHSCOPE 是阿里云通义千问的模型服务
      DASHSCOPE_API_KEY: ${{ secrets.HIGRESS_OPENAI_API_KEY }}
      MODEL_NAME: ${{ secrets.HIGRESS_OPENAI_API_MODEL }}
      MODEL_SERVER: ${{ secrets.MODEL_SERVER }}

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0 # 拉取所有历史记录, AI 总结可能需要深度的 git log 历史

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: 1.24

      # 核心步骤 1: 构建 GitHub MCP Server (Model Context Protocol)
      # 目的: MCP 是一种让大模型安全访问外部资源的协议. 
      # 这里下载并编译了 GitHub 的 MCP 服务器, 后续 AI Agent 可以通过它读取 PR 的具体 Diff(代码变更). 
      - name: Clone GitHub MCP Server
        run: |
          git clone https://github.com/github/github-mcp-server.git
          cd github-mcp-server
          git checkout 5904a0365ec11f661ecea5c255e86860d279f3b1 # 锁定特定 commit 保证稳定性
          go build -o ../github-mcp-serve ./cmd/github-mcp-server
          cd ..
          chmod u+x github-mcp-serve

      - name: Setup Python
        uses: actions/setup-python@v4
        with:
          python-version: "3.10"

      # 核心步骤 2: 拉取 Higress 专门编写的 AI 报告生成智能体
      - name: Clone Higress Report Agent
        run: |
          git clone https://github.com/higress-group/higress-report-agent.git
          mv github-mcp-serve higress-report-agent/ # 将刚才编译的 MCP Server 塞入 Agent 目录供其调用

      # 步骤 3: 清理历史遗留文件(保证幂等性)
      - name: Clean up old release notes
        run: |
          RELEASE_VERSION=$(cat ${GITHUB_WORKSPACE}/VERSION)
          CLEAN_VERSION=${RELEASE_VERSION#v} # 语法: Bash 字符串截取, 删除开头的 'v' (例如 v1.0.0 变成 1.0.0)
          if [ -d "release-notes/${CLEAN_VERSION}" ]; then
              rm -rf release-notes/${CLEAN_VERSION}
          fi

      # ==========================================
      # 步骤 4: 元编程(动态生成爬取与生成报告的 Bash 脚本)
      # ==========================================
      - name: Create Release Report Script
        run: |
          # 语法: 使用 cat > file << 'EOF' 将其中的内容写入 generate_release_report.sh 文件
          cat > generate_release_report.sh << 'EOF'
          #!/bin/bash
          
          # 1. 下载 GitHub 自动生成的 Release 网页 HTML
          curl -L "https://github.com/${GITHUB_REPO_OWNER}/${GITHUB_REPO_NAME}/releases/tag/${RELEASE_VERSION}" -o release_page.html

          # 2. 内嵌 Python 脚本: 使用 BeautifulSoup 解析 HTML
          # 目的: 开发者可能在 Release 描述里写了一段 <h2>system prompt</h2> 告诉 AI 重点关注什么, 这里将其提取出来. 
          pip install beautifulsoup4 markdownify
          SYSTEM_PROMPT=$(python3 -c "
          import sys
          from bs4 import BeautifulSoup
          from markdownify import markdownify
          with open('release_page.html', 'r') as f:
              soup = BeautifulSoup(f, 'html.parser')
          system_prompt_header = soup.find('h2', string='system prompt')
          # ... (提取 H2 标签下的内容并转为 Markdown 格式) ...
          ")

          # 3. 解析 HTML 提取所有相关的 PR Number
          # 语法: 使用 grep 和正则匹配出所有的 /pull/1234, 并提取纯数字进行排序去重
          PR_NUMS=$(cat release_page.html | grep -o "/${GITHUB_REPO_OWNER}/${GITHUB_REPO_NAME}/pull/[0-9]*" | grep -o "[0-9]*$" | sort -n | uniq | tr '\n' ',')
          PR_NUMS=${PR_NUMS%,} # 移除末尾多余的逗号

          # 4. 提取 "Important" 的 PR
          # 目的: GitHub UI 上标粗(<strong>)的 PR 通常是核心变更, 将其单独提取传给 AI 重点分析. 
          IMPORTANT_PR_NUMS=$(cat release_page.html | grep -o "<strong>.*/pull/[0-9]*.*</strong>" | grep -o "pull/[0-9]*" | grep -o "[0-9]*" | sort -n | uniq | tr '\n' ',')
          IMPORTANT_PR_NUMS=${IMPORTANT_PR_NUMS%,}

          # 5. 调用 AI Agent 进行推理生成
          cd higress-report-agent
          pip install uv # uv 是一个用 Rust 写的极速 Python 包管理器
          uv sync

          # 组装传给 Python AI Agent 的命令行参数
          CMD_ARGS="--mode 2 --choice 2 --pr_nums ${PR_NUMS}"
          if [ -n "${IMPORTANT_PR_NUMS}" ]; then CMD_ARGS="${CMD_ARGS} --important_prs ${IMPORTANT_PR_NUMS}"; fi
          if [ -n "${SYSTEM_PROMPT}" ]; then 
              echo "${SYSTEM_PROMPT}" > temp_system_prompt.txt
              CMD_ARGS="${CMD_ARGS} --sys_prompt_file temp_system_prompt.txt"
          fi

          # 启动 AI Agent. 它会利用 MCP 读取这些 PR 的 Diff, 借助 LLM 总结出 report.md (中英双语)
          uv run report_main.py ${CMD_ARGS}
          
          # 6. 将 AI 产出的结果归档到 release-notes/ 目录下
          cd ..
          CLEAN_VERSION=${MAIN_RELEASE_VERSION#v}
          mkdir -p release-notes/${CLEAN_VERSION}
          
          # 替换标题并保存
          sed 's/# Release Notes//' report.md >>release-notes/${CLEAN_VERSION}/README_ZH.md
          sed 's/# Release Notes//' report.EN.md >>release-notes/${CLEAN_VERSION}/README.md
          EOF
          chmod +x generate_release_report.sh

      # 步骤 5: 为 Higress 主库生成 Release Notes(执行上面生成的脚本)
      - name: Generate Release Notes for Higress
        env:
          GITHUB_REPO_OWNER: alibaba
          GITHUB_REPO_NAME: higress
          REPORT_TITLE: Higress
        run: |
          export MAIN_RELEASE_VERSION=$(cat ${GITHUB_WORKSPACE}/VERSION)
          export RELEASE_VERSION=$(cat ${GITHUB_WORKSPACE}/VERSION)
          bash generate_release_report.sh

      # 步骤 6: 为 Higress Console 前端库生成 Release Notes
      # 目的: 微服务项目通常有多个仓库. 这里利用统一的流水线, 一并总结前端仓库的 PR. 
      - name: Generate Release Notes for Higress Console
        env:
          GITHUB_REPO_OWNER: higress-group
          GITHUB_REPO_NAME: higress-console
          REPORT_TITLE: Higress Console
        run: |
          export MAIN_RELEASE_VERSION=$(cat ${GITHUB_WORKSPACE}/VERSION)
          # 从 DEP_VERSION 文件中读取前端的对应版本号
          export RELEASE_VERSION=$(grep "^higress-console:" ${GITHUB_WORKSPACE}/DEP_VERSION | head -n1 | sed 's/higress-console: //')
          bash generate_release_report.sh

      # ==========================================
      # 步骤 7: 元编程(动态生成更新 GitHub Release 页面的脚本)
      # ==========================================
      - name: Create Update Release Notes Script
        run: |
          cat > update_release_note.sh << 'EOF'
          #!/bin/bash
          CLEAN_VERSION=${RELEASE_VERSION#v}

          # 1. 使用 curl 调用 GitHub API v2022-11-28 获取当前 Release 的原有信息
          RELEASE_INFO=$(curl -s -L -H "Authorization: Bearer ${GITHUB_TOKEN}" https://api.github.com/repos/${GITHUB_REPO_OWNER}/${GITHUB_REPO_NAME}/releases/tags/${RELEASE_VERSION})
          RELEASE_ID=$(echo $RELEASE_INFO | jq -r .id)
          RELEASE_BODY=$(echo $RELEASE_INFO | jq -r .body)

          # 2. 从原有的 GitHub 自动生成的 Body 中, 把"新贡献者"和"完整变更日志"这两段固定内容抠出来
          NEW_CONTRIBUTORS=$(echo "$RELEASE_BODY" | awk '/## New Contributors/{flag=1; next} /\*\*Full Changelog\*\*/{flag=0} flag' | sed 's/\\n/\n/g')
          FULL_CHANGELOG=$(echo "$RELEASE_BODY" | awk '/\*\*Full Changelog\*\*:/{print $0}' | sed 's/\*\*Full Changelog\*\*: //g' | sed 's/\\n/\n/g')

          # 3. 将 AI 生成的内容, 加上抠出来的这两段固定内容拼装在一起
          RELEASE_NOTES=$(cat release-notes/${CLEAN_VERSION}/README.md | sed 's/# /## /g')
          # ... (拼接逻辑) ...

          # 4. 生成 JSON Payload, 利用 PATCH 请求将新的 Release Notes 覆盖到 GitHub Release 页面上
          JSON_DATA=$(jq -n --arg body "$RELEASE_NOTES" '{body: $body}')
          curl -L -X PATCH -H "Authorization: Bearer ${GITHUB_TOKEN}" https://api.github.com/repos/.../${RELEASE_ID} -d "$JSON_DATA"
          EOF
          chmod +x update_release_note.sh

      # 步骤 8: 执行更新 GitHub Release 页面
      - name: Update Release Notes
        env:
          GITHUB_REPO_OWNER: alibaba
          GITHUB_REPO_NAME: higress
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          export RELEASE_VERSION=$(cat ${GITHUB_WORKSPACE}/VERSION)
          bash update_release_note.sh

      # 步骤 9: 清理各种临时工具和脚本, 防止它们被提交到代码库
      - name: Clean
        run: |
          rm generate_release_report.sh update_release_note.sh
          rm -rf higress-report-agent github-mcp-server

      # 步骤 10: 提交 PR 保存生成的文档
      # 目的: 不仅在 Release 页面展示, 还将 Markdown 文件通过 PR 的形式归档进本仓库的 `release-notes/` 目录. 
      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v7
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          commit-message: "Add release notes"
          branch: add-release-notes
          title: "Add release notes"
          labels: release notes, automated
          base: main
```

---

### 二、高级开发者视角总结(架构设计思想)

这是我目前看到的最精彩的 GitHub Actions 脚本之一, 它完美融合了 "大模型 AI 提效" 和 "工程自动化". 以下是高级维度的设计剖析: 

#### 1. AI-Native CI/CD: 引入 MCP 协议
* 背景: 以前如果要让 AI 写 Release Notes, 我们要在 CI 脚本里写极其复杂的逻辑去调用 GitHub API 获取每个 PR 的 Diff, 拼装成超长的 Prompt 喂给大模型. 
* 高阶实践: 此脚本引入了目前极其火爆的 MCP (Model Context Protocol) 机制. 它直接启动了一个 GitHub MCP Server, AI Agent (`higress-report-agent`) 只需要知道 PR 的号码, 就能通过 MCP 自动去 GitHub 检索该 PR 的具体代码差异、提交者信息等上下文. 这实现了 CI 工具链与大模型交互的解耦. 

#### 2. 黑客级的网页爬虫替代 API (The HTML Scraping Hack)
你会发现脚本里花了大篇幅用 `curl` 抓取 HTML 页面, 并用 BeautifulSoup 和 Regex 去解析, 而不是用标准的 GitHub API 获取 PR 列表. 为什么要这么费劲? 
* 洞察力: 当打 Tag 发布时, GitHub 会自动生成一个 Release Draft, 上面会把 PR 分门别类(如 `Features`, `Bug Fixes`), 且会自动加粗(`<strong>`)一些核心 PR. 
* 巧妙借力: GitHub API 只能返回冷冰冰的 Markdown 原本字符串, 无法得知哪些是被强调的. 通过抓取 HTML 页面, 脚本能精准提取被加粗的 `IMPORTANT_PR_NUMS`, 并单独提取维护者在发布页面手写的 `<h2>system prompt</h2>`(系统提示词), 将其传递给 AI. 这是极其巧妙的"以 UI 为数据源"的变通手法. 

#### 3. 动态脚本生成(元编程)应对多仓库汇总
* `cat > script.sh << 'EOF'` 是 Bash 中的"Here-Document"语法, 常用于"脚本编写脚本". 
* 为什么不直接把逻辑写在 run 里面?  因为 Higress 是一个包含底层引擎(higress)和控制台(higress-console)的复杂项目. 核心逻辑写成通用的 Bash 脚本后, 只需改变环境变量(`GITHUB_REPO_NAME`), 就能分别针对后端仓库和前端仓库运行两次, 最后汇总到同一个 `release-notes/` 目录下. 极大提高了代码的复用性. 

#### 4. "双写"架构与 Human-in-the-loop (人机协同)
生成的 Release Notes 去了哪里? 脚本采用了"双写"设计: 
1. 自动发布: 通过发送 PATCH 请求, 直接更新并覆盖刚才 GitHub 自动生成的粗糙 Release Notes. 
2. 文档归档: 使用 `peter-evans/create-pull-request` 将中英双语的 markdown 实体文件提交为 PR. 
   * 价值: 大模型偶尔会产生幻觉(Hallucination). 通过生成 PR(而不是直接 push main 分支), 它把最终的合并决定权交给了人类维护者(Human-in-the-loop). 维护者可以在 PR 里审阅 AI 写的总结是否有偏差, 修改后再合入主干, 形成完美的文档沉淀. 
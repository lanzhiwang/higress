这段 GitHub Action 脚本实现了一个智能文档翻译流水线. 其核心目的在于: 当 `helm/higress/README.md`(英文版)被修改并合并到 `main` 分支后, CI 流水线会自动捕捉到该变更, 调用 OpenAI 兼容的大语言模型(LLM)将内容翻译为中文, 并最终自动生成一个 Pull Request(PR)提请维护者合并, 从而保证中英文文档的同步.

以下是为你准备的详细内容: 第一部分是带逐行技术注释的代码; 第二部分是对其核心设计与 GitHub Actions 使用方法的深度拆解.

---

### 第一部分: 带有详细注释的完整脚本

```yaml
# [工作流命名]
# 在 GitHub Actions 界面显示的名称.
name: "Helm Docs"

# [触发器定义]
on:
  # 触发器 1: 手动触发
  # 目的: 在特殊情况下(例如 API 额度不足导致某次同步失败), 提供手动重试的机会, 不用重新提交代码.
  workflow_dispatch: ~

  # 触发器 2: 分支推送触发
  push:
    branches: [main]
    # [高级过滤器: 精确路径过滤 (Path Filtering)]
    # 限制条件: 只有当 `helm/higress/README.md` 文件发生修改时, 才会触发该工作流.
    # 目的: 大语言模型的 API 翻译是有 Token 费用成本的. 如果不加过滤, 每次对代码的任何小修改
    # 都会触发翻译脚本, 将造成极大的 API 额度浪费和不必要的网络开销.
    paths:
      - "helm/higress/README.md"

# [任务定义]
jobs:
  translate-readme:
    runs-on: ubuntu-latest # 使用最新的 Ubuntu 作为构建环境, 自带常用的 Python 运行环境.

    steps:
      # ================= step 1 =================
      # 步骤 1: 拉取代码
      - name: Checkout repository
        uses: actions/checkout@v4

      # ================= step 2 =================
      # 步骤 2: 安装依赖
      - name: Install dependencies
        # 目的: 由于后续可能涉及一些 Shell 对 JSON 的处理, 在这里预装 jq 工具.
        # 提示: 在此脚本后续并未显式使用 jq, 这通常是作为工具包初始化的一部分.
        run: |
          sudo apt-get update
          sudo apt-get install -y jq

      # ================= step 3 =================
      # 步骤 3: 对比文件变动(核心差异判断步骤)
      - name: Compare README.md
        id: compare_readme
        run: |
          cd ./helm/higress

          # 获取对比基准: 如果是 PR 环境则使用 GITHUB_BASE_REF, 否则默认对比 origin/main.
          BASE_BRANCH=${GITHUB_BASE_REF:-main}
          git fetch origin $BASE_BRANCH

          # 使用 `git diff --quiet` 检查文件是否有实际改动.
          # `--quiet` 选项会让命令在文件无差异时返回状态码 0, 在有差异时返回 1.
          if git diff --quiet origin/$BASE_BRANCH -- README.md; then
            echo "README.md has no local changes compared to $BASE_BRANCH. Skipping translation."
            # [核心使用方法: 跨步骤状态传递]
            # 使用 `>> $GITHUB_ENV` 将状态变量写入到 GitHub 特定的环境文件中.
            # 目的: 在 GitHub Actions 中, 不同步骤(Step)是独立的进程, 变量默认不共享.
            # 将变量追加到 $GITHUB_ENV 意味着后续的所有步骤都能读取到 `env.skip_translation` 这个值.
            echo "skip_translation=true" >> $GITHUB_ENV
          else
            echo "README.md has local changes compared to $BASE_BRANCH. Proceeding with translation."
            echo "skip_translation=false" >> $GITHUB_ENV
            echo "--------- diff ---------"
            git diff origin/$BASE_BRANCH -- README.md
            echo "------------------------"
          fi

      # ================= step 4 =================
      # 步骤 4: 通过 Python 调用 LLM 完成翻译
      - name: Translate README.md to Chinese
        # [核心使用方法: 条件控制 (if)]
        # 只有在上一阶段检测到 README 有实际更改(skip_translation 值为 'false')时, 才运行该翻译步骤.
        # 目的: 通过前置判断, 彻底规避空跑流水线造成的 API 资源浪费.
        if: env.skip_translation == 'false'
        # [环境变量密钥注入]
        # 将保存在 GitHub 仓库中(Settings -> Secrets)的 API URL、Key 和模型名称安全注入为局部环境变量.
        # 目的: 确保敏感的密钥信息不会泄露在构建日志或者提交的公共脚本中.
        env:
          API_URL: ${{ secrets.HIGRESS_OPENAI_API_URL }}
          API_KEY: ${{ secrets.HIGRESS_OPENAI_API_KEY }}
          API_MODEL: ${{ secrets.HIGRESS_OPENAI_API_MODEL }}
        run: |
          # [内联脚本编写: Here Document 模式]
          # 使用 `cat << 'EOF'` 生成本地 Python 脚本文件.
          # 关键技巧: 在 'EOF' 上使用单引号, 可以防止 Bash 提前对 Python 脚本中的 `$os.environ` 变量进行插值计算,
          # 从而确保这些变量名能原封不动地写入 `.py` 文件, 交由 Python 引擎解释.
          cat << 'EOF' > translate_readme.py
          import os
          import json
          import requests

          API_URL = os.environ["API_URL"]
          API_KEY = os.environ["API_KEY"]
          API_MODEL = os.environ["API_MODEL"]
          README_PATH = "./helm/higress/README.md"
          OUTPUT_PATH = "./helm/higress/README.zh.md"

          # 使用流式(Streaming)请求处理 API 返回
          # 目的: 由于 README 篇幅可能极长, 流式处理可以防止客户端因长时间等候响应而造成连接超市断开,
          # 并且可以实时将生成的数据分批次刷入磁盘, 避免高内存开销.
          def stream_translation(api_url, api_key, payload):
              headers = {
                  "Content-Type": "application/json",
                  "Authorization": f"Bearer {api_key}",
              }
              response = requests.post(api_url, headers=headers, json=payload, stream=True)
              response.raise_for_status()

              with open(OUTPUT_PATH, "w", encoding="utf-8") as out_file:
                  for line in response.iter_lines(decode_unicode=True):
                      if line.strip() == "" or not line.startswith("data: "):
                          continue
                      data = line[6:]
                      if data.strip() == "[DONE]":
                          break
                      try:
                          chunk = json.loads(data)
                          content = chunk["choices"][0]["delta"].get("content", "")
                          if content:
                              out_file.write(content)
                      except Exception as e:
                          print("Error parsing chunk:", e)

          def main():
              if not os.path.exists(README_PATH):
                  print("README.md not found!")
                  return

              with open(README_PATH, "r", encoding="utf-8") as f:
                  content = f.read()

              # 设置大模型的输入参数
              payload = {
                  "model": API_MODEL,
                  "messages": [
                      {
                          "role": "system",
                          "content": "You are a translation assistant that translates English Markdown text to Chinese. Preserve original Markdown formatting and line breaks."
                      },
                      {
                          "role": "user",
                          "content": content
                      }
                  ],
                  # [关键调参]: 将 temperature 设定在较低的 0.3.
                  # 目的: 大模型翻译属于客观翻译, 不需要文学创作或发散. 较低的温度值能够使模型
                  # 的输出更加稳定(Deterministic)、准确, 降低"幻觉"和错误排版的可能.
                  "temperature": 0.3,
                  "stream": True
              }

              print("Streaming translation started...")
              stream_translation(API_URL, API_KEY, payload)
              print(f"Translation completed and saved to {OUTPUT_PATH}.")

          if __name__ == "__main__":
              main()
          EOF

          # 执行刚刚创建的临时翻译脚本
          python3 translate_readme.py
          # 立即彻底清理临时创建的翻译文件, 防止被作为变更检测到, 保持工作区干净
          rm -rf translate_readme.py

      # ================= step 5 =================
      # 步骤 5: 自动创建 Pull Request 提请人工审查
      # 限制条件: 同样仅在有新变动发生时运行.
      - name: Create Pull Request
        if: env.skip_translation == 'false'
        # 使用 peter-evans/create-pull-request 自动提交临时分支并建 PR
        uses: peter-evans/create-pull-request@v7
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          commit-message: "Update helm translated README.zh.md"
          branch: update-helm-readme-zh # 指定 PR 来源分支.
          # 提示: 如果上次的翻译 PR 仍未被合入, 该插件会在此分支上直接覆盖追加, 不会重复开无意义的多个 PR.
          title: "Update helm translated README.zh.md"
          body: |
            This PR updates the translated README.zh.md file.

            - Automatically generated by GitHub Actions
          labels: translation, automated
          base: main # 指定 PR 试图合并入的主分支.
```

---

### 第二部分: 高级开发者视角的技术设计剖析(Why)

这段流水线脚本展现了几处精妙的、符合软件工程最佳实践的设计:

#### 1. 跨步骤状态持久化 (`$GITHUB_ENV` 的高级用法)
* 如何工作的: 在第 3 步(Compare README)中, 脚本执行了 `git diff`, 并将结果转化为了一个逻辑真假值, 使用 `echo "skip_translation=true" >> $GITHUB_ENV` 写入了环境 [4].
* 为什么要这么写:
    * 在早期的 GitHub Actions 中, 开发者往往会通过类似 `::set-output` 的黑魔法命令向外抛出值. 现代做法是统一写入 `$GITHUB_ENV` 临时环境.
    * 通过将"逻辑判断(第 3 步)"与"执行处理(第 4、5 步)"进行解耦, 你可以单独去优化"差异对比"的逻辑(比如只对比具体某几个行数), 而不会影响到后续复杂的 Python 代码的执行.

#### 2. 内联 Python 脚本 (`cat << 'EOF'`) 模式
* 为什么这么写:
    * 很多人会纠结"是把翻译脚本直接建个 `translate.py` 文件保存在仓库里, 还是写在 CI 里? "
    * 作为高级开发者, 建议采用"内联(Inlined)方式". 因为这个翻译逻辑完全是为 CI 服务的基础设施级临时脚本, 与项目本身的业务代码(Go/Rust/Helm Chart)没有任何关联.
    * 如果将它作为一个真实文件放入项目目录, 很容易污染主项目的代码结构, 还容易在代码安全检查(Lint)中触发无谓的告警. 将其作为 Here Doc 写在 CI 中, 并在运行完毕后执行 `rm -rf`, 能在保持项目根目录极致干净的同时, 完整闭环整个工作.

#### 3. 为什么是自动提 Pull Request, 而不是直接 `git push` 到 main?
* 合规性与容错机制(LLM 校验): 即便设置了 `temperature: 0.3`, 由于大语言模型本身存在不稳定性, 完全脱离人工监管直接覆盖 `main` 分支的文档依然是一种高风险操作(例如: 可能翻译错误、破坏了特定的 Markdown 链接格式、甚至出现了胡言乱语的幻觉).
* 通过自动开启一个 PR 并打上 `translation, automated` 标签, 可以将最终的"决定权"交还给开源项目的 Reviewer.
* 人类审查员只需要花几秒钟在 GitHub 上点开 PR 比对, 确认翻译无误后一键点击 `Merge`. 这种 "AI 自动发起, 人类最后把关(Human-in-the-loop)" 的协同方式, 是目前企业级 AIGC 落地最稳健、最实用的闭环路径.
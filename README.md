# hoa-news

## 项目说明

一个用于自动聚合和生成 GitHub 组织/团队动态（最新速递与周报总结）的工具代码库。通过 GitHub API 获取指定组织的 Issues、Pull Requests 和 Commits 数据，且支持大语言模型生成总结报告。项目主要包含以下两类报告生成模式：

- **最新速递 (news)**：汇总最近一天的动态，直接列出 Commits / PRs / Issues 更新列表。
- **周报总结 (summary)**：生成完整的周报，汇总一周以内各仓库和人员的贡献数据，并通过大语言模型生成总览性总结文本。

## 使用方法

1. 配置环境变量

程序运行时需要依赖以下环境变量：

- `ORG_NAME`：读取动态的 GitHub 组织名或用户名（必需）。
- `PERSONAL_ACCESS_TOKEN`：GitHub 个人访问令牌（PAT），需具备读取代码仓库等权限（必需）。
- `OPENAI_API_KEY`：用于 `summary` 命令生成周报摘要的模型 API Key（可选）。
- `OPENAI_BASE_URL`：配置模型环境的 Base URL 代理地址（可选）。
- `repos_array`：以 JSON 数组字符串形式（如 `["repoA", "repoB"]`）指定需要追踪哪些公共仓库。如不配置，则程序不会进行扫描（必需）。

2. 运行生成报告

使用 `go run` 启动入口文件并指定命令类型：

```bash
# 生成最新速递
go run cmd/main.go news

# 生成周报总结
go run cmd/main.go summary
```

执行后，生成或更新的文档（如 `news/daily.mdx`、`news/weekly/<日期>/index.mdx`）将被直接写入本地对应的目录中。

## 各模块功能

整体架构划分在 `internal/` 包和 `cmd/` 包下，其对应职责如下：

- **`cmd/main.go`**
  程序入口点。解析命令行参数（`news` 或 `summary`），检验并加载对应的环境变量，启动执行流程。

- **`internal/github/`**
  封装 GitHub API 功能，如拉取指定仓库的 Issues、Pull Requests、Commits 记录等。

- **`internal/openai/`**
  与 LLM 的 API 通信模块，用于周报总结。

- **`internal/report/`**
  执行报告生成的核心业务逻辑：
  - `news.go`：并行拉取最近一天的更新，通过模板生成并刷新 `daily.mdx`。
  - `summary.go`：并行拉取过去一周的提交记录，过滤 bot 提交，调用 `openai` 进行总结生成，持久化输出至 `weekly/` 子目录下。

- **`internal/utils/`**
  工具库。包含格式化、Bot 账号屏蔽规则以及 MDX Front-Matter 数据生成等通用处理逻辑。

- **`news/`**
  生成的 Markdown/MDX 文件数据汇总目录，存放产出的最新日报、历史周报目录结构。
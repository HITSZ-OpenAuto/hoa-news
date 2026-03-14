# hoa-news

自动聚合 GitHub 组织动态，生成日报与周报。

## 使用方法

配置环境变量：

- `ORG_NAME`：GitHub 组织名（必需）
- `PERSONAL_ACCESS_TOKEN`：GitHub PAT（必需）
- `OPENAI_API_KEY`：用于 `summary` 生成摘要（可选）
- `OPENAI_BASE_URL`：OpenAI 代理地址（可选）

运行：

```bash
go run cmd/main.go daily   # 生成日报 → news/daily.md
go run cmd/main.go weekly  # 生成周报 → news/weekly/<日期>/index.md
```

## CI 工作流

- `release.yml`：推送 `v*` tag 时构建并发布 Linux 二进制（amd64/arm64）
- `daily.yml`：每三小时生成日报
- `weekly.yml`：每周五生成周报

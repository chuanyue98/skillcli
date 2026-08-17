# sk · Agent Skills 包管理器

像 npm / uv 一样管理 AI Agent 技能（SKILL.md）——**按名字从注册表安装**（SkillHub 站：https://skillhub-ai.vercel.app），**一次命令装到多个 agent**（Claude Code / Codex CLI / Cursor / Gemini CLI / 通用 `~/.agents`）。

```
SkillHub 站（注册表：384 技能 · 评分 · 热度 · 分类）
        │ HTTP
        ▼
sk（客户端：search / install / update / list / remove）
        │ 写入
        ▼
~/.claude/skills · ~/.codex/skills · ~/.cursor/skills · ~/.gemini/skills · ~/.agents/skills
```

## 安装

```bash
go install github.com/chuanyue98/skillcli@latest   # 或
cd skillcli && go build -o sk . && sudo mv sk /usr/local/bin/
```

## 用法

```bash
# 按名字安装（npm install 体验）：注册表解析 → 自动下载 → 装到已检测的 agent
sk install brainstorming
sk install business-growth-skills -a claude,codex

# 搜索注册表（显示评分 / 星数 / 分类）
sk search 头脑风暴
sk search brainstorming --limit 20 --json

# 直接从仓库安装（保留原方式）
sk add anthropics/skills
sk add obra/superpowers@brainstorming

# 多 agent 统一管理（核心差异化）
sk list                    # 一次看到所有 agent 已装技能
sk update                  # 按 lockfile 更新所有已装技能（拉取仓库最新默认分支）
sk remove brainstorming    # 从所有 agent 卸载（同步清理 lockfile）

# 项目级安装（仅 claude / gemini 支持）
sk add owner/repo --project
```

## 注册表

- 默认注册表：`https://skillhub-ai.vercel.app`（SkillHub 站公开 API）
- 私有/自建源：`SKILLCLI_REGISTRY=https://你的地址 sk install xxx`
- 同名冲突解析策略：**官方 > 评分 > 星数**；`sk add owner/repo@skill` 可精确指定
- 版本策略：始终拉取仓库**默认分支**；lockfile 记录安装时间，`sk update` 重新拉取

## 安装记录（lockfile）

安装成功后写入 `~/.config/skillcli/sk.lock`（JSON），记录每个技能的来源仓库、技能名、安装时间。`sk update` 基于它批量更新，`sk remove` 同步清理。

## 支持的 agent 与技能目录

| agent | 全局目录 | 项目级 |
|---|---|---|
| Claude Code | `~/.claude/skills/` | `.claude/skills/` |
| Codex CLI | `~/.codex/skills/` | — |
| Cursor | `~/.cursor/skills/` | — |
| Gemini CLI | `~/.gemini/skills/` | `.gemini/skills/` |
| 通用 | `~/.agents/skills/` | — |

## 设计

- **纯 Go 标准库**，零第三方依赖，单二进制分发
- 拉取走 GitHub codeload tarball，不依赖 API 配额
- 安装成功后向注册表上报计数（站上热度数据包含 CLI 安装）
- 卸载前校验目录内含 `SKILL.md`，绝不盲删
- 中文优先的输出与文档

## 路线图

- [x] `sk search`：接入 SkillHub 站数据（评分 / 热度 / 分类）
- [x] `sk install <技能名>`：按名字从注册表安装
- [x] `sk update`：按 lockfile 更新已装技能
- [ ] 版本化安装 `name@tag`、lockfile 支持项目级安装
- [ ] 私有 registry / 离线源
- [ ] Windows / macOS 真机测试

# sk · Agent Skills 包管理器

像 npm / uv 一样管理 AI Agent 技能（SKILL.md），**一次命令装到多个 agent**（Claude Code / Codex CLI / Cursor / Gemini CLI / 通用 `~/.agents`）。

## 安装

```bash
go install github.com/chuanyue98/skillcli@latest   # 或
cd skillcli && go build -o sk . && sudo mv sk /usr/local/bin/
```

## 用法

```bash
# 安装：自动检测本机已安装的 agent，装到各自的技能目录
sk add anthropics/skills

# 指定技能 + 指定 agent
sk add obra/superpowers@brainstorming -a claude,codex

# 装到全部支持的 agent
sk add vercel-labs/agent-skills -a '*'

# 项目级安装（仅 claude / gemini 支持）
sk add owner/repo --project

# 多 agent 统一管理（核心差异化）
sk list                    # 一次看到所有 agent 已装技能
sk remove brainstorming    # 从所有 agent 卸载
sk remove brainstorming -a claude   # 只从 Claude Code 卸载
```

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
- 卸载前校验目录内含 `SKILL.md`，绝不盲删
- 中文优先的输出与文档

## 路线图

- [ ] `sk search`：接入 SkillHub 站数据（评分 / 热度 / 分类）
- [ ] `sk update`：检查并更新已装技能
- [ ] 私有 registry / 离线源
- [ ] Windows / macOS 真机测试

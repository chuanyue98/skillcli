# SkillHub 技能生态 · 需求文档与功能架构

> 版本：v0.1（需求评审稿）
> 状态：待评审
> 范围：SkillHub 站（注册表）+ skillcli（CLI 客户端）的整体设计

---

## 1. 背景与定位

### 1.1 现状

- **SkillHub 站**（https://skillhub-ai.vercel.app/）：聚合 384 个 GitHub 开源 Agent Skills（SKILL.md），提供搜索、标签筛选、分类浏览（11 个职业分类）、质量评分（A-D）、复制计数、热榜、Official 徽章、中英切换、公开 REST API。
- **skillcli**（`sk`，独立仓库）：Go 写的多 agent 技能包管理器，MVP 已支持 `add / remove / list`，自动检测 Claude Code / Codex / Cursor / Gemini / 通用 `~/.agents` 五类技能目录，从 GitHub codeload tarball 直接下载。

### 1.2 目标

把 SkillHub 做成 **Agent Skills 领域的 npm / uv**：

```
SkillHub 站 = 注册表（目录、评分、热度、版本）
skillcli    = 客户端（搜索、按名字安装、多 agent 统一管理）
```

核心闭环：**站上收录技能 → CLI 按名字即可搜到、装到本地任意 agent**。用户不再需要知道技能在哪个仓库。

### 1.3 差异化定位（相对 skills.sh）

| 维度 | skills.sh | skillcli + SkillHub |
|---|---|---|
| 安装 | `npx skills add owner/repo` | 同样支持，另加**按名字安装**（注册表解析） |
| 管理 | 装完分散在各工具，无统一视图 | `sk list` 一次看所有 agent 已装技能，跨工具卸载 |
| 数据 | 仅索引 | 质量评分、热度、职业分类（注册表自带信任信号） |
| 语言 | 英文 | 中文优先的输出与文档 |
| 扩展 | 中心化公开源 | 预留私有 registry |

### 1.4 非目标（本期不做）

- 不做技能编辑器 / 图形化 GUI
- 不做技能的沙箱执行或运行时
- 不做技能市场支付 / 付费技能

---

## 2. 用户画像与场景

| 用户 | 核心诉求 | 关键场景 |
|---|---|---|
| **Agent 使用者**（Claude Code / Codex / Cursor / Gemini 用户） | 快速找到好技能并装进自己的工具 | 搜"写文档的技能" → 看评分/星数 → `sk install xxx` → 装到 Claude Code + Codex 两个工具 |
| **技能作者** | 让自己的技能被更多人安装 | 发布到 GitHub → 收录进 SkillHub → 看安装计数与热度 |
| **团队/企业** | 统一管理团队技能、内网可用 | 私有 registry、离线安装（P2） |

---

## 3. 核心概念与数据模型

### 3.1 概念

- **Skill（技能）**：一个 SKILL.md 及其所在目录，全局唯一标识 `owner/repo/name`。
- **Registry（注册表）**：SkillHub 站的公开 API，提供技能索引、搜索、详情、评分、热度。
- **Agent（工具）**：本地 AI coding 工具及其技能目录规范（全局/项目级）。
- **Lockfile（锁文件）**：记录本机已装技能及其来源版本，支持可复现管理与 `update`。

### 3.2 Skill 数据模型（站侧，已有）

```ts
SkillSnapshot {
  id:          "owner/repo/name"      // 全局唯一
  name:        string                 // 技能名（frontmatter，缺省取目录名）
  description: string                 // 用途说明（agent 决定何时加载的关键）
  body:        string                 // SKILL.md 正文
  tags:        string[]
  author?:     string
  version?:    string                 // frontmatter 版本号（注意：≠ git tag，见 §6.1）
  license?:    string
  path:        string                 // 仓库内 SKILL.md 路径
  install:     string                 // npx skills add owner/repo --skill name
  repo:        { fullName, description, stars, htmlUrl, updatedAt }
  official?:   boolean                // 官方来源徽章
  category:    string                 // 职业分类（11 类）
  score:       { total, level, items } // 质量评分（A-D，14 项明细）
}
```

### 3.3 注册表协议（API，站侧）

| 端点 | 用途 | 现状 |
|---|---|---|
| `GET /api/skills?q=&tags=&category=&sort=&page=&limit=&fields=` | 索引 / 搜索 / 筛选 / 排序 / 分页 / 字段裁剪，带 CORS | ✅ 已有 |
| `GET /api/skills/{id}` | 单技能详情 | ✅ 已有 |
| `POST /api/counts` | 安装/复制计数上报 | ✅ 已有 |
| `GET /api/trending` | 近 7 天热榜 | ✅ 已有 |

**待补充（M1）**：索引项需显式带下载所需字段 `ref`（分支名或 tag），避免 CLI 端猜测。

---

## 4. 功能需求

优先级：**P0 = 本期必须（注册表接入）**，P1 = 下一迭代，P2 = 远期。

### P0：注册表接入（npm install 体验）

| # | 需求 | 验收标准 |
|---|---|---|
| F1 | `sk search <关键词>` | 调注册表 API，列表展示：技能名、评分等级、星数、分类、一句话描述。支持 `--limit`、`--json` |
| F2 | `sk install <技能名>` | 按名字解析到来源仓库 → 下载 → 装到指定 agent（复用现有 add 的 agent 逻辑）。**不要求用户知道 owner/repo** |
| F3 | `sk update` | 对比本机已装技能与注册表版本，列出可升级项，`--yes` 批量升级 |
| F4 | 名字冲突解析 | 同名技能存在多个来源时：**官方 > 评分 > 星数** 排序取最优；`--id owner/repo/name` 可精确指定；交互列出候选（P1 完善） |
| F5 | 安装计数上报 | `install` 成功后 POST 到 `/api/counts`，让站上热度数据包含 CLI 安装 |

### P1：版本与可复现管理

| # | 需求 | 说明 |
|---|---|---|
| F6 | Lockfile（`sk.lock`） | 记录每个已装技能的来源 `owner/repo + path + ref + 安装时间`，放 `~/.config/skillcli/` 或项目根。支撑 update/回滚 |
| F7 | 版本化安装 `sk install name@tag` | 从注册表记录或显式 tag 拉取指定版本 |
| F8 | 多 agent 冲突检测 | 同一技能名在不同 agent 版本不一致时告警 |
| F9 | 站端收录自动化 | 作者提交 GitHub 仓库 → 自动抓取、评分、收录（GitHub App / webhook 或定时任务） |

### P2：扩展

| # | 需求 | 说明 |
|---|---|---|
| F10 | 私有 registry | `sk config set registry https://内网地址`，企业内网离线可用 |
| F11 | 平台支持 | Windows / macOS 真机验证与 CI |
| F12 | 安全校验 | 下载内容校验（sha256）、官方来源签名、`--no-scripts` 类保护 |
| F13 | 技能依赖 | SKILL.md 声明依赖其他技能（当前生态少见，需先调研格式） |
| F14 | 发布协议 | `sk publish`：作者把本地技能目录发布到注册表（需认证，P2） |

---

## 5. 功能架构

### 5.1 整体架构

```
                        ┌────────────────────────────────────────────┐
                        │              SkillHub 站（registry）         │
                        │  /api/skills 索引/搜索/详情  /api/trending   │
                        │  评分 · 热度 · 分类 · Official · 多语言       │
                        └───────────────▲────────────────────────────┘
                                        │ HTTP (JSON, CORS)
   技能作者发布 ──► GitHub ──► sync ──► │
                                        │
                        ┌───────────────▼────────────────────────────┐
                        │              skillcli（客户端）              │
                        │  search ──► registry.go（解析/缓存索引）      │
                        │  install ──► resolver → github.go（tarball） │
                        │  list/remove/update ◄── lockfile.go         │
                        └───────────────┬────────────────────────────┘
                                        │ 写入技能目录
   ┌──────────┬───────────┬────────────┼─────────────┬──────────────┐
   ▼          ▼           ▼            ▼             ▼              ▼
~/.claude  ~/.codex   ~/.cursor   ~/.gemini   ~/.agents    项目级 .claude/skills
 /skills    /skills    /skills     /skills     /skills      .gemini/skills
```

### 5.2 CLI 模块划分（skillcli 仓库）

```
skillcli/
├── main.go            # 命令分发（add/remove/list/search/install/update/help）
├── agent.go           # agent 定义 + 检测（已有）
├── github.go          # tarball 下载/解压（已有，需支持指定 ref）
├── install.go         # 安装逻辑（已有，改造为接收解析结果）
├── list.go            # 多 agent 列表（已有）
├── remove.go          # 卸载（已有）
├── registry.go        # 【新增】注册表 API 客户端：search / resolve(name) / update 检查
├── resolver.go        # 【新增】名字 → (owner, repo, path, ref) 解析 + 冲突排序
├── lockfile.go        # 【新增】sk.lock 读写、update 比对
└── output.go          # 【新增】统一输出：彩色终端 / --json
```

### 5.3 关键数据流

**search**：`sk search "写文档"` → `GET /api/skills?q=写文档&sort=score` → 表格/JSON 输出

**install（按名字）**：
```
sk install brainstorming
  → registry.resolve("brainstorming")        # API 搜索 + 冲突排序 → owner/repo/path/ref
  → github.go 下载 tarball（指定 ref）          # 复用现有逻辑
  → 解压 → 定位 SKILL.md 所在目录 → 复制到各 agent 技能目录
  → POST /api/counts（上报，P0 F5）
  → 写 lockfile
```

**update**：读 lockfile → 逐个查注册表最新版本 → 比对 → 下载替换

### 5.4 失败与降级

| 场景 | 行为 |
|---|---|
| 注册表 API 不可达 | 报错提示，`install` 回退到 `owner/repo` 直接下载（现有路径） |
| 按名字找不到 | 提示最接近的候选（用 API 的搜索建议） |
| 同名冲突且无官方/评分数据 | 列出全部候选 + 评分/星数，要求 `--id` 精确指定 |
| GitHub 下载失败 | 明确报错（仓库不存在 / 网络），不动本地已有安装 |
| 重复安装 | 检测同名已存在 → 跳过并提示（现有逻辑） |

---

## 6. 关键技术决策（需拍板）

### 6.1 版本来源：frontmatter version vs git tag

- 现状：站索引的 `version` 来自 SKILL.md frontmatter（如 `2.9.0`），**不保证对应 git tag**。
- 方案 A（✅ **已定，M1 采用**）：下载始终用**仓库默认分支**，lockfile 记录 `ref = 默认分支 + 获取时间`；`update` 即重新拉默认分支。简单可靠，与现状一致。
- 方案 B（P1 再做）：站端同步时探测仓库 tag/release，索引记录 `version → git ref` 映射；支持 `install name@tag` 精确版本。工程量大，需要站端加探测逻辑。

### 6.2 计数上报的防刷与真实度

- CLI 上报与网页复制共用 `POST /api/counts`，需在请求带 `source=cli|web`，站端热榜可区分。
- 防刷：简单限流（IP/技能维度）即可，不追求严格。

### 6.3 项目级 vs 全局安装

- 已有 `--project` 支持（claude/gemini）。lockfile 需同时记录**全局与项目级**两类安装，`update` 都覆盖。

### 6.4 安全边界

- 技能本质是「注入 agent 的指令文本」，装错有被误导风险。站端已有评分 + Official 徽章；CLI 端 P2 再加下载校验。
- `sk list` 应能显示来源（哪个仓库装的），方便用户审计。

---

## 7. 里程碑

| 里程碑 | 内容 | 依赖 |
|---|---|---|
| **M1（本期）** | F1 search、F2 install-by-name、F3 update、F5 计数上报 | 站 API 补 `ref` 字段；CLI 新增 registry/resolver/lockfile |
| **M2** | F4 冲突交互选择、F6 lockfile 完善、F7 版本化安装 | M1 |
| **M3** | F9 站端收录自动化、F8 冲突检测 | M2 |
| **M4** | F10 私有 registry、F11 平台 CI、F12 安全校验、F13/F14 | 视需求 |

---

## 8. 风险清单

| 风险 | 影响 | 缓解 |
|---|---|---|
| 技能名大量冲突（同名不同仓库） | 安装错技能 | 冲突排序策略 + 精确 `--id` + 候选列表 |
| frontmatter version 与仓库实际内容脱节 | update 语义模糊 | M1 用默认分支方案，version 仅展示用 |
| 注册表 API 是静态快照，更新滞后 | 新技能/新版本看不到 | sync 定时刷新 + M3 收录自动化 |
| 站与 CLI 是两套部署，协议变更需同步 | 版本不匹配 | API 加版本号字段，CLI 做兼容解析 |

package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

var (
	errNoAgent      = errors.New("没有检测到已安装的 agent（用 -a '*' 指定全部）")
	errUnknownAgent = errors.New("未知 agent，可选: claude, codex, cursor, gemini, agents")
)

type cliFlags struct {
	agents  string // -a：逗号分隔或 '*'
	project bool   // --project：装到项目级目录
	limit   int    // search --limit
	json    bool   // search --json
	help    bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}
	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "add", "a", "install", "i":
		flags, positional, err := parseFlags(rest)
		if err != nil {
			return err
		}
		if flags.help || len(positional) == 0 {
			fmt.Println("用法: sk add owner/repo[@skill] 或 sk install <技能名> [-a claude,codex|*] [--project]")
			return nil
		}
		home, _ := os.UserHomeDir()
		cwd, _ := os.Getwd()
		return cmdAdd(positional, flags, home, cwd)
	case "remove", "rm":
		flags, positional, err := parseFlags(rest)
		if err != nil {
			return err
		}
		if flags.help || len(positional) == 0 {
			fmt.Println("用法: sk remove <技能名> [-a claude,codex|*] [--project]")
			return nil
		}
		home, _ := os.UserHomeDir()
		cwd, _ := os.Getwd()
		return cmdRemove(positional, flags, home, cwd)
	case "list", "ls":
		flags, _, err := parseFlags(rest)
		if err != nil {
			return err
		}
		home, _ := os.UserHomeDir()
		cwd, _ := os.Getwd()
		return cmdList(nil, flags, home, cwd)
	case "search", "s":
		flags, positional, err := parseFlags(rest)
		if err != nil {
			return err
		}
		return cmdSearch(positional, flags)
	case "update", "up":
		flags, _, err := parseFlags(rest)
		if err != nil {
			return err
		}
		home, _ := os.UserHomeDir()
		cwd, _ := os.Getwd()
		return cmdUpdate(nil, flags, home, cwd)
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("未知命令 %q（可用: add / install / remove / list / search / update / help）", cmd)
	}
}

// parseFlags 解析共享 flag：-a/--agent、--project、--limit、--json、-h/--help。
func parseFlags(args []string) (*cliFlags, []string, error) {
	f := &cliFlags{}
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-a" || arg == "--agent":
			if i+1 >= len(args) {
				return nil, nil, errors.New("-a 需要参数，如 -a claude,codex 或 -a '*'")
			}
			i++
			f.agents = args[i]
		case arg == "--project" || arg == "-p":
			f.project = true
		case arg == "--limit":
			if i+1 >= len(args) {
				return nil, nil, errors.New("--limit 需要参数，如 --limit 10")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n <= 0 {
				return nil, nil, fmt.Errorf("--limit 需要正整数，收到: %q", args[i])
			}
			f.limit = n
		case arg == "--json":
			f.json = true
		case arg == "-h" || arg == "--help":
			f.help = true
		case len(arg) > 0 && arg[0] == '-':
			return nil, nil, fmt.Errorf("未知参数 %q", arg)
		default:
			positional = append(positional, arg)
		}
	}
	return f, positional, nil
}

func printHelp() {
	fmt.Println(`sk — Agent Skills 包管理器（多 agent 统一管理 + 注册表）

用法:
  sk install <技能名>          按名字从注册表安装（npm install 体验）
  sk add owner/repo[@skill]   直接从仓库安装
  sk search <关键词>           搜索注册表（评分/星数/分类）
  sk update                   按 lockfile 更新所有已装技能
  sk remove <技能名>          从所有/指定 agent 卸载
  sk list                     列出所有 agent 已安装的技能
  sk help                     显示帮助

参数:
  -a, --agent <列表|*>       指定目标 agent（claude,codex,cursor,gemini,agents；'*' = 全部）
  -p, --project              装到项目级目录（仅 claude / gemini 支持）
      --limit N              search 结果数量（默认 10）
      --json                 search 输出 JSON
  -h, --help                 显示帮助

环境变量:
  SKILLCLI_REGISTRY          注册表地址（默认 https://skillhub-ai.vercel.app）

示例:
  sk install brainstorming
  sk install business-growth-skills -a claude,codex
  sk search 文档
  sk add anthropics/skills
  sk add obra/superpowers@brainstorming
  sk list
  sk update
  sk remove brainstorming`)
}

package main

import (
	"errors"
	"fmt"
	"os"
)

var (
	errNoAgent      = errors.New("没有检测到已安装的 agent（用 -a '*' 指定全部）")
	errUnknownAgent = errors.New("未知 agent，可选: claude, codex, cursor, gemini, agents")
)

type cliFlags struct {
	agents  string // -a：逗号分隔或 '*'
	project bool   // --project：装到项目级目录
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
	case "add", "a":
		flags, positional, err := parseFlags(rest)
		if err != nil {
			return err
		}
		if flags.help || len(positional) == 0 {
			fmt.Println("用法: sk add owner/repo[@skill] [-a claude,codex|*] [--project]")
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
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("未知命令 %q（可用: add / remove / list / help）", cmd)
	}
}

// parseFlags 解析共享 flag：-a/--agent、--project、-h/--help。
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
	fmt.Println(`sk — Agent Skills 包管理器（多 agent 统一管理）

用法:
  sk add owner/repo[@skill]  安装技能到已检测到的 agent
  sk remove <技能名>          从所有/指定 agent 卸载
  sk list                    列出所有 agent 已安装的技能
  sk help                    显示帮助

参数:
  -a, --agent <列表|*>       指定目标 agent（claude,codex,cursor,gemini,agents；'*' = 全部）
  -p, --project              装到项目级目录（仅 claude / gemini 支持）
  -h, --help                 显示帮助

示例:
  sk add anthropics/skills
  sk add obra/superpowers@brainstorming
  sk add vercel-labs/agent-skills -a claude,codex
  sk add owner/repo -a '*' --project
  sk list
  sk remove brainstorming`)
}

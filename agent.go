package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Agent 描述一个支持 Agent Skills 的工具及其技能目录规范。
type Agent struct {
	ID         string // 命令/标识，如 claude、codex
	Name       string // 展示名
	GlobalDir  string // 全局技能目录
	ProjectDir string // 项目级技能目录（不支持则为空）
}

// Supported 当前支持的全部 agent。
func Supported(home string) []Agent {
	return []Agent{
		{
			ID:         "claude",
			Name:       "Claude Code",
			GlobalDir:  filepath.Join(home, ".claude", "skills"),
			ProjectDir: filepath.Join(".claude", "skills"),
		},
		{
			ID:        "codex",
			Name:      "Codex CLI",
			GlobalDir: filepath.Join(home, ".codex", "skills"),
		},
		{
			ID:        "cursor",
			Name:      "Cursor",
			GlobalDir: filepath.Join(home, ".cursor", "skills"),
		},
		{
			ID:         "gemini",
			Name:       "Gemini CLI",
			GlobalDir:  filepath.Join(home, ".gemini", "skills"),
			ProjectDir: filepath.Join(".gemini", "skills"),
		},
		{
			ID:        "agents",
			Name:      "通用 (~/.agents)",
			GlobalDir: filepath.Join(home, ".agents", "skills"),
		},
	}
}

// Detect 检测本机已安装的 agent：按可执行文件或目录存在性判断。
func Detect(home string) []Agent {
	var out []Agent
	for _, a := range Supported(home) {
		if isInstalled(a, home) {
			out = append(out, a)
		}
	}
	return out
}

func isInstalled(a Agent, home string) bool {
	switch a.ID {
	case "cursor", "agents":
		// 桌面应用 / 通用目录：看目录是否存在
		_, err := os.Stat(a.GlobalDir)
		return err == nil
	default:
		_, err := exec.LookPath(a.ID)
		return err == nil
	}
}

// resolveAgents 把 -a 参数解析成目标 agent 列表。
// -a '*' → 全部支持；-a ''（未指定）→ 已检测到的；否则按逗号分隔匹配。
func resolveAgents(param string, home string) ([]Agent, error) {
	all := Supported(home)
	if param == "" {
		detected := Detect(home)
		if len(detected) == 0 {
			return nil, errNoAgent
		}
		return detected, nil
	}
	if param == "*" {
		return all, nil
	}
	ids := map[string]bool{}
	for _, s := range splitComma(param) {
		ids[s] = true
	}
	var out []Agent
	for _, a := range all {
		if ids[a.ID] {
			out = append(out, a)
		}
	}
	if len(out) == 0 {
		return nil, errUnknownAgent
	}
	return out, nil
}

func splitComma(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

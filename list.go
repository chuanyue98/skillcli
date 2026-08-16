package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func cmdList(args []string, flags *cliFlags, home, cwd string) error {
	targets, err := resolveAgents(flags.agents, home)
	if err != nil {
		return err
	}

	type row struct {
		agent string
		skill string
		dir   string
	}
	var rows []row
	for _, a := range targets {
		dirs := []string{a.GlobalDir}
		if flags.project && a.ProjectDir != "" {
			dirs = append(dirs, filepath.Join(cwd, a.ProjectDir))
		}
		for _, d := range dirs {
			skills, err := listSkillDirs(d)
			if err != nil {
				continue
			}
			for _, s := range skills {
				scope := "global"
				if d == filepath.Join(cwd, a.ProjectDir) {
					scope = "project"
				}
				rows = append(rows, row{agent: a.Name + "(" + scope + ")", skill: s, dir: filepath.Join(d, s)})
			}
		}
	}

	if len(rows) == 0 {
		fmt.Println("📭 没有已安装的技能。试试：sk add owner/repo")
		return nil
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].agent != rows[j].agent {
			return rows[i].agent < rows[j].agent
		}
		return rows[i].skill < rows[j].skill
	})

	// 按 agent 分组打印
	cur := ""
	for _, r := range rows {
		if r.agent != cur {
			cur = r.agent
			fmt.Printf("\n%s\n", cur)
		}
		fmt.Printf("  %-30s %s\n", r.skill, r.dir)
	}
	return nil
}

// listSkillDirs 列出目录下所有含 SKILL.md 的技能子目录。
func listSkillDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		md := filepath.Join(dir, e.Name(), "SKILL.md")
		if info, err := os.Stat(md); err == nil && !info.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

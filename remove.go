package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func cmdRemove(args []string, flags *cliFlags, home, cwd string) error {
	if len(args) < 1 {
		return fmt.Errorf("用法: sk remove <技能名> [-a agents]")
	}
	name := args[0]
	targets, err := resolveAgents(flags.agents, home)
	if err != nil {
		return err
	}

	removed := 0
	for _, a := range targets {
		dirs := []string{a.GlobalDir}
		if flags.project && a.ProjectDir != "" {
			dirs = append(dirs, filepath.Join(cwd, a.ProjectDir))
		}
		for _, d := range dirs {
			dest := filepath.Join(d, name)
			md := filepath.Join(dest, "SKILL.md")
			info, err := os.Stat(md)
			if err != nil || info.IsDir() {
				continue // 该 agent 没装这个技能
			}
			// 安全校验：只删除含 SKILL.md 的目录，绝不盲删
			if err := os.RemoveAll(dest); err != nil {
				fmt.Printf("⚠️  %s：删除失败 %v\n", a.Name, err)
				continue
			}
			fmt.Printf("🗑️  %s → 已卸载 %s\n", a.Name, dest)
			removed++
			// 同步清理 lockfile（按技能名匹配所有来源记录）
			if err := removeFromLock(home, name); err != nil {
				fmt.Printf("⚠️  lockfile 清理失败: %v\n", err)
			}
		}
	}
	if removed == 0 {
		fmt.Printf("ℹ️  没找到已安装的技能 %q\n", name)
	}
	return nil
}

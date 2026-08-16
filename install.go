package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// parsePackage 解析 owner/repo[@skill] 或完整 GitHub URL。
func parsePackage(pkg string) (owner, repo, skill string, err error) {
	s := strings.TrimSpace(pkg)
	s = strings.TrimSuffix(s, "/")
	// https://github.com/owner/repo
	if strings.HasPrefix(s, "https://github.com/") {
		s = strings.TrimPrefix(s, "https://github.com/")
	}
	if i := strings.IndexByte(s, '@'); i >= 0 {
		skill = s[i+1:]
		s = s[:i]
	}
	parts := strings.Split(s, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("包名格式应为 owner/repo[@skill]，收到: %q", pkg)
	}
	owner, repo = parts[0], parts[1]
	if ok, _ := regexp.MatchString(`^[A-Za-z0-9_.-]+$`, repo); !ok {
		return "", "", "", fmt.Errorf("仓库名不合法: %q", repo)
	}
	return owner, repo, skill, nil
}

func cmdAdd(args []string, flags *cliFlags, home, cwd string) error {
	if len(args) < 1 {
		return fmt.Errorf("用法: sk add owner/repo[@skill] [-a agents] [--project]")
	}
	owner, repo, skillFilter, err := parsePackage(args[0])
	if err != nil {
		return err
	}
	targets, err := resolveAgents(flags.agents, home)
	if err != nil {
		return err
	}
	if flags.project {
		// 只保留支持项目级目录的 agent
		var ok []Agent
		for _, a := range targets {
			if a.ProjectDir != "" {
				ok = append(ok, a)
			}
		}
		targets = ok
		if len(targets) == 0 {
			return fmt.Errorf("--project 模式下没有支持的 agent（仅 claude / gemini 支持项目级）")
		}
	}

	fmt.Printf("⬇️  拉取 %s/%s ...\n", owner, repo)
	tarPath, err := downloadTarball(owner, repo)
	if err != nil {
		return err
	}
	defer os.Remove(tarPath)
	tmpDir, err := os.MkdirTemp("", "skillcli-x-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	root, err := extractTarball(tarPath, tmpDir)
	if err != nil {
		return err
	}
	skills, err := findSkills(filepath.Join(tmpDir, root))
	if err != nil {
		return err
	}
	if skillFilter != "" {
		var matched []skillSource
		for _, s := range skills {
			if s.name == skillFilter {
				matched = append(matched, s)
			}
		}
		skills = matched
		if len(skills) == 0 {
			return fmt.Errorf("仓库里没找到技能 %q（可用: %s）", skillFilter, skillNames(skills))
		}
	}
	if len(skills) == 0 {
		return fmt.Errorf("仓库里没找到任何 SKILL.md")
	}

	for _, s := range skills {
		for _, a := range targets {
			dir := a.GlobalDir
			if flags.project {
				dir = filepath.Join(cwd, a.ProjectDir)
			}
			dest := filepath.Join(dir, s.name)
			if _, err := os.Stat(dest); err == nil {
				fmt.Printf("⏭️  %s → %s：已存在，跳过（先 sk remove %s 再装）\n", a.Name, dest, s.name)
				continue
			}
			if err := copyDir(s.dir, dest); err != nil {
				fmt.Printf("⚠️  %s → %s：%v\n", a.Name, dest, err)
				continue
			}
			fmt.Printf("✅ %s → %s\n", a.Name, dest)
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func skillNames(skills []skillSource) string {
	var names []string
	for _, s := range skills {
		names = append(names, s.name)
	}
	return strings.Join(names, ", ")
}

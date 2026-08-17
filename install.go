package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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

// cmdAdd 安装入口：支持 owner/repo[@skill] 或注册表按名字（sk install <技能名>）。
func cmdAdd(args []string, flags *cliFlags, home, cwd string) error {
	if len(args) < 1 {
		return fmt.Errorf("用法: sk add owner/repo[@skill] 或 sk install <技能名>")
	}
	arg := strings.TrimSpace(args[0])

	var (
		owner, repo, skillFilter string
		source                   string
	)
	if strings.Contains(arg, "/") {
		// owner/repo[@skill] 直接安装
		var err error
		owner, repo, skillFilter, err = parsePackage(arg)
		if err != nil {
			return err
		}
		source = "repo"
	} else {
		// 按名字走注册表解析
		rs, err := resolveSkill(arg)
		if err != nil {
			return err
		}
		owner, repo, skillFilter = rs.Owner, rs.Repo, rs.Skill
		source = "registry"
		if rs.Official {
			fmt.Printf("🔵 官方技能 %s（评分 %s）\n", rs.Name, rs.Level)
		}
		fmt.Printf("🔎 解析到 %s/%s%s（★%d，评分 %s）\n",
			rs.Owner, rs.Repo, skillSuffix(rs.Skill), rs.Stars, rs.Level)
	}

	installed, err := installFromSource(owner, repo, skillFilter, flags, home, cwd, false)
	if err != nil {
		return err
	}
	if len(installed) == 0 {
		return nil
	}

	// 记录 lockfile（每个装成功的技能一条）
	for _, name := range installed {
		id := fmt.Sprintf("%s/%s/%s", owner, repo, name)
		if err := upsertLock(home, lockEntry{
			ID: id, Owner: owner, Repo: repo, Skill: name,
			Source: source, InstalledAt: time.Now().Format(time.RFC3339),
		}); err != nil {
			fmt.Printf("⚠️  lockfile 写入失败: %v\n", err)
		}
		reportInstall(id)
	}
	fmt.Printf("✅ 安装完成：%s 已装到 %d 个技能\n", owner+"/"+repo, len(installed))
	return nil
}

func skillSuffix(skill string) string {
	if skill == "" {
		return ""
	}
	return " · " + skill
}

// installFromSource 从仓库下载并安装（replace=true 时替换已存在目录，用于 update）。
// 返回实际装成功/更新的技能目录名列表（去重）。
func installFromSource(owner, repo, skillFilter string, flags *cliFlags, home, cwd string, replace bool) ([]string, error) {
	targets, err := resolveAgents(flags.agents, home)
	if err != nil {
		return nil, err
	}
	if flags.project {
		var ok []Agent
		for _, a := range targets {
			if a.ProjectDir != "" {
				ok = append(ok, a)
			}
		}
		targets = ok
		if len(targets) == 0 {
			return nil, fmt.Errorf("--project 模式下没有支持的 agent（仅 claude / gemini 支持项目级）")
		}
	}

	fmt.Printf("⬇️  拉取 %s/%s ...\n", owner, repo)
	tarPath, err := downloadTarball(owner, repo)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tarPath)
	tmpDir, err := os.MkdirTemp("", "skillcli-x-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	root, err := extractTarball(tarPath, tmpDir)
	if err != nil {
		return nil, err
	}
	skills, err := findSkills(filepath.Join(tmpDir, root))
	if err != nil {
		return nil, err
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
			return nil, fmt.Errorf("仓库里没找到技能 %q（可用: %s）", skillFilter, skillNames(skills))
		}
	}
	if len(skills) == 0 {
		return nil, fmt.Errorf("仓库里没找到任何 SKILL.md")
	}

	installed := make(map[string]bool)
	for _, s := range skills {
		for _, a := range targets {
			dir := a.GlobalDir
			if flags.project {
				dir = filepath.Join(cwd, a.ProjectDir)
			}
			dest := filepath.Join(dir, s.name)
			md := filepath.Join(dest, "SKILL.md")
			if _, err := os.Stat(md); err == nil {
				if !replace {
					fmt.Printf("⏭️  %s → %s：已存在，跳过（先 sk remove %s 再装）\n", a.Name, dest, s.name)
					continue
				}
				// update 模式：替换
				if err := os.RemoveAll(dest); err != nil {
					fmt.Printf("⚠️  %s → %s：删除旧版失败 %v\n", a.Name, dest, err)
					continue
				}
				if err := copyDir(s.dir, dest); err != nil {
					fmt.Printf("⚠️  %s → %s：%v\n", a.Name, dest, err)
					continue
				}
				fmt.Printf("🔄 %s → %s 已更新\n", a.Name, dest)
			} else {
				if err := copyDir(s.dir, dest); err != nil {
					fmt.Printf("⚠️  %s → %s：%v\n", a.Name, dest, err)
					continue
				}
				fmt.Printf("✅ %s → %s\n", a.Name, dest)
			}
			installed[s.name] = true
		}
	}
	out := make([]string, 0, len(installed))
	for name := range installed {
		out = append(out, name)
	}
	return out, nil
}

// reportInstall 向注册表上报一次安装（fire-and-forget，失败静默）。
func reportInstall(id string) {
	body, _ := json.Marshal(map[string]string{"id": id})
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(registryBase()+"/api/counts", "application/json", bytes.NewReader(body))
	if err != nil {
		return
	}
	resp.Body.Close()
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

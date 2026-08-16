package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// skillSource 仓库中一个技能：目录名 + 解压根路径。
type skillSource struct {
	name string // 技能目录名（安装到 <agentDir>/<name>/）
	dir  string // 解压后该技能的完整路径
}

// downloadTarball 下载仓库 tarball 到临时文件并返回路径。
// 先试默认分支 main，404 则回退 master。
func downloadTarball(owner, repo string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	for _, ref := range []string{"main", "master"} {
		url := fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz/refs/heads/%s", owner, repo, ref)
		resp, err := client.Get(url)
		if err != nil {
			return "", fmt.Errorf("下载失败: %w", err)
		}
		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return "", fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
		}
		tmp, err := os.CreateTemp("", "skillcli-*.tar.gz")
		if err != nil {
			resp.Body.Close()
			return "", err
		}
		// 临时文件由调用方在使用后清理（defer 在这里会提前删掉，导致后续解压读不到）
		if _, err := io.Copy(tmp, resp.Body); err != nil {
			resp.Body.Close()
			tmp.Close()
			return "", err
		}
		resp.Body.Close()
		tmp.Close()
		return tmp.Name(), nil
	}
	return "", fmt.Errorf("仓库 %s/%s 不存在（main/master 都找不到）", owner, repo)
}

// extractTarball 解压到目标目录，返回解压根目录名（如 claude-skills-main）。
func extractTarball(tarPath, dest string) (string, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	root := ""
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		// 跳过 tar 元数据条目（PAX 全局头等），避免被当成目录根
		if hdr.Typeflag == tar.TypeXGlobalHeader ||
			hdr.Typeflag == tar.TypeXHeader ||
			strings.HasPrefix(hdr.Name, "pax_global_header") {
			continue
		}
		name := strings.TrimPrefix(hdr.Name, "./")
		if i := strings.IndexByte(name, '/'); i >= 0 {
			if root == "" {
				root = name[:i]
			}
		} else if root == "" && name != "" {
			root = name
		}
		target := filepath.Join(dest, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return "", err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return "", err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return "", err
			}
			out.Close()
		}
	}
	if root == "" {
		return "", errors.New("tarball 为空")
	}
	return root, nil
}

// findSkills 在解压根目录下找所有含 SKILL.md 的技能目录。
// 仓库根目录本身有 SKILL.md 时，整个仓库就是一个技能。
func findSkills(rootDir string) ([]skillSource, error) {
	rootMd := filepath.Join(rootDir, "SKILL.md")
	if info, err := os.Stat(rootMd); err == nil && !info.IsDir() {
		return []skillSource{{name: filepath.Base(rootDir), dir: rootDir}}, nil
	}
	var out []skillSource
	err := filepath.WalkDir(rootDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "SKILL.md" {
			return nil
		}
		skillDir := filepath.Dir(p)
		if skillDir == rootDir {
			return nil // 根级已单独处理
		}
		out = append(out, skillSource{name: filepath.Base(skillDir), dir: skillDir})
		return nil
	})
	return out, err
}

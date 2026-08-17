package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// lockEntry 一条已装技能记录（全局安装；项目级安装 M1 暂不记录）。
type lockEntry struct {
	ID          string `json:"id"` // owner/repo/技能名
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Skill       string `json:"skill,omitempty"` // 技能目录名；空 = 整仓
	Source      string `json:"source"`          // registry | repo
	InstalledAt string `json:"installedAt"`
}

type lockfile struct {
	Version int         `json:"version"`
	Entries []lockEntry `json:"entries"`
}

func lockPath(home string) string {
	return filepath.Join(home, ".config", "skillcli", "sk.lock")
}

func readLock(home string) (*lockfile, error) {
	lf := &lockfile{Version: 1}
	data, err := os.ReadFile(lockPath(home))
	if err != nil {
		if os.IsNotExist(err) {
			return lf, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, lf); err != nil {
		return nil, fmt.Errorf("lockfile 解析失败（%s）: %w", lockPath(home), err)
	}
	return lf, nil
}

func writeLock(home string, lf *lockfile) error {
	dir := filepath.Dir(lockPath(home))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(lockPath(home), data, 0o644)
}

// upsertLock 新增或更新一条记录。
func upsertLock(home string, e lockEntry) error {
	lf, err := readLock(home)
	if err != nil {
		return err
	}
	for i := range lf.Entries {
		if lf.Entries[i].ID == e.ID {
			lf.Entries[i] = e
			return writeLock(home, lf)
		}
	}
	lf.Entries = append(lf.Entries, e)
	return writeLock(home, lf)
}

// removeFromLock 删除记录（卸载时调用）。
// 匹配方式：完整 id（owner/repo/name）或技能名（目录名）均可。
func removeFromLock(home, idOrName string) error {
	lf, err := readLock(home)
	if err != nil {
		return err
	}
	out := lf.Entries[:0]
	changed := false
	for _, e := range lf.Entries {
		if e.ID == idOrName || e.Skill == idOrName {
			changed = true
			continue
		}
		out = append(out, e)
	}
	if !changed {
		return nil // 没有变化
	}
	lf.Entries = out
	return writeLock(home, lf)
}

// cmdUpdate 基于 lockfile 逐个重新拉取仓库默认分支并替换已装技能。
func cmdUpdate(args []string, flags *cliFlags, home, cwd string) error {
	lf, err := readLock(home)
	if err != nil {
		return err
	}
	if len(lf.Entries) == 0 {
		fmt.Println("📭 lockfile 为空，没有可更新的技能（先 sk install / sk add 安装）")
		return nil
	}
	updated := 0
	for _, e := range lf.Entries {
		fmt.Printf("⬆️  更新 %s ...\n", e.ID)
		installed, err := installFromSource(e.Owner, e.Repo, e.Skill, flags, home, cwd, true)
		if err != nil {
			fmt.Printf("  ⚠️  %v\n", err)
			continue
		}
		if len(installed) > 0 {
			e.InstalledAt = time.Now().Format(time.RFC3339)
			if err := upsertLock(home, e); err != nil {
				fmt.Printf("  ⚠️  lockfile 写入失败: %v\n", err)
			}
			updated++
		}
	}
	fmt.Printf("✅ 更新完成：%d/%d 个技能已更新（%s）\n", updated, len(lf.Entries), lockPath(home))
	return nil
}

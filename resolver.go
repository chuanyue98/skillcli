package main

import (
	"fmt"
	"sort"
	"strings"
)

// resolvedSkill 名字解析结果：定位到来源仓库 + 技能目录。
type resolvedSkill struct {
	ID       string // owner/repo/name（注册表 id）
	Owner    string
	Repo     string
	Skill    string // 技能目录名；空 = 整个仓库就是一个技能
	Name     string
	Desc     string
	Level    string
	Score    int
	Stars    int
	Official bool
}

// resolveSkill 按名字从注册表解析最佳匹配。
// 策略：精确匹配名字优先 → 候选按 官方 > 评分 > 星数 排序取最优。
func resolveSkill(name string) (*resolvedSkill, error) {
	items, err := searchRegistry(name, 50)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("注册表里没找到技能 %q（试试 sk search %s）", name, name)
	}

	var cands []registryItem
	for _, it := range items {
		if it.Name == name {
			cands = append(cands, it)
		}
	}
	if len(cands) == 0 {
		cands = items
	}
	if len(cands) > 1 {
		sort.SliceStable(cands, func(i, j int) bool {
			if cands[i].Official != cands[j].Official {
				return cands[i].Official
			}
			if cands[i].Score.Total != cands[j].Score.Total {
				return cands[i].Score.Total > cands[j].Score.Total
			}
			return cands[i].Repo.Stars > cands[j].Repo.Stars
		})
	}

	best := cands[0]
	owner, repo, ok := strings.Cut(best.Repo.FullName, "/")
	if !ok {
		return nil, fmt.Errorf("注册表数据异常：仓库名 %q", best.Repo.FullName)
	}
	return &resolvedSkill{
		ID:       best.ID,
		Owner:    owner,
		Repo:     repo,
		Skill:    skillDirFromPath(best.Path),
		Name:     best.Name,
		Desc:     best.Description,
		Level:    best.Score.Level,
		Score:    best.Score.Total,
		Stars:    best.Repo.Stars,
		Official: best.Official,
	}, nil
}

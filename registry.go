package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// registryBase 注册表地址：默认 SkillHub 站，可用环境变量覆盖（为私有源铺路）。
func registryBase() string {
	if v := os.Getenv("SKILLCLI_REGISTRY"); v != "" {
		return strings.TrimSuffix(v, "/")
	}
	return "https://skillhub-ai.vercel.app"
}

// registryItem 注册表索引项（对应站端 /api/skills 的条目子集）。
type registryItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Path        string   `json:"path"`
	Tags        []string `json:"tags"`
	Version     string   `json:"version"`
	Category    string   `json:"category"`
	Official    bool     `json:"official"`
	Score       struct {
		Total int    `json:"total"`
		Level string `json:"level"`
	} `json:"score"`
	Repo struct {
		FullName string `json:"fullName"`
		Stars    int    `json:"stars"`
	} `json:"repo"`
}

type registryResponse struct {
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int            `json:"totalPages"`
	Items      []registryItem `json:"items"`
}

// registryGet 请求注册表 API（GET，自动拼接 base）。
func registryGet(path string, params url.Values) (*registryResponse, error) {
	u := registryBase() + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("注册表请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("注册表返回 HTTP %d（%s）", resp.StatusCode, registryBase()+path)
	}
	var out registryResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("注册表响应解析失败: %w", err)
	}
	return &out, nil
}

// registryFields 注册表条目需要的字段子集（跳过 body 等大字段，省流量）。
const registryFields = "id,name,description,path,tags,version,category,official,score.total,score.level,repo.fullName,repo.stars"

// searchRegistry 搜索注册表，按评分降序。
func searchRegistry(query string, limit int) ([]registryItem, error) {
	p := url.Values{}
	p.Set("q", query)
	p.Set("sort", "score")
	p.Set("fields", registryFields)
	if limit > 0 {
		p.Set("limit", strconv.Itoa(limit))
	}
	resp, err := registryGet("/api/skills", p)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// skillDirFromPath 从 SKILL.md 路径推导技能目录名；根级技能返回 ""（=整个仓库就是一个技能）。
func skillDirFromPath(p string) string {
	p = strings.Trim(p, "/")
	if p == "" || p == "SKILL.md" {
		return ""
	}
	parts := strings.Split(p, "/")
	if parts[len(parts)-1] == "SKILL.md" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// cmdSearch 注册表搜索。
func cmdSearch(args []string, flags *cliFlags) error {
	if len(args) < 1 {
		return fmt.Errorf("用法: sk search <关键词> [--limit N] [--json]")
	}
	items, err := searchRegistry(args[0], flags.limit)
	if err != nil {
		return err
	}
	if flags.json {
		b, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	if len(items) == 0 {
		fmt.Printf("📭 注册表里没找到与 %q 相关的技能\n", args[0])
		return nil
	}
	fmt.Printf("%-30s %-6s %-9s %-14s %s\n", "名称", "评分", "星数", "分类", "描述")
	for _, it := range items {
		stars := fmt.Sprintf("%d", it.Repo.Stars)
		if it.Repo.Stars >= 1000 {
			stars = fmt.Sprintf("%.1fk", float64(it.Repo.Stars)/1000)
		}
		score := it.Score.Level
		if it.Score.Total > 0 {
			score = fmt.Sprintf("%s(%d)", it.Score.Level, it.Score.Total)
		}
		desc := truncate(it.Description, 44)
		mark := ""
		if it.Official {
			mark = "✓"
		}
		fmt.Printf("%-30s %-6s %-9s %-14s %s%s\n",
			it.Name+mark, score, stars, it.Category, desc, "")
	}
	return nil
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

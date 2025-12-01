package utils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveProgramPaths 将输入解析为可执行文件的绝对路径列表
// 支持：
// 1. 绝对路径文件 -> 返回单个路径
// 2. 绝对路径目录 -> 返回目录下所有exe和dll
// 3. 程序名称（PATH查找） -> 返回单个路径
// 4. 程序名称（模糊查找） -> 返回单个路径
func ResolveProgramPaths(input string) ([]string, error) {
	in := strings.TrimSpace(input)
	if in == "" {
		return nil, errors.New("程序名称为空")
	}

	fmt.Printf("正在查找程序路径: %s ...\n", in)

	// 1. 检查是否为有效路径（文件或目录）
	if isPathLike(in) {
		abs := in
		if !filepath.IsAbs(abs) {
			a, _ := filepath.Abs(abs)
			abs = a
		}
		fi, err := os.Stat(abs)
		if err == nil {
			// 如果是目录，递归查找所有exe和dll
			if fi.IsDir() {
				return findAllExecutablesInDir(abs)
			}
			// 如果是文件，直接返回
			return []string{abs}, nil
		}
	}

	// 2. 尝试在PATH中使用where命令查找精确匹配
	if p, err := whereFind(in); err == nil && p != "" {
		return []string{p}, nil
	}

	// 3. 在常见目录中模糊查找
	p, err := findExecutableByName(in)
	if err != nil {
		return nil, err
	}
	if p == "" {
		return nil, fmt.Errorf("无法通过名称找到程序: %s", in)
	}
	return []string{p}, nil
}

// findAllExecutablesInDir 遍历目录查找所有exe和dll文件
func findAllExecutablesInDir(dir string) ([]string, error) {
	var results []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // 忽略访问错误
		}
		if d.IsDir() {
			return nil
		}
		lowerPath := strings.ToLower(path)
		if strings.HasSuffix(lowerPath, ".exe") || strings.HasSuffix(lowerPath, ".dll") {
			results = append(results, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("在目录中未找到可执行文件: %s", dir)
	}
	return results, nil
}

// isPathLike 判断字符串是否像路径或以 .exe 结尾
func isPathLike(s string) bool {
	return strings.Contains(s, "\\") || strings.Contains(s, "/") || strings.HasSuffix(strings.ToLower(s), ".exe")
}

// whereFind 使用 Windows where 命令在 PATH 中查找可执行文件
func whereFind(name string) (string, error) {
	candidates := []string{name, name + ".exe"}
	for _, c := range candidates {
		cmd := exec.Command("where", c)
		out, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(lines) > 0 {
				p := strings.TrimSpace(lines[0])
				if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
					return p, nil
				}
			}
		}
	}
	return "", fmt.Errorf("在PATH中未找到: %s", name)
}

// findExecutableByName 在常见目录中模糊匹配查找包含片段的 .exe
func findExecutableByName(part string) (string, error) {
	partLower := strings.ToLower(part)
	roots := collectSearchRoots()
	maxMatches := 5
	found := ""

	// 遍历每个搜索根目录
	for _, r := range roots {
		_ = filepath.WalkDir(r, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(path), ".exe") {
				base := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
				if strings.Contains(base, partLower) {
					found = path
					maxMatches--
					if maxMatches <= 0 || found != "" {
						return errors.New("stop")
					}
				}
			}
			return nil
		})
		if found != "" {
			break
		}
	}
	return found, nil
}

// collectSearchRoots 收集用于搜索的目录集合，包括 PATH 和常见安装目录
func collectSearchRoots() []string {
	roots := []string{}
	// PATH 环境变量
	for _, p := range strings.Split(os.Getenv("PATH"), ";") {
		p = strings.TrimSpace(p)
		if p != "" {
			roots = append(roots, p)
		}
	}
	// 常见程序安装位置
	common := []string{
		`C:\\Program Files`,
		`C:\\Program Files (x86)`,
		`C:\\ProgramData`,
		filepath.Join(os.Getenv("USERPROFILE"), `AppData\\Local\\Programs`),
	}
	for _, c := range common {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			roots = append(roots, c)
		}
	}
	return roots
}

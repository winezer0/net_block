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
// 增加了对所有驱动器中常见程序目录的扫描
func collectSearchRoots() []string {
	roots := []string{}
	// PATH 环境变量
	for _, p := range strings.Split(os.Getenv("PATH"), ";") {
		p = strings.TrimSpace(p)
		if p != "" {
			roots = append(roots, p)
		}
	}

	// 扫描所有可用驱动器
	drives := getAvailableDrives()
	for _, drive := range drives {
		// 预定义的常见目录名称
		commonNames := []string{
			"Program Files",
			"Program Files (x86)",
			"ProgramData",
			"Games",
			"Software",
			"Apps",
			"Programs",
			"Program",
			"Progreen",
		}

		// 1. 添加预定义的常见目录
		for _, name := range commonNames {
			path := filepath.Join(drive, name)
			if fi, err := os.Stat(path); err == nil && fi.IsDir() {
				roots = append(roots, path)
			}
		}

		// 2. 扫描驱动器根目录，查找包含 Program, Game, App 等关键字的目录
		if entries, err := os.ReadDir(drive); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					nameLower := strings.ToLower(entry.Name())
					if strings.Contains(nameLower, "program") ||
						strings.Contains(nameLower, "game") ||
						strings.Contains(nameLower, "app") ||
						strings.Contains(nameLower, "soft") {
						// 避免重复添加（如果已经在 commonNames 中涵盖）
						path := filepath.Join(drive, entry.Name())
						alreadyAdded := false
						for _, r := range roots {
							if strings.EqualFold(r, path) {
								alreadyAdded = true
								break
							}
						}
						if !alreadyAdded {
							roots = append(roots, path)
						}
					}
				}
			}
		}
	}

	// 用户特定目录 (通常在 C 盘，但也可能被重定向)
	userPrograms := filepath.Join(os.Getenv("USERPROFILE"), `AppData\Local\Programs`)
	if fi, err := os.Stat(userPrograms); err == nil && fi.IsDir() {
		roots = append(roots, userPrograms)
	}

	return roots
}

// getAvailableDrives 获取系统中所有可用的逻辑驱动器根路径 (例如 "C:\", "D:\")
func getAvailableDrives() []string {
	var drives []string
	for _, drive := range "CDEFGHIJKLMNOPQRSTUVWXYZ" {
		path := string(drive) + ":\\"
		if _, err := os.Stat(path); err == nil {
			drives = append(drives, path)
		}
	}
	return drives
}

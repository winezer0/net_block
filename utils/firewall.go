package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// AddFirewallRules 为指定程序添加入站与出站的阻断规则
// ruleBase 为规则名称前缀，programPath 为程序绝对路径
func AddFirewallRules(programPath string) error {
	ruleName := generateRuleName(programPath)

	// 添加出站规则
	if err := runNetsh([]string{"advfirewall", "firewall", "add", "rule",
		fmt.Sprintf("name=%q", ruleName), "dir=out",
		fmt.Sprintf("program=%q", programPath), "action=block", "enable=yes"}); err != nil {
		return err
	}

	// 添加入站规则
	inName := ruleName + "_In"
	if err := runNetsh([]string{"advfirewall", "firewall", "add", "rule",
		fmt.Sprintf("name=%q", inName), "dir=in",
		fmt.Sprintf("program=%q", programPath), "action=block", "enable=yes"}); err != nil {
		return err
	}
	return nil
}

// DeleteFirewallRules 删除指定程序的入站与出站阻断规则
func DeleteFirewallRules(programPath string) error {
	ruleName := generateRuleName(programPath)

	// 删除出站规则
	_ = runNetsh([]string{"advfirewall", "firewall", "delete", "rule",
		fmt.Sprintf("name=%q", ruleName), "dir=out",
		fmt.Sprintf("program=%q", programPath)})

	// 删除入站规则
	inName := ruleName + "_In"
	_ = runNetsh([]string{"advfirewall", "firewall", "delete", "rule",
		fmt.Sprintf("name=%q", inName), "dir=in",
		fmt.Sprintf("program=%q", programPath)})
	return nil
}

// CheckFirewallStatus 检查指定程序的入站与出站阻断规则存在性
// 返回状态：BLOCKED（已阻止）、ALLOWED（未阻止/允许）、PARTIAL（部分阻止）
func CheckFirewallStatus(programPath string) (string, error) {
	ruleName := generateRuleName(programPath)

	outExists, err := ruleExists(ruleName, "out", programPath)
	if err != nil {
		return "", err
	}
	inExists, err := ruleExists(ruleName+"_In", "in", programPath)
	if err != nil {
		return "", err
	}
	if outExists && inExists {
		return "BLOCKED", nil
	}
	if !outExists && !inExists {
		return "ALLOWED", nil
	}
	return "PARTIAL", nil
}

// ruleExists 通过 netsh show rule 检查指定名称与方向的规则是否存在且匹配程序路径
func ruleExists(name, dir, programPath string) (bool, error) {
	args := []string{"advfirewall", "firewall", "show", "rule", fmt.Sprintf("name=%q", name), fmt.Sprintf("dir=%s", dir)}
	cmd := exec.Command("netsh", args...)
	out, err := cmd.CombinedOutput()
	output := convertGBKToUTF8(out)
	text := strings.ToLower(output)

	if err != nil {
		// 无匹配规则时返回特定提示，视为不存在而非错误
		if strings.Contains(text, "no rules match") || strings.Contains(text, "没有与指定标准相匹配的规则") {
			return false, nil
		}
		return false, fmt.Errorf("netsh error: %v, output: %s", err, strings.TrimSpace(output))
	}

	if strings.Contains(text, "no rules match") || strings.Contains(text, "没有与指定标准相匹配的规则") {
		return false, nil
	}

	// 匹配程序路径（不区分大小写）
	if strings.Contains(text, strings.ToLower(programPath)) {
		return true, nil
	}
	return false, nil
}

// runNetsh 执行 netsh 命令并返回错误信息
func runNetsh(args []string) error {
	cmd := exec.Command("netsh", args...)
	out, err := cmd.CombinedOutput()
	output := convertGBKToUTF8(out)
	if err != nil {
		return fmt.Errorf("netsh error: %v, output: %s", err, strings.TrimSpace(output))
	}
	return nil
}

// generateRuleName 根据程序路径生成唯一的规则名称
// 格式：BlockProgram_<BaseName>_<PathHash>
func generateRuleName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	nameNoExt := strings.TrimSuffix(base, ext)

	// 计算路径的MD5哈希以确保唯一性
	hash := md5.Sum([]byte(strings.ToLower(path)))
	hashStr := hex.EncodeToString(hash[:])[:8] // 取前8位即可

	return fmt.Sprintf("BlockProgram_%s_%s", nameNoExt, hashStr)
}

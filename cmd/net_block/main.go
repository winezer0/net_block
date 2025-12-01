package main

import (
	"cut_app_net/utils"
	"fmt"
	"os"

	flags "github.com/jessevdk/go-flags"
)

// Options 定义命令行参数结构
type Options struct {
	Programs []string `short:"p" long:"program" description:"程序完整路径或部分名称 (支持多次指定)" required:"true"`
	Mode     int      `short:"m" long:"mode" description:"1=允许访问, 2=阻止访问, 3=检查状态" required:"true"`
}

// main 程序入口函数
func main() {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		// 处理帮助信息请求
		if fe, ok := err.(*flags.Error); ok && fe.Type == flags.ErrHelp {
			os.Exit(0)
		}
		fmt.Println("参数解析失败:", err)
		os.Exit(1)
	}

	// 遍历所有输入的程序参数
	for _, inputProg := range opts.Programs {
		fmt.Println("--------------------------------------------------")
		fmt.Printf("处理输入: %s\n", inputProg)

		// 解析路径（可能返回多个，例如输入目录时）
		targetPaths, err := utils.ResolveProgramPaths(inputProg)
		if err != nil {
			fmt.Printf("错误: 解析程序失败 '%s': %v\n", inputProg, err)
			continue
		}

		// 对解析出的每个实际路径执行操作
		for _, progPath := range targetPaths {
			fmt.Printf("目标程序: %s\n", progPath)
			if err := processProgram(progPath, opts.Mode); err != nil {
				fmt.Printf("操作失败: %v\n", err)
			}
		}
	}
	fmt.Println("--------------------------------------------------")
	fmt.Println("所有操作完成。")
}

// processProgram 根据模式对单个程序执行操作
func processProgram(progPath string, mode int) error {
	switch mode {
	case 1: // 允许访问
		if err := utils.DeleteFirewallRules(progPath); err != nil {
			return fmt.Errorf("移除规则失败: %v", err)
		}
		fmt.Println("成功: 已移除防火墙规则")
	case 2: // 阻止访问
		if err := utils.AddFirewallRules(progPath); err != nil {
			return fmt.Errorf("添加规则失败: %v", err)
		}
		fmt.Println("成功: 已添加防火墙阻止规则")
	case 3: // 检查状态
		status, err := utils.CheckFirewallStatus(progPath)
		if err != nil {
			return fmt.Errorf("检查状态失败: %v", err)
		}
		fmt.Printf("状态: %s\n", translateStatus(status))
	default:
		return fmt.Errorf("无效模式: %d", mode)
	}
	return nil
}

// translateStatus 将状态代码转换为中文描述
func translateStatus(status string) string {
	switch status {
	case "BLOCKED":
		return "已阻止 (BLOCKED)"
	case "ALLOWED":
		return "允许访问 (ALLOWED)"
	case "PARTIAL":
		return "部分阻止 (PARTIAL)"
	default:
		return status
	}
}

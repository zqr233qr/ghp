package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"ghp/pkg/ai"
	"ghp/pkg/config"
	"ghp/pkg/executor"
)

var (
	useStream    bool
	useConcise   bool
	forceMode    bool
	analyzeMode  bool
	generateMode bool
)

var rootCmd = &cobra.Command{
	Use:   "ghp [command] [subcommand...]",
	Short: "AI powered CLI helper",
	Long:  `ghp is a CLI tool that uses AI to explain commands and provide usage examples.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 0. 参数互斥检查
		// 解析模式和生成模式互斥
		if analyzeMode && generateMode {
			fmt.Println("错误: 无法同时使用解析模式 (-a) 和生成模式 (-g)。请只选择一种操作。")
			return
		}
		// 强制模式不能与解析或生成模式混用（因为这两种模式依赖准确的帮助文档上下文）
		if forceMode && (analyzeMode || generateMode) {
			fmt.Println("错误: 强制模式 (-f) 仅适用于普通查询，不能与解析 (-a) 或生成 (-g) 模式混用。")
			return
		}

		program := args[0]
		var subQuery string
		if len(args) > 1 {
			subQuery = strings.Join(args[1:], " ")
		}

		ctx, cancel := context.WithCancel(context.Background())
		go gracefulShutdown(cancel)

		// 1. 加载配置
		cfg, err := config.Load()
		if err != nil {
			fmt.Println(err)
			return
		}

		// 2. 初始化 AI 客户端
		aiClient := ai.NewClient(cfg.NewClientConfig(), cfg.Model)

		// 3. 检查命令是否存在
		cmdPath, err := executor.CheckCommandExists(program)
		isMissing := false

		if err != nil {
			// 如果是分析模式或生成模式，必须要求命令存在
			if analyzeMode || generateMode {
				fmt.Println(err)
				fmt.Println("错误: 无法处理未安装的命令。我们需要本地帮助文档来确保解释/生成的准确性。")
				return
			}
			if !forceMode {
				fmt.Println(err)
				fmt.Println("提示: 使用 -f 或 --force 参数可以强制查询未安装的命令")
				return
			}
			isMissing = true
			cmdPath = "该命令尚未安装"
		}

		var helpOutput, verOutput string
		usedCmd := program

		// 4-6. 仅在命令存在时执行获取帮助逻辑
		if !isMissing {
			// 4. 获取查询指令 (Help & Version)
			helpCmdArgs, verCmdArgs, err := aiClient.GetHelpCommand(ctx, program)
			if err != nil {
				fmt.Println("获取查询指令失败:", err)
				return
			}

			// 5. 执行帮助命令
			hOut, hUsed, success := executor.RunCommandWithRetry(
				ctx, helpCmdArgs, [][]string{{"--help"}, {"-h"}, {"help"}}, program,
			)
			if !success {
				// 如果开启了强制模式，即使运行失败也尝试降级处理
				// 这对于 Windows 上存在的 Store Redirector (空壳 exe) 很有用
				if forceMode {
					fmt.Println("警告: 无法获取命令帮助文档，将转为强制模式进行查询。")
					isMissing = true
					cmdPath = "检测到命令但无法运行"
				} else {
					fmt.Println("无法获取命令帮助文档。已尝试 AI 推荐指令及标准参数。")
					fmt.Println("提示: 命令可能无法正常运行（如 Windows 应用商店别名），请尝试使用 -f 或 --force 参数强制查询。")
					return
				}
			} else {
				helpOutput = hOut
				usedCmd = hUsed
			}

			// 6. 执行版本命令 (仅精简模式需尝试，且不在分析/生成模式下)
			if useConcise && !analyzeMode && !generateMode {
				out, _, success := executor.RunCommandWithRetry(
					ctx, verCmdArgs, [][]string{{"--version"}, {"-v"}, {"version"}}, program,
				)
				if success {
					verOutput = out
				} else {
					verOutput = "无法获取版本信息"
				}
			}
		}

		// 分支：命令分析模式
		if analyzeMode {
			// 使用 reconstructArgs 为包含空格的参数添加引号，防止 AI 解析错误
			fullCommand := reconstructArgs(args)
			if err := aiClient.ExplainCommand(ctx, useStream, fullCommand, helpOutput, cmdPath); err != nil {
				fmt.Println("AI 解析失败:", err)
			}
			return
		}

		// 分支：命令生成模式
		if generateMode {
			description := subQuery
			if description == "" {
				fmt.Println("错误: 生成模式需要提供自然语言描述 (例如: ghp -g git 设置全局用户名)")
				return
			}
			if err := aiClient.GenerateCommand(ctx, useStream, program, description, helpOutput, cmdPath); err != nil {
				fmt.Println("AI 生成失败:", err)
			}
			return
		}

		// 7. 常规 AI 分析并输出 (支持未安装模式)
		if err := aiClient.AnalyzeHelpDoc(ctx, useStream, useConcise, isMissing, subQuery, usedCmd, helpOutput, verOutput, cmdPath); err != nil {
			fmt.Println("AI 分析失败:", err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// reconstructArgs 重组参数，为包含空格的参数添加引号
// 例如: ["git", "commit", "-m", "fix bug"] -> "git commit -m \"fix bug\""
func reconstructArgs(args []string) string {
	var parts []string
	for _, arg := range args {
		if strings.Contains(arg, " ") || strings.Contains(arg, "\t") {
			parts = append(parts, fmt.Sprintf("%q", arg))
		} else {
			parts = append(parts, arg)
		}
	}
	return strings.Join(parts, " ")
}

func init() {
	rootCmd.Flags().BoolVarP(&useStream, "stream", "s", true, "是否使用流式输出")
	rootCmd.Flags().BoolVarP(&useConcise, "concise", "c", true, "是否精简输出")
	rootCmd.Flags().BoolVarP(&forceMode, "force", "f", false, "强制查询模式 (即使命令不存在也查询)")
	rootCmd.Flags().BoolVarP(&analyzeMode, "analyze", "a", false, "解析模式 (解释具体命令及参数含义)")
	rootCmd.Flags().BoolVarP(&generateMode, "generate", "g", false, "生成模式 (根据自然语言描述生成命令)")

	// 关键修复：禁用 Flag 穿插解析
	// 一旦遇到第一个非 Flag 参数（如 "go"），后续所有内容（包括 -v, --help 等）都将作为 Args 处理
	// 这对于像 `ghp go build -v` 这样的子命令透传至关重要
	rootCmd.Flags().SetInterspersed(false)
}

func gracefulShutdown(cancel context.CancelFunc) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
}

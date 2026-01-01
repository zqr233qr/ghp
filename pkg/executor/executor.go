package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var isExistCmdMap = map[string]string{
	"linux":   "which",
	"darwin":  "which",
	"windows": "where",
}

// CheckCommandExists 检查命令是否存在，返回命令位置或描述
func CheckCommandExists(cmdName string) (string, error) {
	osname := runtime.GOOS
	checkCmd := isExistCmdMap[osname]

	// 1. 尝试直接查找
	cmd := exec.Command(checkCmd, cmdName)
	output, err := cmd.Output()
	if err == nil {
		path := strings.TrimSpace(string(output))
		// windows where 可能返回多行，取第一行
		if osname == "windows" {
			lines := strings.Split(path, "\n")
			if len(lines) > 0 {
				path = strings.TrimSpace(lines[0])
			}
		}
		return path, nil
	}

	// 2. 如果直接查找失败，尝试通过 Shell 查找
	userShell := os.Getenv("SHELL")
	if userShell == "" {
		userShell = "/bin/bash"
	}

	// 使用 command -v 在交互式 shell 中检测
	shellCmd := exec.Command(userShell, "-i", "-c", "command -v "+cmdName)
	shellOut, shellErr := shellCmd.CombinedOutput()

	// 恢复终端
	if runtime.GOOS != "windows" {
		FixTerminal()
	}

	if shellErr == nil {
		path := strings.TrimSpace(string(shellOut))
		if path != "" {
			return path, nil
		}
		return "Shell Builtin/Alias", nil
	}

	return "", fmt.Errorf("命令不存在: %s", cmdName)
}

// RunCommandWithRetry 执行命令，支持重试、超时和 Shell 兜底
func RunCommandWithRetry(ctx context.Context, aiCmd []string, fallbackArgs [][]string, needHelp string) (string, string, bool) {
	type tryCmd struct {
		args     []string
		useShell bool
	}
	var tries []tryCmd
	seenCmds := make(map[string]bool)

	addTry := func(args []string, useShell bool) {
		cmdStr := strings.Join(args, " ")
		key := fmt.Sprintf("%v:%s", useShell, cmdStr) // 区分 shell 模式和直接执行
		if !seenCmds[key] {
			tries = append(tries, tryCmd{args: args, useShell: useShell})
			seenCmds[key] = true
		}
	}

	if len(aiCmd) > 0 {
		addTry(aiCmd, false)
	}
	for _, args := range fallbackArgs {
		addTry(append([]string{needHelp}, args...), false)
	}
	if len(aiCmd) > 0 {
		addTry(aiCmd, true)
	}
	addTry([]string{needHelp, "--help"}, true)

	userShell := os.Getenv("SHELL")
	if userShell == "" {
		userShell = "/bin/bash"
	}

	for _, try := range tries {
		if len(try.args) == 0 {
			continue
		}

		timeout := 3 * time.Second
		if try.useShell {
			timeout = 8 * time.Second
		}

		tCtx, cancel := context.WithTimeout(ctx, timeout)
		var cmd *exec.Cmd
		if try.useShell {
			cmd = exec.CommandContext(tCtx, userShell, "-i", "-c", strings.Join(try.args, " "))
		} else {
			cmd = exec.CommandContext(tCtx, try.args[0], try.args[1:]...)
		}

		out, err := cmd.CombinedOutput()
		cancel()

		if try.useShell && runtime.GOOS != "windows" {
			FixTerminal()
		}

		if tCtx.Err() == context.DeadlineExceeded {
			continue
		}

		outStr := string(out)
		if err == nil || (len(outStr) > 50 && !strings.Contains(strings.ToLower(outStr), "not found")) {
			return outStr, strings.Join(try.args, " "), true
		}
	}
	return "", "", false
}

// FixTerminal 恢复终端状态
func FixTerminal() {
	cmd := exec.Command("stty", "sane")
	cmd.Stdin = os.Stdin
	cmd.Run()
}

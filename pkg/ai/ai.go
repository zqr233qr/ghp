package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type Client struct {
	client *openai.Client
	model  string
}

func NewClient(cfg openai.ClientConfig, model string) *Client {
	return &Client{
		client: openai.NewClientWithConfig(cfg),
		model:  model,
	}
}

// GetHelpCommand 获取帮助和版本查询命令
func (c *Client) GetHelpCommand(ctx context.Context, program string) ([]string, []string, error) {
	osname := runtime.GOOS
	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: "你是一个命令行专家。请直接给出获取以下程序信息的**最佳命令**。\n\n" +
						"规则：\n" +
						"1. **输出两行**：\n" +
						"   - 第一行：获取帮助信息的命令 (如 git --help)\n" +
						"   - 第二行：获取版本信息的命令 (如 git --version)。如果该程序没有版本命令，第二行输出 `NONE`。\n" +
						"2. **只输出命令**：不要包含任何解释、Markdown 格式。每行只包含一个可执行命令。\n" +
						"3. **优先标准参数**：优先使用 `--help` 和 `--version`。\n" +
						"4. **内置命令处理**：Shell 内置命令使用 `help` 或 `man`，版本命令输出 `NONE`。\n" +
						"5. **示例**：\n" +
						"   输入: git\n" +
						"   输出:\n" +
						"   git --help\n" +
						"   git --version",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("我的系统是%s, 我需要查询的命令是: %s", osname, program),
				},
			},
			Temperature: 1,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	lines := strings.Split(strings.TrimSpace(resp.Choices[0].Message.Content), "\n")
	var helpCmd, verCmd []string
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		helpCmd = strings.Fields(strings.TrimSpace(lines[0]))
	}
	if len(lines) > 1 && strings.TrimSpace(lines[1]) != "NONE" {
		verCmd = strings.Fields(strings.TrimSpace(lines[1]))
	}
	return helpCmd, verCmd, nil
}

// AnalyzeHelpDoc 分析帮助文档并输出
func (c *Client) AnalyzeHelpDoc(ctx context.Context, useStream, useShort, isMissing bool, subQuery, usedCmd, helpOutput, versionOutput, cmdPath string) error {
	osname := runtime.GOOS
	systemPrompt := c.buildSystemPrompt(useShort, isMissing, subQuery)
	userContent := c.buildUserPrompt(osname, usedCmd, helpOutput, versionOutput, subQuery, cmdPath, isMissing, useShort)

	req := openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userContent},
		},
		Temperature: 1,
	}

	if useStream {
		stream, err := c.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			return err
		}
		defer stream.Close()

		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}
			fmt.Print(resp.Choices[0].Delta.Content)
		}
		fmt.Println()
		return nil
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return err
	}
	fmt.Println(resp.Choices[0].Message.Content)
	return nil
}

// ExplainCommand 解析并解释完整的命令
func (c *Client) ExplainCommand(ctx context.Context, useStream bool, fullCommand, helpOutput, cmdPath string) error {
	osname := runtime.GOOS
	systemPrompt := "你是一个命令行专家。用户输入了一条具体的命令，你需要详细解析该命令的含义，并给出优化建议。\n\n" +
		"【必须遵守的规则】\n" +
		"1. **逐层解析**：拆解命令的每一个部分（主命令、子命令、Flag参数、参数值），解释其具体作用。\n" +
		"2. **总结作用**：用一句话概括这条命令执行后会发生什么。\n" +
		"3. **优化建议**：基于该命令的意图，给出 1-2 条优化建议、更现代的替代方案，或者执行该命令后的常见后续操作。\n" +
		"4. **严禁 Markdown**：绝对不要使用 markdown 格式。输出必须是纯文本，使用缩进和列表来组织结构。\n" +
		"5. **准确性**：必须参考提供的帮助文档，不要编造参数含义。\n" +
		"6. **格式范例**：\n" +
		"   命令: git commit -am 'fix bug'\n" +
		"   位置: /usr/bin/git\n\n" +
		"   解析:\n" +
		"     git commit   提交代码更改\n" +
		"     -a           自动暂存所有已修改的文件（不包括新文件）\n" +
		"     -m '...'     指定提交说明信息\n\n" +
		"   总结: 暂存所有已跟踪文件的修改并创建新的提交。\n\n" +
		"   建议:\n" +
		"     - 如果有未跟踪的新文件，请先执行 `git add .`\n" +
		"     - 提交后通常需要执行 `git push` 推送到远程仓库"

	userContent := fmt.Sprintf("我的系统环境是%s\n命令安装位置: %s\n\n**用户输入的完整命令**: %s\n\n参考帮助文档:\n%s", osname, cmdPath, fullCommand, helpOutput)

	req := openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userContent},
		},
		Temperature: 1,
	}

	if useStream {
		stream, err := c.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			return err
		}
		defer stream.Close()

		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}
			fmt.Print(resp.Choices[0].Delta.Content)
		}
		fmt.Println()
		return nil
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return err
	}
	fmt.Println(resp.Choices[0].Message.Content)
	return nil
}

// GenerateCommand 根据自然语言描述生成命令
func (c *Client) GenerateCommand(ctx context.Context, useStream bool, program, description, helpOutput, cmdPath string) error {
	osname := runtime.GOOS
	systemPrompt := "你是一个命令行专家。用户指定了一个主命令和一段自然语言描述，请根据帮助文档，将用户的自然语言需求转换为最准确的执行命令。\n\n" +
		"【必须遵守的规则】\n" +
		"1. **生成命令**：直接给出一条可执行的、最符合用户需求的完整命令。\n" +
		"2. **命令解析**：简要解释命令中用到的关键参数。\n" +
		"3. **使用简短命令**：除非必要，否则**不要**在生成的命令中使用绝对路径（例如，使用 `git` 而不是 `/usr/bin/git`）。\n" +
		"4. **相关建议**：执行该命令后的注意事项或下一步操作建议。\n" +
		"5. **严禁 Markdown**：绝对不要使用 markdown 格式。输出必须是纯文本。\n" +
		"6. **准确性**：必须参考提供的帮助文档。\n" +
		"7. **格式范例**：\n" +
		"   需求: git 设置全局用户名\n" +
		"   位置: /usr/bin/git\n\n" +
		"   推荐命令:\n" +
		"     git config --global user.name \"Your Name\"\n\n" +
		"   解析:\n" +
		"     config    修改配置文件\n" +
		"     --global  写入全局配置 (~/.gitconfig)\n" +
		"     user.name 设置用户名字段\n\n" +
		"   建议:\n" +
		"     - 你可能还需要设置邮箱: git config --global user.email \"you@example.com\"\n" +
		"     - 查看当前配置: git config --list"

	userContent := fmt.Sprintf("我的系统环境是%s\n命令安装位置: %s\n主命令: %s\n**用户需求**: %s\n\n参考帮助文档:\n%s", osname, cmdPath, program, description, helpOutput)

	req := openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userContent},
		},
		Temperature: 1,
	}

	if useStream {
		stream, err := c.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			return err
		}
		defer stream.Close()

		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}
			fmt.Print(resp.Choices[0].Delta.Content)
		}
		fmt.Println()
		return nil
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return err
	}
	fmt.Println(resp.Choices[0].Message.Content)
	return nil
}

func (c *Client) buildSystemPrompt(useShort, isMissing bool, subQuery string) string {
	if isMissing {
		return "你是一个精通服务器的专家。用户询问的命令**在本地尚未安装**。\n" +
			"你的任务是根据你的知识库，为用户提供该命令的介绍、安装指南和基础用法。\n\n" +
			"【必须遵守的规则】\n" +
			"1. **介绍**：一句话简要说明该命令的核心功能。\n" +
			"2. **位置**：必须输出一行 `位置: 该命令尚未安装`。\n" +
			"3. **安装指南**：结合用户的操作系统，提供 2-3 种推荐的安装方式（按推荐程度排序）。例如 macOS 首选 brew，Linux 首选 apt/yum，Windows 首选 winget/choco。必须包含具体的可执行命令。\n" +
			"4. **常用示例**：提供 3-5 个最经典的基础用法示例。\n" +
			"5. **全程中文**：解释说明必须是中文。\n" +
			"6. **严禁 Markdown**：绝对不要使用 markdown 格式。输出必须是纯文本。\n" +
			"7. **格式范例**：\n" +
			"   介绍: 强大的HTTP命令行客户端\n" +
			"   位置: 该命令尚未安装\n\n" +
			"   推荐安装:\n" +
			"     1. Homebrew (推荐): brew install curl\n" +
			"     2. 源码编译: wget https://curl.se/download/curl-7.79.1.tar.gz ...\n\n" +
			"   常用示例:\n" +
			"     curl google.com     # 发送 GET 请求"
	}

	if subQuery != "" {
		return "你是一个命令行专家。用户想查询主命令下某个**特定子命令或参数**的具体用法。\n" +
			"请基于主命令的帮助文档（以及你的专业知识），重点解释该子命令。\n\n" +
			"【必须遵守的规则】\n" +
			"1. **验证有效性**：首先判断用户查询的子命令/参数是否有效。如果显然是错误的（文档里没有且不符合常理），请直接输出“用法错误：该命令不存在此用法”，并给出最接近的正确用法建议。\n" +
			"2. **核心解释**：一句话概括该子命令的核心作用（中文）。\n" +
			"3. **实战示例**：这是重点！提供 3-5 个结合开发经验的、最实用的场景示例（例如 `go build -o app main.go`）。示例必须准确、可执行。\n" +
			"4. **全程中文**：解释说明必须是中文。\n" +
			"5. **严禁 Markdown**：绝对不要使用 markdown 格式。输出必须是纯文本，类似精简速查表。\n" +
			"6. **格式范例**：\n" +
			"   子命令: build\n" +
			"   作用: 编译包和依赖项\n\n" +
			"   常用用法:\n" +
			"     go build -o myapp .   # 编译当前目录并指定输出文件名\n" +
			"     go build -v ./...     # 编译当前及子目录下所有包，并显示进度"
	}
	if useShort {
		return "你是一个精通服务器的专家。请为用户生成一份**中文精简速查表**。\n\n" +
			"【必须遵守的规则】\n" +
			"1. **简要介绍**：在输出的第一行，必须先用一句话简要说明该命令的核心功能。\n" +
			"2. **位置信息**：在介绍下方单列一行 `位置: [程序路径]`（路径由用户提供）。\n" +
			"3. **版本分析**：从提供的版本信息中提取版本号。如果提供了版本信息，在位置下方单列一行 `版本: x.y.z`。如果未提供，则不显示。\n" +
			"4. **只看核心**：忽略版本号、版权、页脚等无关信息，只筛选出最常用、最高频的 5-10 个选项/参数。\n" +
			"5. **全程中文**：所有解释必须是中文。如果原输出是英文，必须翻译。\n" +
			"6. **严禁 Markdown**：绝对不要使用 markdown 格式。输出必须是纯文本。\n" +
			"7. **极简风格**：采用紧凑的列表格式。参数解释不超过 20 个字。\n" +
			"8. **实战示例**：必须提供 3-5 个最经典的实战场景命令。\n" +
			"9. **无废话**：直接输出内容。\n\n" +
			"【输出格式范例】\n" +
			"介绍: 用于列出目录内容及文件信息的常用工具。\n" +
			"位置: /bin/ls\n" +
			"版本: 8.32 (如无则省略)\n\n" +
			"常用选项:\n" +
			"  -a, --all   显示所有文件（包括隐藏文件）\n" +
			"  -l          使用详细列表格式\n" +
			"  -h          以人类可读的格式显示大小\n\n" +
			"常用示例:\n" +
			"  ls -lah     # 以列表方式显示所有文件的大小\n" +
			"  ls *.go     # 列出所有 go 文件"
	}
	return "你是一个精通服务器及各种编程语言的专家。用户会提供一段程序命令的帮助文档。\n" +
		"你的任务是生成一份高质量的**中文帮助手册**。\n\n" +
		"【必须遵守的规则】\n" +
		"1. **全程中文**：所有解释、说明必须翻译成中文。保留命令参数（如 -h, --help）和专有名词（如 TCP, JSON）的原样。\n" +
		"2. **严禁 Markdown**：绝对不要使用 markdown 格式（不要用 ```代码块```，不要用 **加粗**，不要用 # 标题）。输出必须是纯文本，模仿终端的原始输出风格。\n" +
		"3. **格式保持**：尽量保持原文档的缩进和布局，方便用户左右对照。\n" +
		"4. **常用示例**：在文档末尾，必须补充 3-5 个最常用的实战命令示例，并附带简短中文说明。\n" +
		"5. **信息补充**：请在文档开头明确列出程序的安装位置（用户会提供）。\n" +
		"6. **无废话**：直接输出结果，不要包含“好的”、“这是翻译”等对话性文字。"
}

func (c *Client) buildUserPrompt(osname, usedCmd, helpOut, verOut, subQuery, cmdPath string, isMissing, useShort bool) string {
	if isMissing {
		return fmt.Sprintf("我的系统环境是%s\n我想要查询的命令是: %s (该命令在本地未安装)", osname, usedCmd)
	}

	content := fmt.Sprintf("我的系统环境是%s\n命令安装位置: %s\n这是执行的帮助指令及输出的文档%s\n%s", osname, cmdPath, usedCmd, helpOut)
	if subQuery != "" {
		content += fmt.Sprintf("\n\n**我具体想了解的子命令/参数是**: %s", subQuery)
	} else if useShort && verOut != "" {
		content += fmt.Sprintf("\n\n这是版本查询输出:\n%s", verOut)
	}
	return content
}

# GHP (General Help Provider)

GHP 是一个基于 AI (DeepSeek) 的智能命令行助手。它不仅能帮你查询命令的帮助文档，还能解析复杂的命令行参数，甚至在你未安装某个命令时告诉你如何安装。

告别繁琐的 `man` 手册，让 AI 为你提供精简、实用的中文速查表。

## ✨ 核心特性

*   **⚡️ 智能速查**：自动提取最常用的参数和示例，生成中文精简速查表。
*   **🔍 子命令查询**：支持深入查询特定子命令（如 `ghp git commit`）。
*   **🧐 命令解析**：逐层解析复杂的命令行参数，告诉你这行命令到底在干什么（`-a/--analyze`）。
*   **✨ 自然语言生成**：用人话描述需求，AI 帮你生成精准的执行命令（`-g/--generate`）。
*   **👻 离线/未安装支持**：本地没有安装的命令？没关系，AI 告诉你它的作用和安装方法（`-f/--force`）。
*   **🛠️ 自动容错**：智能探测命令是否存在，支持 `nvm` 等 Shell 函数，自动处理终端格式问题。

## 🚀 快速开始

### 安装

```bash
# 编译安装
go build -o ghp main.go
# 建议移动到 PATH 路径下
mv ghp /usr/local/bin/
```

### 配置

GHP 需要 OpenAI 兼容的 API Key（推荐使用 DeepSeek）。请设置以下环境变量：

```bash
export GHP_API_KEY="your-api-key"
# 可选，默认为 DeepSeek 官方 API
export GHP_BASE_URL="https://dashscope.aliyuncs.com/compatible-mode/v1"
export GHP_MODEL="deepseek-v3.2"
```

## 📖 使用指南

### 1. 基础查询 (默认精简模式)
查询 `ls` 命令的最常用法：
```bash
ghp ls
```

### 2. 子命令查询
查询 `git` 的 `commit` 子命令用法：
```bash
ghp git commit
```

### 3. 命令解析模式 (-a / --analyze)
遇到看不懂的长命令？让 AI 帮你拆解：
```bash
ghp -a git commit -am "fix bug"
```
*输出：解释 commit 是提交，-a 是自动暂存，-m 是消息，并给出建议。*

### 4. 命令生成模式 (-g / --generate)
想用 `ffmpeg` 转码但忘了参数？直接告诉 AI：
```bash
ghp -g ffmpeg 将 video.mp4 转换为 gif 图片
```
*输出：AI 生成的准确 ffmpeg 命令及参数解释。*

### 5. 强制查询模式 (-f / --force)
想查 `rust` 但本地没装？
```bash
ghp -f rust
```
*输出：Rust 介绍，以及针对你系统的安装命令（如 brew install rust）。*

### 6. 完整模式
如果你需要查看 AI 翻译的完整帮助文档（非精简版）：
```bash
ghp --concise=false git
```

## 📝 License

MIT
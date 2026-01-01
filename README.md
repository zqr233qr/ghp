# GHP (General Help Provider) 🧞‍♂️

**让命令行更懂你。**

GHP 是一个基于 AI (DeepSeek) 的智能命令行助手。它不仅能帮你查询命令的帮助文档，还能解析复杂的命令行参数，甚至在你未安装某个命令时告诉你如何安装。

告别繁琐的 `man` 手册，让 AI 为你提供精简、实用的中文速查表。

[![CI](https://github.com/zqr233qr/ghp/actions/workflows/ci.yml/badge.svg)](https://github.com/zqr233qr/ghp/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/zqr233qr/ghp)](https://goreportcard.com/report/github.com/zqr233qr/ghp)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## ✨ 核心特性

*   **⚡️ 智能速查**：自动提取最常用的参数和示例，生成中文精简速查表。
*   **🔍 子命令查询**：支持深入查询特定子命令（如 `ghp git commit`）。
*   **🧐 命令解析**：逐层解析复杂的命令行参数，告诉你这行命令到底在干什么（`-a/--analyze`）。
*   **✨ 自然语言生成**：用人话描述需求，AI 帮你生成精准的执行命令（`-g/--generate`）。
*   **👻 离线/未安装支持**：本地没有安装的命令？没关系，AI 告诉你它的作用和安装方法（`-f/--force`）。
*   **🛠️ 自动容错**：智能探测命令是否存在，支持 `nvm` 等 Shell 函数，自动处理终端格式问题。

---

## 🚀 快速开始

### 安装

推荐直接从 [Releases](https://github.com/zqr233qr/ghp/releases) 页面下载对应您系统的最新二进制文件。

**macOS / Linux:**
```bash
# 下载并解压 (示例: v0.0.2)
wget https://github.com/zqr233qr/ghp/releases/download/v0.0.2/ghp_Linux_x86_64.tar.gz
tar -xvf ghp_Linux_x86_64.tar.gz

# 移动到 PATH 路径
sudo mv ghp /usr/local/bin/
```

**源码编译:**
```bash
go install github.com/zqr233qr/ghp@latest
```

### 配置

GHP 需要 OpenAI 兼容的 API Key（推荐使用 DeepSeek）。请在您的 `.bashrc` 或 `.zshrc` 中设置：

```bash
export GHP_API_KEY="your-api-key"
# 可选，默认为 DeepSeek 官方 API
export GHP_BASE_URL="https://dashscope.aliyuncs.com/compatible-mode/v1"
export GHP_MODEL="deepseek-v3.2"
```

---

## 📖 使用指南与实战演示

### 1. 基础查询 (默认精简模式)
查询命令的核心用法，直接展示中文介绍、位置、版本和常用示例。

```bash
$ ghp git

介绍: 用于分布式版本控制的强大工具。
位置: /usr/local/bin/git
版本: 2.52.0

常用选项:
  -C <路径>    指定工作目录路径
  -c <名称>=<取值>  设置 Git 配置
  -h, --help    显示帮助信息

常用示例:
  git clone <仓库地址>  # 克隆远程仓库
  git add .   # 添加当前目录所有文件到暂存区
  git commit -m "提交说明"  # 提交暂存区内容到本地仓库
  git push origin main  # 将本地 main 分支推送到远程
```

### 2. 子命令查询
查询 `git` 的 `commit` 子命令用法，现在也会显示主命令信息。

```bash
$ ghp git commit

子命令: commit
作用: 记录变更到仓库

常用用法:
  git commit -m "添加新功能"  # 提交暂存区的修改，并添加简短描述
  git commit -am "更新文档"  # 自动将所有已跟踪文件的修改添加到暂存区并提交，同时添加描述
  git commit --amend -m "修正之前提交的描述"  # 修改上一次提交的描述
```

### 3. 命令解析模式 (-a / --analyze)
遇到复杂的长命令，不知道每个参数什么意思？让 AI 帮你逐层拆解。

```bash
$ ghp -a go build -ldflags "-s -w" -o app .

命令: go build -ldflags "-s -w" -o app .
位置: /usr/local/bin/go

解析:
  go build       编译包及其依赖项
  -ldflags       指定传递给链接器的额外标志
  "-s -w"        去除符号表和调试信息，减小二进制文件大小
  -o app         指定输出的可执行文件名为 app
  .              指定当前目录作为要编译的包

总结: 编译当前目录下的 Go 包及其依赖项，去除符号表和调试信息以减小文件大小，并将生成的可执行文件命名为 app。
```

### 4. 命令生成模式 (-g / --generate)
忘记具体参数怎么写？直接告诉 AI 你想干什么。

```bash
$ ghp -g git 将当前分支重命名为 main

推荐命令:
  git branch -m main

解析:
  branch    用于管理分支的命令
  -m        重命名分支

建议:
  - 若该分支已推送到远程仓库，需要将重命名后的分支推送到远程仓库...
```

### 5. 强制/离线查询模式 (-f / --force)
想了解一个还没安装的命令？使用 `-f` 强制查询。

```bash
$ ghp -f rust

介绍: Rust是一种系统级编程语言，注重安全性、性能和并发性。
位置: 该命令尚未安装

推荐安装:
  1. rustup (推荐): curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
  2. Homebrew: brew install rust

常用示例:
  cargo new my_project --bin     # 创建一个新的Rust二进制项目
  cargo build                    # 编译项目
  cargo run                      # 编译并运行项目
```

### 6. 完整模式 (-c=false / --concise=false)
需要查看 AI 翻译的完整帮助文档，格式现在也更清晰了。

```bash
$ ghp -c=false ssh-copy-id

介绍: 将本地 SSH 公钥复制到远程主机，以实现无密码登录
位置: /usr/bin/ssh-copy-id

帮助原文:
  用法: /usr/bin/ssh-copy-id [-h|-?|-f|-n|-s|-x] [-i [identity_file]] ...
  选项:
    -f: 强制模式 -- 不检查密钥是否已安装，直接复制密钥
    -n: 试运行模式 -- 不实际复制密钥
    ...

常用示例:
  ssh-copy-id user@example.com      # 将本地公钥复制到 example.com 主机的 user 用户
  ssh-copy-id -i ~/.ssh/id_rsa.pub user@example.com  # 指定公钥文件并复制...
```

## ⚙️ 参数说明

| 选项 | 全称 | 描述 |
| :--- | :--- | :--- |
| `-c` | `--concise` | 是否精简输出 (默认 true) |
| `-s` | `--stream` | 是否使用流式输出 (默认 true) |
| `-a` | `--analyze` | 解析模式：解释具体命令及参数含义 |
| `-g` | `--generate` | 生成模式：根据自然语言描述生成命令 |
| `-f` | `--force` | 强制模式：查询未安装的命令 |

## 📝 License

MIT © 2025 [zqr233qr](https://github.com/zqr233qr)
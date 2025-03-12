# AImmit

AImmit 是一个用 Go 开发的命令行 AI 工具，用于根据代码变更自动生成 commit message，以及总结 Git 仓库的 commit 历史。它可以帮助开发者提高工作效率，生成规范的提交信息。

## 命名由来

AI + Commit，发音类似“Aim it”，意为“瞄准它”，比喻精准总结

## 功能特点

- **生成Commit Message**：根据当前工作区的代码变更，自动生成符合约定式提交规范的 commit message
- **多种输出格式**：支持文本、JSON 和约定式提交格式
- **自动提交**：可选择自动执行 git commit 操作
- **简单易用**：友好的命令行界面

## 安装

确保您已安装 Go 1.20 或更高版本，然后运行：

```bash
go install github.com/rust17/AImmit/cmd/aimmit@latest
```

或者从源码构建：

```bash
git clone https://github.com/rust17/AImmit.git
cd AImmit
go build -o aimmit ./cmd/aimmit
```

## 使用方法

### 生成 Commit Message（默认模式）

```bash
aimmit
```

### 命令行参数
- `--format`: 输出格式，支持 text、json、conventional（仅commit模式），默认为 text
- `--repo`: Git 仓库路径（默认为当前目录）
- `--staged`: 是否只分析已暂存的更改（默认为true，只分析已暂存的更改）
- `--auto-commit`: 是否自动执行 git commit 操作（默认为false）
- `--ollama-url`: Ollama 服务 URL
- `--model`: Ollama 模型名称（默认为qwen2.5:3b）

### 示例

生成 commit message 并自动提交：

```bash
aimmit --auto-commit
```

分析未暂存的更改：

```bash
aimmit --staged=false
```

分析指定仓库路径：

```bash
aimmit --repo=/path/to/repo
```

## 约定式提交规范

AImmit 生成的 commit message 遵循[约定式提交规范](https://www.conventionalcommits.org/)，格式如下：

```
<类型>[可选的作用域]: <描述>

[可选的正文]

[可选的脚注]
```

常见的提交类型包括：

- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `style`: 代码风格调整（不影响代码功能）
- `refactor`: 代码重构
- `perf`: 性能优化
- `test`: 测试相关
- `build`: 构建系统或外部依赖变更
- `ci`: CI配置变更
- `chore`: 其他变更

## 许可证

MIT
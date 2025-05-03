# AImmit

AImmit 是一个用 Go 开发的命令行 AI 工具，用于根据代码变更自动生成 commit message，以及总结 Git 仓库的 commit 历史。它可以帮助开发者提高工作效率，生成规范的提交信息。

## 命名由来

AI + Commit，发音类似"Aim it"，意为"瞄准它"，比喻精准总结

## 功能特点

- **生成Commit Message**：根据当前工作区的代码变更，自动生成符合约定式提交规范的 commit message
- **多种输出格式**：支持文本、JSON 和约定式提交格式
- **自动提交**：可选择自动执行 git commit 操作
- **简单易用**：友好的命令行界面
- **本地大模型**：使用llama.cpp本地调用大语言模型，无需联网，保护代码安全

## 安装

### 方法一

确保您已安装 Go 1.20 或更高版本，然后运行：

```bash
go install github.com/rust17/AImmit/cmd/aimmit@latest
```

或者从源码构建：

```bash
git clone https://github.com/rust17/AImmit.git
cd AImmit
go build -o aimmit ./cmd/main.go
```

### 下载 llama.cpp
由于这个项目依赖 llama.cpp 调用大模型，所以需要下载 llama.cpp 的二进制文件，并放入 `./llama-c-path/bin` 目录下。

```bash
wget -q https://github.com/ggml-org/llama.cpp/releases/download/{llama-bin.zip} -O temp.zip && unzip temp.zip -d ./temp && mv ./temp/build/bin/* ./llama-c-path/ && rm -rf temp && rm temp.zip
```

### 下载 GGUF 模型

```bash
wget https://huggingface.co/lmstudio-community/Qwen3-1.7B-GGUF/resolve/main/Qwen3-1.7B-Q6_K.gguf -O ./model/Qwen3-1.7B-Q6_K.gguf
```

### 方法二（Docker 构建）

AImmit 也支持通过 Docker 进行构建和运行：

```bash
# 克隆仓库
git clone https://github.com/rust17/AImmit.git
cd AImmit

# 将你的GGUF模型文件放入models目录
wget https://huggingface.co/lmstudio-community/Qwen3-1.7B-GGUF/resolve/main/Qwen3-1.7B-Q6_K.gguf -O ./model/Qwen3-1.7B-Q6_K.gguf

# 构建Docker镜像
docker build -t aimmit:latest .
```

## 使用方法

### 生成 Commit Message（默认模式）
直接运行
```bash
aimmit
```
或者用 Docker 运行，需要将模型文件放入 `./model` 目录下，并挂载当前目录为 git-repo：
```bash
docker run -v $(pwd)/model:/app/model -v $(pwd):/git-repo -it aimmit
```

### 命令行参数
- `--format`: 输出格式，支持 text、json、conventional（仅commit模式），默认为 text
- `--repo`: Git 仓库路径（默认为当前目录）
- `--staged`: 是否只分析已暂存的更改（默认为true，只分析已暂存的更改）
- `--auto-commit`: 是否自动执行 git commit 操作（默认为false）
- `--model-path`: llama.cpp模型文件路径，例如：`/home/user/models/llama3.gguf`
- `--llama-c-path`: llama.cpp可执行文件路径（默认为llama-main）
- `--only-prompt`: 是否只显示prompt（默认为false）

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
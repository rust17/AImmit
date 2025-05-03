package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rust17/AImmit/internal/git"
)

// Client 是AI服务的客户端
type Client struct {
	debug        bool    // 是否开启debug模式
	modelPath    string  // llama.cpp模型文件路径
	modelName    string  // 模型名称
	llamaCppPath string  // llama.cpp可执行文件路径
	temperature  float64 // 生成温度
	maxTokens    int     // 最大生成的token数
	topP         float64 // top-p
	topK         int     // top-k
	minP         float64 // min-p
}

// NewClient 创建一个新的AI客户端
// 参数是模型文件路径，如果为空则尝试使用默认路径
func NewClient(debug bool) *Client {
	return &Client{
		debug:     debug,
		modelName: "Qwen3", // 默认使用Qwen3模型
		maxTokens: 2048,
		topP:      0.8,
		topK:      20,
		minP:      0,
	}
}

// SetModel 设置要使用的模型路径
func (c *Client) SetModel(modelPath string) {
	c.modelPath = modelPath
}

// SetModelName 设置模型名称
func (c *Client) SetModelName(modelName string) {
	c.modelName = modelName
}

// SetLlamaCppPath 设置llama.cpp可执行文件路径
func (c *Client) SetLlamaCppPath(path string) {
	c.llamaCppPath = path
}

// SetTemperature 设置生成温度
func (c *Client) SetTemperature(temp float64) {
	c.temperature = temp
}

// SetMaxTokens 设置最大生成的token数
func (c *Client) SetMaxTokens(tokens int) {
	c.maxTokens = tokens
}

// CommitMessage 表示生成的提交信息
type CommitMessage struct {
	Subject         string `json:"subject"`          // 提交的主题行（简短描述）
	Body            string `json:"body"`             // 提交的详细描述
	Type            string `json:"type"`             // 提交类型（feat, fix, docs等）
	Scope           string `json:"scope"`            // 影响范围
	BreakingChanges bool   `json:"breaking_changes"` // 是否包含破坏性变更
	RawDiff         string `json:"-"`                // 原始diff内容（不包含在JSON输出中）
}

// callLlamaCpp 调用llama.cpp可执行文件生成回复
func (c *Client) callLlamaCpp(prompt string, onlyPrompt bool) (string, error) {
	// 终止标记（可以自定义）
	stopMarker := "<|end_of_text|>"
	// 添加系统提示到用户提示之前
	fullPrompt := fmt.Sprintf("<|im_start|>system\n你是一个专业的代码提交分析助手，擅长总结Git提交历史和生成规范的commit message。可以拼接技术术语英文，不过请尽可能用中文回答。请以字符%s结束/no_think<|im_end|>\n<|im_start|>user\n%s<|im_end|>\n<|im_start|>assistant\n", stopMarker, prompt)
	// 超时时间
	timeout := 2 * time.Minute

	// 如果只是打印提示信息，则输出并退出
	if onlyPrompt || c.debug {
		fmt.Println(fullPrompt)
		if onlyPrompt {
			os.Exit(1)
		}
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 构建llama.cpp命令行参数
	cmd := exec.CommandContext(
		ctx,
		c.llamaCppPath+"/llama-cli",
		"-m", c.modelPath,
		"-p", fullPrompt,
		"--no-display-prompt",
		"--n-predict", fmt.Sprintf("%d", c.maxTokens),
		// Qwen3-1.7B-Q6_K.gguf 模型最佳参数
		"--min-p", fmt.Sprintf("%.2f", c.minP),
		"--temp", fmt.Sprintf("%.2f", c.temperature),
		"--top-p", fmt.Sprintf("%.2f", c.topP),
		"--top-k", fmt.Sprintf("%d", c.topK),
	)
	cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+c.llamaCppPath)

	// 创建管道获取实时输出
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("创建输出管道失败: %w", err)
	}

	if c.debug {
		// 创建管道获取实时错误输出
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return "", fmt.Errorf("创建错误输出管道失败: %w", err)
		}

		// 启动goroutine来处理错误输出
		go func() {
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}
		}()
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("启动llama.cpp失败: %w", err)
	}

	// 用于存储完整输出
	var outputBuilder strings.Builder
	// 使用扫描器来实时读取输出
	scanner := bufio.NewScanner(stdoutPipe)

	// 启动goroutine来处理输出
	go func() {
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				// 上下文被取消，立即退出
				return
			default:
				line := scanner.Text()
				outputBuilder.WriteString(line + "\n")

				if c.debug {
					fmt.Println(line) // debug 实时打印输出
				}

				if strings.Contains(line, stopMarker) {
					cmd.Process.Kill()
					return
				}
			}
		}
	}()

	// 确保进程已结束
	err = cmd.Wait()
	if err != nil && ctx.Err() == context.DeadlineExceeded {
		return outputBuilder.String(), fmt.Errorf("执行llama.cpp超时")
	}

	return outputBuilder.String(), nil
}

// GenerateCommitMessage 根据diff生成commit message
func (c *Client) GenerateCommitMessage(diffInfo *git.DiffInfo, onlyPrompt bool) (*CommitMessage, error) {
	// 构建提示信息
	prompt := buildDiffPrompt(diffInfo)

	// 调用llama.cpp
	response, err := c.callLlamaCpp(prompt, onlyPrompt)
	if err != nil {
		return nil, err
	}
	if c.debug {
		fmt.Println(response) // debug 响应
	}

	// 解析AI响应
	commitMsg, err := parseCommitMessage(response, diffInfo)
	if err != nil {
		return nil, err
	}

	return commitMsg, nil
}

// buildDiffPrompt 构建发送给AI的提示信息（用于生成commit message）
func buildDiffPrompt(diffInfo *git.DiffInfo) string {
	var sb strings.Builder

	sb.WriteString("请根据以下Git差异信息，生成一个符合约定式提交规范(Conventional Commits)的提交信息。\n\n")

	sb.WriteString("修改的文件：\n")
	for i, file := range diffInfo.Files {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, file))
	}

	sb.WriteString(fmt.Sprintf("\n添加行数: %d\n", diffInfo.Additions))
	sb.WriteString(fmt.Sprintf("删除行数: %d\n", diffInfo.Deletions))

	// 最大允许的diff内容长度
	const maxDiffLength = 3000

	// 如果diff内容不太长，则包含完整diff
	if len(diffInfo.RawDiff) <= maxDiffLength {
		sb.WriteString("\n差异详情：\n```\n")
		sb.WriteString(diffInfo.RawDiff)
		sb.WriteString("\n```\n")
	} else {
		// 对于长diff，尝试为每个文件提供一些上下文
		sb.WriteString("\n差异详情（摘要）：\n")

		// 按文件分割diff内容
		fileDiffs := splitDiffByFile(diffInfo.RawDiff)

		// 为每个文件分配一定的字符配额
		quotaPerFile := maxDiffLength / len(fileDiffs)
		if quotaPerFile < 500 {
			quotaPerFile = 500 // 确保每个文件至少有500个字符
		}

		totalUsed := 0
		for i, fileDiff := range fileDiffs {
			if i >= 10 { // 最多显示10个文件的diff
				sb.WriteString("\n... 还有更多文件的变更未显示 ...\n")
				break
			}

			// 计算这个文件可以使用的字符数
			availableChars := quotaPerFile
			if totalUsed+availableChars > maxDiffLength {
				availableChars = maxDiffLength - totalUsed
				if availableChars < 300 { // 如果剩余空间太小，就不再显示更多文件
					sb.WriteString("\n... 还有更多文件的变更未显示 ...\n")
					break
				}
			}

			// 提取文件名
			fileName := extractFileName(fileDiff)
			sb.WriteString(fmt.Sprintf("\n文件: %s\n```\n", fileName))

			// 如果文件diff太长，则截断
			if len(fileDiff) > availableChars {
				// 尝试保留文件开头和结尾的一些内容
				headLength := availableChars * 2 / 3
				tailLength := availableChars - headLength - 20 // 20是省略号的长度

				if headLength > 0 && tailLength > 0 {
					sb.WriteString(fileDiff[:headLength])
					sb.WriteString("\n... (内容过长已截断) ...\n")
					if len(fileDiff) > len(fileDiff)-tailLength {
						sb.WriteString(fileDiff[len(fileDiff)-tailLength:])
					}
				} else {
					// 如果无法同时保留头尾，则只保留开头
					sb.WriteString(fileDiff[:availableChars])
					sb.WriteString("\n... (内容过长已截断) ...\n")
				}
			} else {
				sb.WriteString(fileDiff)
			}

			sb.WriteString("\n```\n")

			totalUsed += min(len(fileDiff), availableChars) + 100 // 100是文件名和格式化的额外字符

			if totalUsed >= maxDiffLength {
				sb.WriteString("\n... 还有更多文件的变更未显示 ...\n")
				break
			}
		}
	}

	sb.WriteString("\n请以JSON格式返回，包含以下字段：\n")
	sb.WriteString("1. type: 提交类型（feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert等）\n")
	sb.WriteString("2. scope: 影响范围（可选，例如组件名或文件名）\n")
	sb.WriteString("3. subject: 简短描述（不超过50个字符）\n")
	sb.WriteString("4. body: 详细描述（可选,不超过100个字符）\n")
	sb.WriteString("\n重要：请只返回一个JSON对象，不要返回JSON数组。请综合所有变更生成一个最合适的提交信息。\n")

	return sb.String()
}

// splitDiffByFile 将完整的diff内容按文件分割
func splitDiffByFile(rawDiff string) []string {
	// 使用"diff --git"作为文件分隔符
	diffParts := strings.Split(rawDiff, "diff --git")

	result := []string{}
	for i, part := range diffParts {
		if i == 0 && len(part) == 0 {
			continue // 跳过第一个空元素
		}

		if i > 0 {
			// 重新添加分隔符，因为Split会移除它
			part = "diff --git" + part
		}

		result = append(result, part)
	}

	return result
}

// extractFileName 从文件diff中提取文件名
func extractFileName(fileDiff string) string {
	// 尝试从"diff --git a/path/to/file b/path/to/file"格式中提取
	lines := strings.Split(fileDiff, "\n")
	if len(lines) == 0 {
		return "未知文件"
	}

	firstLine := lines[0]
	if strings.HasPrefix(firstLine, "diff --git") {
		parts := strings.Split(firstLine, " ")
		if len(parts) >= 4 {
			// 通常格式是 "diff --git a/path/to/file b/path/to/file"
			// 我们取 b/path/to/file 部分
			return strings.TrimPrefix(parts[3], "b/")
		}
	}

	// 如果无法从第一行提取，尝试从+++ 行提取
	for _, line := range lines {
		if strings.HasPrefix(line, "+++ b/") {
			return strings.TrimPrefix(line, "+++ b/")
		}
	}

	return "未知文件"
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseCommitMessage 解析AI返回的commit message
func parseCommitMessage(response string, diffInfo *git.DiffInfo) (*CommitMessage, error) {
	// 尝试从响应中提取JSON部分
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		// 如果没有找到有效的JSON，尝试创建一个基本的commit message
		panic("没有找到有效的JSON，请重试")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var commitMsg CommitMessage
	if err := json.Unmarshal([]byte(jsonStr), &commitMsg); err != nil {
		// 如果解析失败，创建一个基本的commit message
		panic("没有找到有效的JSON，请重试")
	}

	// 添加原始diff信息
	commitMsg.RawDiff = diffInfo.RawDiff

	return &commitMsg, nil
}

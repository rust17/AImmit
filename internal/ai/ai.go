package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rust17/AImmit/internal/git"
)

// Client 是AI服务的客户端
type Client struct {
	ollamaURL string
	modelName string
}

// NewClient 创建一个新的AI客户端
// 参数可以是Ollama服务的URL，如果为空则使用默认值
func NewClient(ollamaURL string) *Client {
	// 如果未提供URL，使用默认的本地Ollama地址
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	return &Client{
		ollamaURL: ollamaURL,
		modelName: "llama3", // 默认使用llama3模型，可以根据需要修改
	}
}

// SetModel 设置要使用的Ollama模型
func (c *Client) SetModel(modelName string) {
	c.modelName = modelName
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

// Ollama API请求结构
type ollamaRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Ollama API响应结构
type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Error string `json:"error,omitempty"`
}

// callOllama 调用Ollama API
func (c *Client) callOllama(prompt string) (string, error) {
	// 使用Client中配置的Ollama服务地址
	url := c.ollamaURL + "/api/chat"

	reqBody := ollamaRequest{
		Model: c.modelName,
		Messages: []message{
			{
				Role:    "system",
				Content: "你是一个专业的代码提交分析助手，擅长总结Git提交历史和生成规范的commit message。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false, // 不使用流式响应
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("解析API响应失败: %w", err)
	}

	if ollamaResp.Error != "" {
		return "", fmt.Errorf("API错误: %s", ollamaResp.Error)
	}

	return ollamaResp.Message.Content, nil
}

// GenerateCommitMessage 根据diff生成commit message
func (c *Client) GenerateCommitMessage(diffInfo *git.DiffInfo) (*CommitMessage, error) {
	// 构建提示信息
	prompt := buildDiffPrompt(diffInfo)

	// 调用Ollama API
	response, err := c.callOllama(prompt)
	if err != nil {
		return nil, err
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
	const maxDiffLength = 6000

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

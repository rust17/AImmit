package summarizer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rust17/AImmit/internal/ai"
)

// Client 是总结格式化的客户端
type Client struct{}

// NewClient 创建一个新的总结客户端
func NewClient() *Client {
	return &Client{}
}

// FormatCommitMessage 根据指定格式输出commit message
func (c *Client) FormatCommitMessage(commitMsg *ai.CommitMessage, format string) (string, error) {
	switch strings.ToLower(format) {
	case "text":
		return c.formatCommitAsText(commitMsg), nil
	case "json":
		return c.formatCommitAsJSON(commitMsg)
	case "conventional":
		return c.formatCommitAsConventional(commitMsg), nil
	default:
		return "", fmt.Errorf("不支持的输出格式: %s", format)
	}
}

// formatCommitAsText 以纯文本格式输出commit message
func (c *Client) formatCommitAsText(commitMsg *ai.CommitMessage) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s: %s\n", commitMsg.Type, commitMsg.Subject))
	if commitMsg.Scope != "" {
		sb.WriteString(fmt.Sprintf("范围: %s\n", commitMsg.Scope))
	}

	if commitMsg.Body != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", commitMsg.Body))
	}

	if commitMsg.BreakingChanges {
		sb.WriteString("\n⚠️ 包含破坏性变更\n")
	}

	return sb.String()
}

// formatCommitAsJSON 以JSON格式输出commit message
func (c *Client) formatCommitAsJSON(commitMsg *ai.CommitMessage) (string, error) {
	// 创建一个包含所有信息的结构体
	type jsonOutput struct {
		Type            string `json:"type"`
		Scope           string `json:"scope,omitempty"`
		Subject         string `json:"subject"`
		Body            string `json:"body,omitempty"`
		BreakingChanges bool   `json:"breaking_changes"`
		Conventional    string `json:"conventional"`
	}

	output := jsonOutput{
		Type:            commitMsg.Type,
		Scope:           commitMsg.Scope,
		Subject:         commitMsg.Subject,
		Body:            commitMsg.Body,
		BreakingChanges: commitMsg.BreakingChanges,
		Conventional:    c.formatCommitAsConventional(commitMsg),
	}

	// 序列化为JSON
	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化JSON失败: %w", err)
	}

	return string(jsonBytes), nil
}

// formatCommitAsConventional 以约定式提交格式输出commit message
func (c *Client) formatCommitAsConventional(commitMsg *ai.CommitMessage) string {
	var sb strings.Builder

	// 构建第一行（类型、范围和主题）
	if commitMsg.BreakingChanges {
		if commitMsg.Scope != "" {
			sb.WriteString(fmt.Sprintf("%s(%s)!: %s", commitMsg.Type, commitMsg.Scope, commitMsg.Subject))
		} else {
			sb.WriteString(fmt.Sprintf("%s!: %s", commitMsg.Type, commitMsg.Subject))
		}
	} else {
		if commitMsg.Scope != "" {
			sb.WriteString(fmt.Sprintf("%s(%s): %s", commitMsg.Type, commitMsg.Scope, commitMsg.Subject))
		} else {
			sb.WriteString(fmt.Sprintf("%s: %s", commitMsg.Type, commitMsg.Subject))
		}
	}

	// 如果有详细描述，添加空行和详细描述
	if commitMsg.Body != "" {
		sb.WriteString("\n\n")
		sb.WriteString(commitMsg.Body)
	}

	// 如果有破坏性变更，添加BREAKING CHANGE标记
	if commitMsg.BreakingChanges {
		if commitMsg.Body != "" {
			sb.WriteString("\n\n")
		} else {
			sb.WriteString("\n")
		}
		sb.WriteString("BREAKING CHANGE: 此提交包含破坏性变更")
	}

	return sb.String()
}

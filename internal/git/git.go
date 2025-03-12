package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Commit 表示一个Git提交
type Commit struct {
	Hash    string
	Author  string
	Date    time.Time
	Message string
}

// DiffInfo 表示Git差异信息
type DiffInfo struct {
	Files      []string // 修改的文件列表
	Additions  int      // 添加的行数
	Deletions  int      // 删除的行数
	RawDiff    string   // 原始diff内容
	StagedOnly bool     // 是否只包含已暂存的更改
}

// Client 是Git操作的客户端
type Client struct {
	RepoPath string // 导出字段，使其可以在外部访问
}

// NewClient 创建一个新的Git客户端
func NewClient(repoPath string) *Client {
	return &Client{
		RepoPath: repoPath,
	}
}

// GetCurrentDiff 获取当前工作区的差异
func (c *Client) GetCurrentDiff(stagedOnly bool) (*DiffInfo, error) {
	var cmd *exec.Cmd

	if stagedOnly {
		// 只获取已暂存的更改
		cmd = exec.Command("git", "-C", c.RepoPath, "diff", "--staged")
	} else {
		// 获取所有更改（包括未暂存的）
		cmd = exec.Command("git", "-C", c.RepoPath, "diff")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取diff失败: %w", err)
	}

	rawDiff := string(output)

	// 如果没有差异，尝试获取未跟踪的文件
	if rawDiff == "" && !stagedOnly {
		cmd = exec.Command("git", "-C", c.RepoPath, "ls-files", "--others", "--exclude-standard")
		output, err = cmd.Output()
		if err == nil && len(output) > 0 {
			rawDiff = "未跟踪的文件:\n" + string(output)
		}
	}

	// 获取修改的文件列表
	var filesCmd *exec.Cmd
	if stagedOnly {
		filesCmd = exec.Command("git", "-C", c.RepoPath, "diff", "--staged", "--name-only")
	} else {
		filesCmd = exec.Command("git", "-C", c.RepoPath, "diff", "--name-only")
	}

	filesOutput, err := filesCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取修改文件列表失败: %w", err)
	}

	files := []string{}
	if len(filesOutput) > 0 {
		files = strings.Split(strings.TrimSpace(string(filesOutput)), "\n")
	}

	// 获取未跟踪的文件
	if !stagedOnly {
		untrackedCmd := exec.Command("git", "-C", c.RepoPath, "ls-files", "--others", "--exclude-standard")
		untrackedOutput, err := untrackedCmd.Output()
		if err == nil && len(untrackedOutput) > 0 {
			untrackedFiles := strings.Split(strings.TrimSpace(string(untrackedOutput)), "\n")
			files = append(files, untrackedFiles...)
		}
	}

	// 计算添加和删除的行数
	var additions, deletions int

	// 使用git diff --stat来获取统计信息
	var statCmd *exec.Cmd
	if stagedOnly {
		statCmd = exec.Command("git", "-C", c.RepoPath, "diff", "--staged", "--stat")
	} else {
		statCmd = exec.Command("git", "-C", c.RepoPath, "diff", "--stat")
	}

	statOutput, err := statCmd.Output()
	if err == nil {
		statLines := strings.Split(strings.TrimSpace(string(statOutput)), "\n")
		if len(statLines) > 0 {
			// 最后一行通常包含总结信息，如 "10 files changed, 100 insertions(+), 50 deletions(-)"
			summaryLine := statLines[len(statLines)-1]
			// 解析添加的行数
			if idx := strings.Index(summaryLine, "insertion"); idx != -1 {
				start := strings.LastIndex(strings.TrimSpace(summaryLine[:idx]), " ") + 1
				addStr := summaryLine[start : idx-1]
				fmt.Sscanf(addStr, "%d", &additions)
			}

			// 解析删除的行数
			if idx := strings.Index(summaryLine, "deletion"); idx != -1 {
				start := strings.LastIndex(strings.TrimSpace(summaryLine[:idx]), " ") + 1
				delStr := summaryLine[start : idx-1]
				fmt.Sscanf(delStr, "%d", &deletions)
			}
		}
	}

	return &DiffInfo{
		Files:      files,
		Additions:  additions,
		Deletions:  deletions,
		RawDiff:    rawDiff,
		StagedOnly: stagedOnly,
	}, nil
}

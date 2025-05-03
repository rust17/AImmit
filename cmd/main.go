package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rust17/AImmit/internal/ai"
	"github.com/rust17/AImmit/internal/git"
	"github.com/rust17/AImmit/internal/summarizer"
	"github.com/rust17/AImmit/internal/utils"
)

func main() {
	// 定义命令行参数
	format := flag.String("format", "conventional", "输出格式 (text, json, conventional)")
	repoPath := flag.String("repo", ".", "Git仓库路径")
	stagedOnly := flag.Bool("staged", true, "是否只分析已暂存的更改")
	autoCommit := flag.Bool("auto-commit", false, "是否自动执行git commit")
	enableDebug := flag.Bool("debug", false, "是否开启debug模式")
	onlyPrompt := flag.Bool("only-prompt", false, "只显示prompt")
	llamaCPath := flag.String("llama-c-path", filepath.Join(utils.GetProjectRoot(), "./llama-c-path"), "llama.cpp项目路径")
	flag.Parse()

	// 从环境变量获取参数
	formatEnv := os.Getenv("FORMAT")
	if formatEnv != "" {
		format = &formatEnv
	}
	repoPathEnv := os.Getenv("REPO")
	if repoPathEnv != "" {
		repoPath = &repoPathEnv
	}
	llamaCPathEnv := os.Getenv("LLAMA_C_PATH")
	if llamaCPathEnv != "" {
		llamaCPath = &llamaCPathEnv
	}

	// 创建Git客户端
	gitClient := git.NewClient(*repoPath)

	// 创建AI客户端
	aiClient := ai.NewClient(*enableDebug)
	aiClient.SetLlamaCppPath(*llamaCPath)

	// 创建Summarizer客户端
	summarizerClient := summarizer.NewClient()

	if *enableDebug {
		startTime := time.Now()
		defer func() {
			fmt.Printf("执行时间: %v\n", time.Since(startTime))
		}()
	}

	// 生成commit message模式
	generateCommitMessage(gitClient, aiClient, summarizerClient, *format, *stagedOnly, *autoCommit, *onlyPrompt)
}

// generateCommitMessage 生成commit message
func generateCommitMessage(gitClient *git.Client, aiClient *ai.Client, summarizerClient *summarizer.Client, format string, stagedOnly, autoCommit, onlyPrompt bool) {
	// 获取当前差异
	diffInfo, err := gitClient.GetCurrentDiff(stagedOnly)
	if err != nil {
		fmt.Printf("获取差异信息失败: %v\n", err)
		os.Exit(1)
	}

	// 检查是否有差异
	if diffInfo.RawDiff == "" && len(diffInfo.Files) == 0 {
		fmt.Println("没有检测到任何更改")
		os.Exit(0)
	}

	// 调用AI服务生成commit message
	commitMsg, err := aiClient.GenerateCommitMessage(diffInfo, onlyPrompt)
	if err != nil {
		fmt.Printf("生成commit message失败: %v\n", err)
		os.Exit(1)
	}

	// 格式化并显示结果
	output, err := summarizerClient.FormatCommitMessage(commitMsg, format)
	if err != nil {
		fmt.Printf("格式化输出失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(output)

	// 如果启用了自动提交，执行git commit
	if autoCommit {
		// 获取约定式提交格式的commit message
		conventionalMsg, err := summarizerClient.FormatCommitMessage(commitMsg, "conventional")
		if err != nil {
			fmt.Printf("格式化commit message失败: %v\n", err)
			os.Exit(1)
		}

		// 执行git commit
		commitCmd := exec.Command("git", "-C", gitClient.RepoPath, "commit", "-m", conventionalMsg)
		if err := commitCmd.Run(); err != nil {
			fmt.Printf("执行git commit失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n✅ 已成功提交更改")
	}
}

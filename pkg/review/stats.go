package review

import (
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// analyzeStats 分析评审统计信息
func (r *Reviewer) analyzeStats(diffContent, reviewResult string) (*ReviewStats, error) {
	stats := &ReviewStats{
		IssuesByLevel:  make(map[string]int),
		CommonIssues:   make([]string, 0),
		ReviewDateTime: time.Now(),
	}

	// 分析 diff 内容
	var currentFile string
	changedFiles := make(map[string]bool)

	lines := strings.Split(diffContent, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Split(line, " ")
			if len(parts) > 2 {
				currentFile = strings.TrimPrefix(parts[2], "b/")
				if !r.shouldIgnoreFile(currentFile) {
					changedFiles[currentFile] = true
				}
			}
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			stats.LinesAdded++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			stats.LinesDeleted++
		}
	}
	stats.FilesChanged = len(changedFiles)

	// 分析评审结果
	sections := strings.Split(reviewResult, "##")
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if strings.HasPrefix(section, "主要问题") {
			// 统计问题级别
			if strings.Contains(section, "严重") {
				stats.IssuesByLevel["严重"]++
			} else if strings.Contains(section, "中等") {
				stats.IssuesByLevel["中等"]++
			} else if strings.Contains(section, "低") {
				stats.IssuesByLevel["低"]++
			}

			// 提取常见问题
			lines := strings.Split(section, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") {
					issue := strings.TrimSpace(strings.TrimPrefix(line, "1."))
					issue = strings.TrimSpace(strings.TrimPrefix(issue, "2."))
					if issue != "" {
						stats.CommonIssues = append(stats.CommonIssues, issue)
					}
				}
			}
		}
	}

	return stats, nil
}

// getGitInfo 获取 Git 信息
func (r *Reviewer) getGitInfo() (*GitInfo, error) {
	gitInfo := &GitInfo{}

	// 获取当前分支
	output, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return nil, err
	}
	gitInfo.Branch = strings.TrimSpace(string(output))

	// 获取最近的提交信息
	output, err = exec.Command("git", "log", "-1", "--pretty=format:%H|%s|%an").Output()
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(output), "|")
	if len(parts) == 3 {
		gitInfo.CommitHash = parts[0]
		gitInfo.CommitMessage = parts[1]
		gitInfo.Author = parts[2]
	}

	return gitInfo, nil
}

// shouldIgnoreFile 检查是否应该忽略文件
func (r *Reviewer) shouldIgnoreFile(filename string) bool {
	for _, pattern := range r.config.Review.IgnorePatterns {
		matched, err := filepath.Match(pattern, filename)
		if err == nil && matched {
			return true
		}
	}
	return false
}

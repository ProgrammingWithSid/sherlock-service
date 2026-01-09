package git

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// FileDiff represents changes to a single file
type FileDiff struct {
	Path      string
	Status    string // "added", "modified", "deleted", "renamed"
	Additions int
	Deletions int
	Hunks     []Hunk
}

// Hunk represents a contiguous block of changes
type Hunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Lines    []DiffLine
}

// DiffLine represents a single line change
type DiffLine struct {
	Type    string // "+", "-", " " (context)
	Content string
	LineNum int // Line number in new file (for additions) or old file (for deletions)
}

// GetFileDiff gets detailed diff information for a file
func (s *CloneService) GetFileDiff(repoPath string, baseBranch string, headBranch string, filePath string) (*FileDiff, error) {
	// Get file status
	status, err := s.getFileStatus(repoPath, baseBranch, headBranch, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file status: %w", err)
	}

	if status == "deleted" {
		return &FileDiff{
			Path:   filePath,
			Status: status,
		}, nil
	}

	// Get diff output
	gitCmd, err := getGitPath()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(gitCmd, "-C", repoPath, "diff", "--unified=0", fmt.Sprintf("%s...%s", baseBranch, headBranch), "--", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	diff := &FileDiff{
		Path:   filePath,
		Status: status,
	}

	// Parse diff output
	hunks, additions, deletions := parseDiffOutput(string(output))
	diff.Hunks = hunks
	diff.Additions = additions
	diff.Deletions = deletions

	return diff, nil
}

// GetChangedLines returns the line numbers that were changed in a file
func (s *CloneService) GetChangedLines(repoPath string, baseBranch string, headBranch string, filePath string) ([]int, error) {
	diff, err := s.GetFileDiff(repoPath, baseBranch, headBranch, filePath)
	if err != nil {
		return nil, err
	}

	changedLines := make(map[int]bool)
	for _, hunk := range diff.Hunks {
		// Add all lines in the hunk range
		for i := 0; i < hunk.NewLines; i++ {
			changedLines[hunk.NewStart+i] = true
		}
	}

	lines := make([]int, 0, len(changedLines))
	for line := range changedLines {
		lines = append(lines, line)
	}

	return lines, nil
}

// getFileStatus determines if a file was added, modified, or deleted
func (s *CloneService) getFileStatus(repoPath string, baseBranch string, headBranch string, filePath string) (string, error) {
	gitCmd, err := getGitPath()
	if err != nil {
		return "", err
	}
	// Check if file exists in base branch
	cmd := exec.Command(gitCmd, "-C", repoPath, "cat-file", "-e", fmt.Sprintf("%s:%s", baseBranch, filePath))
	err = cmd.Run()
	existsInBase := err == nil

	// Check if file exists in head branch
	cmd = exec.Command(gitCmd, "-C", repoPath, "cat-file", "-e", fmt.Sprintf("%s:%s", headBranch, filePath))
	err = cmd.Run()
	existsInHead := err == nil

	if !existsInBase && existsInHead {
		return "added", nil
	}
	if existsInBase && !existsInHead {
		return "deleted", nil
	}
	return "modified", nil
}

// parseDiffOutput parses git diff output into hunks
func parseDiffOutput(output string) ([]Hunk, int, int) {
	if len(output) == 0 {
		return []Hunk{}, 0, 0
	}

	var hunks []Hunk
	var currentHunk *Hunk
	additions := 0
	deletions := 0

	scanner := bufio.NewScanner(strings.NewReader(output))
	hunkHeaderRegex := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

	for scanner.Scan() {
		line := scanner.Text()

		// Match hunk header
		if matches := hunkHeaderRegex.FindStringSubmatch(line); matches != nil {
			if currentHunk != nil {
				hunks = append(hunks, *currentHunk)
			}

			oldStart, _ := strconv.Atoi(matches[1])
			oldLines := 0
			if matches[2] != "" {
				oldLines, _ = strconv.Atoi(matches[2])
			}
			newStart, _ := strconv.Atoi(matches[3])
			newLines := 0
			if matches[4] != "" {
				newLines, _ = strconv.Atoi(matches[4])
			}

			currentHunk = &Hunk{
				OldStart: oldStart,
				OldLines: oldLines,
				NewStart: newStart,
				NewLines: newLines,
				Lines:    []DiffLine{},
			}
			continue
		}

		// Parse diff lines
		if currentHunk != nil {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				lineNum := currentHunk.NewStart + len([...]DiffLine{}) - deletions
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    "+",
					Content: line[1:],
					LineNum: lineNum,
				})
				additions++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				lineNum := currentHunk.OldStart + len([...]DiffLine{}) - additions
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    "-",
					Content: line[1:],
					LineNum: lineNum,
				})
				deletions++
			} else if strings.HasPrefix(line, " ") {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    " ",
					Content: line[1:],
				})
			}
		}
	}

	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks, additions, deletions
}

// GetDiffStats returns statistics about changes between branches
func (s *CloneService) GetDiffStats(repoPath string, baseBranch string, headBranch string) (int, int, error) {
	gitCmd, err := getGitPath()
	if err != nil {
		return 0, 0, err
	}
	cmd := exec.Command(gitCmd, "-C", repoPath, "diff", "--shortstat", fmt.Sprintf("%s...%s", baseBranch, headBranch))
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get diff stats: %w", err)
	}

	// Parse output like " 5 files changed, 25 insertions(+), 10 deletions(-)"
	stats := string(output)
	additions := 0
	deletions := 0

	// Simple regex parsing
	addRegex := regexp.MustCompile(`(\d+) insertion`)
	delRegex := regexp.MustCompile(`(\d+) deletion`)

	if matches := addRegex.FindStringSubmatch(stats); matches != nil {
		additions, _ = strconv.Atoi(matches[1])
	}
	if matches := delRegex.FindStringSubmatch(stats); matches != nil {
		deletions, _ = strconv.Atoi(matches[1])
	}

	return additions, deletions, nil
}

package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type CloneService struct {
	reposPath string
	maxAge    time.Duration
}

func NewCloneService(reposPath string, maxAgeHours int) *CloneService {
	return &CloneService{
		reposPath: reposPath,
		maxAge:    time.Duration(maxAgeHours) * time.Hour,
	}
}

// CloneRepository clones a repository using sparse checkout for efficiency
func (s *CloneService) CloneRepository(cloneURL string, isPrivate bool) (string, error) {
	repoID := uuid.New().String()
	repoPath := filepath.Join(s.reposPath, repoID)

	// Create repos directory if it doesn't exist
	if err := os.MkdirAll(s.reposPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create repos directory: %w", err)
	}

	// Clone with sparse checkout
	cmd := exec.Command("git", "clone", "--filter=blob:none", "--sparse", cloneURL, repoPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	log.Info().Str("repo_path", repoPath).Msg("Repository cloned")

	return repoPath, nil
}

// CreateWorktree creates a git worktree for the review
func (s *CloneService) CreateWorktree(repoPath string, branch string, commitSHA string) (string, error) {
	worktreeID := uuid.New().String()
	worktreePath := filepath.Join(s.reposPath, "worktrees", worktreeID)

	// Create worktrees directory if it doesn't exist
	worktreesDir := filepath.Join(s.reposPath, "worktrees")
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	// Create worktree
	cmd := exec.Command("git", "-C", repoPath, "worktree", "add", worktreePath, commitSHA)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create worktree: %w", err)
	}

	log.Info().Str("worktree_path", worktreePath).Str("branch", branch).Msg("Worktree created")

	return worktreePath, nil
}

// RemoveWorktree removes a git worktree
func (s *CloneService) RemoveWorktree(worktreePath string) error {
	// Get the git directory
	gitDir := filepath.Join(filepath.Dir(worktreePath), "..", ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Try alternative location
		gitDir = filepath.Join(filepath.Dir(worktreePath), ".git")
	}

	cmd := exec.Command("git", "-C", gitDir, "worktree", "remove", worktreePath, "--force")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Warn().Err(err).Str("worktree_path", worktreePath).Msg("Failed to remove worktree, cleaning up manually")
		// Fallback to manual cleanup
		return os.RemoveAll(worktreePath)
	}

	log.Info().Str("worktree_path", worktreePath).Msg("Worktree removed")
	return nil
}

// GetChangedFiles returns the list of changed files between two branches
func (s *CloneService) GetChangedFiles(repoPath string, baseBranch string, headBranch string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "diff", "--name-only", fmt.Sprintf("%s...%s", baseBranch, headBranch))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	// Parse output into file list
	files := []string{}
	lines := string(output)
	for _, line := range splitLines(lines) {
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// CleanupOldRepos removes repositories older than maxAge
func (s *CloneService) CleanupOldRepos() error {
	entries, err := os.ReadDir(s.reposPath)
	if err != nil {
		return fmt.Errorf("failed to read repos directory: %w", err)
	}

	now := time.Now()
	removed := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		age := now.Sub(info.ModTime())
		if age > s.maxAge {
			repoPath := filepath.Join(s.reposPath, entry.Name())
			if err := os.RemoveAll(repoPath); err != nil {
				log.Warn().Err(err).Str("repo_path", repoPath).Msg("Failed to remove old repository")
				continue
			}
			removed++
			log.Info().Str("repo_path", repoPath).Dur("age", age).Msg("Removed old repository")
		}
	}

	if removed > 0 {
		log.Info().Int("removed", removed).Msg("Cleaned up old repositories")
	}

	return nil
}

func splitLines(s string) []string {
	var lines []string
	var current []rune

	for _, r := range s {
		if r == '\n' {
			if len(current) > 0 {
				lines = append(lines, string(current))
				current = []rune{}
			}
		} else {
			current = append(current, r)
		}
	}

	if len(current) > 0 {
		lines = append(lines, string(current))
	}

	return lines
}



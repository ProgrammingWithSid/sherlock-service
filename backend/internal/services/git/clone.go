package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var gitPath string

func init() {
	// Find git executable at startup
	var err error
	gitPath, err = exec.LookPath("git")
	if err != nil {
		log.Warn().Err(err).Msg("git executable not found in PATH - git operations will fail")
		gitPath = "" // Will be checked again on first use
	} else {
		log.Info().Str("git_path", gitPath).Msg("Found git executable")
	}
}

// getGitPath returns the path to git executable, checking PATH if needed
func getGitPath() (string, error) {
	if gitPath != "" {
		return gitPath, nil
	}
	path, err := exec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("git executable not found in PATH: %w", err)
	}
	gitPath = path
	return gitPath, nil
}

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
// If token is provided and the repository is private, it will be embedded in the clone URL
func (s *CloneService) CloneRepository(cloneURL string, isPrivate bool, token ...string) (string, error) {
	repoID := uuid.New().String()
	repoPath := filepath.Join(s.reposPath, repoID)

	// Create repos directory if it doesn't exist
	if err := os.MkdirAll(s.reposPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create repos directory: %w", err)
	}

	// If token is provided, embed it in the clone URL
	// For GitHub, we use token if provided (works for both public and private repos)
	authenticatedURL := cloneURL
	hasToken := len(token) > 0 && token[0] != ""
	if hasToken {
		// Embed token in URL: https://x-access-token:TOKEN@github.com/owner/repo.git
		authenticatedURL = s.embedTokenInURL(cloneURL, token[0])
		log.Info().
			Bool("is_private", isPrivate).
			Bool("has_token", hasToken).
			Str("original_url", cloneURL).
			Str("authenticated_url", authenticatedURL).
			Msg("Using authenticated clone URL")
	} else {
		log.Info().
			Bool("is_private", isPrivate).
			Bool("has_token", hasToken).
			Str("clone_url", cloneURL).
			Msg("Cloning without authentication")
	}

	// Clone with sparse checkout
	gitCmd, err := getGitPath()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(gitCmd, "clone", "--filter=blob:none", "--sparse", authenticatedURL, repoPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	log.Info().Str("repo_path", repoPath).Msg("Repository cloned")

	return repoPath, nil
}

// embedTokenInURL embeds a GitHub token in a clone URL
func (s *CloneService) embedTokenInURL(cloneURL, token string) string {
	// Handle different URL formats:
	// https://github.com/owner/repo.git
	// https://github.com/owner/repo
	// git@github.com:owner/repo.git

	if strings.HasPrefix(cloneURL, "https://github.com/") {
		// Replace https://github.com/ with https://x-access-token:TOKEN@github.com/
		return strings.Replace(cloneURL, "https://github.com/", fmt.Sprintf("https://x-access-token:%s@github.com/", token), 1)
	} else if strings.HasPrefix(cloneURL, "http://github.com/") {
		// Handle http URLs (unlikely but possible)
		return strings.Replace(cloneURL, "http://github.com/", fmt.Sprintf("http://x-access-token:%s@github.com/", token), 1)
	}

	// If URL format is not recognized, return as-is
	log.Warn().Str("clone_url", cloneURL).Msg("Unrecognized clone URL format, token not embedded")
	return cloneURL
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
	gitCmd, err := getGitPath()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(gitCmd, "-C", repoPath, "worktree", "add", worktreePath, commitSHA)
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
	// Check if worktree path exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		// Already removed, nothing to do
		return nil
	}

	// Try to find the git directory from the worktree path
	// Worktrees are typically in: repos/worktrees/{id}
	// Git dir should be in: repos/{repo-id}/.git
	worktreeDir := filepath.Dir(worktreePath)
	reposDir := filepath.Dir(worktreeDir)

	// Try to find the parent repo .git directory
	var gitDir string
	entries, err := os.ReadDir(reposDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && entry.Name() != "worktrees" {
				potentialGitDir := filepath.Join(reposDir, entry.Name(), ".git")
				if _, err := os.Stat(potentialGitDir); err == nil {
					gitDir = potentialGitDir
					break
				}
			}
		}
	}

	// If we found a git dir, try to use git worktree remove
	if gitDir != "" {
		gitCmd, err := getGitPath()
		if err != nil {
			log.Warn().Err(err).Msg("git executable not found, falling back to manual cleanup")
		} else {
			cmd := exec.Command(gitCmd, "-C", gitDir, "worktree", "remove", worktreePath, "--force")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err == nil {
				log.Info().Str("worktree_path", worktreePath).Msg("Worktree removed")
				return nil
			}
		}
	}

	// Fallback to manual cleanup
	log.Warn().Str("worktree_path", worktreePath).Msg("Failed to remove worktree via git command, cleaning up manually")
	if err := os.RemoveAll(worktreePath); err != nil {
		return fmt.Errorf("failed to remove worktree directory: %w", err)
	}

	log.Info().Str("worktree_path", worktreePath).Msg("Worktree removed manually")
	return nil
}

// GetChangedFiles returns the list of changed files between two branches
func (s *CloneService) GetChangedFiles(repoPath string, baseBranch string, headBranch string) ([]string, error) {
	gitCmd, err := getGitPath()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(gitCmd, "-C", repoPath, "diff", "--name-only", fmt.Sprintf("%s...%s", baseBranch, headBranch))
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

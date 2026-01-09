package comment

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v57/github"
	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/types"
	"golang.org/x/oauth2"
)

type GitHubCommentService struct {
	client   *github.Client
	token    string
	botLogin *string // Cached bot login name
}

func NewGitHubCommentService(token string) *GitHubCommentService {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &GitHubCommentService{
		client: client,
		token:  token,
	}
}

// PostReview posts a PR review with inline comments
func (s *GitHubCommentService) PostReview(
	owner string,
	repo string,
	prNumber int,
	headSHA string,
	result *types.ReviewResult,
) error {
	ctx := context.Background()

	// Get PR files to validate line numbers
	prFiles, _, err := s.client.PullRequests.ListFiles(ctx, owner, repo, prNumber, &github.ListOptions{PerPage: 100})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get PR files, posting comments without line validation")
		return s.postReviewWithoutValidation(ctx, owner, repo, prNumber, headSHA, result)
	}

	// Build a map of valid line numbers per file
	validLines := make(map[string]map[int]bool)
	for _, file := range prFiles {
		if file.Patch == nil {
			log.Debug().Str("file", *file.Filename).Msg("File has no patch, skipping")
			continue
		}
		lines := s.parseValidLinesFromPatch(*file.Patch)
		validLines[*file.Filename] = lines
		log.Debug().
			Str("file", *file.Filename).
			Int("valid_lines", len(lines)).
			Msg("Parsed valid lines from patch")
	}

	log.Info().
		Int("pr_files", len(prFiles)).
		Int("files_with_patch", len(validLines)).
		Msg("PR files analyzed")

	// Convert comments to GitHub format, filtering invalid line numbers
	comments := make([]*github.DraftReviewComment, 0)
	var skippedFileNotFound []types.ReviewComment
	var skippedInvalidLine []types.ReviewComment

	for _, comment := range result.Comments {
		// Normalize file path for comparison (remove leading ./ if present)
		commentFile := strings.TrimPrefix(comment.File, "./")

		// Try to find matching file (exact match or normalized)
		var fileLines map[int]bool
		var found bool
		if lines, exists := validLines[comment.File]; exists {
			fileLines = lines
			found = true
		} else if lines, exists := validLines[commentFile]; exists {
			fileLines = lines
			found = true
		} else {
			// Try reverse lookup - check if any PR file matches
			for prFile, lines := range validLines {
				if strings.HasSuffix(prFile, comment.File) || strings.HasSuffix(prFile, commentFile) {
					fileLines = lines
					found = true
					break
				}
			}
		}

		if !found {
			// File not in PR, skip inline comment
			skippedFileNotFound = append(skippedFileNotFound, comment)
			continue
		}

		if !fileLines[comment.Line] {
			// Line number not in diff, skip inline comment
			skippedInvalidLine = append(skippedInvalidLine, comment)
			continue
		}

		body := s.formatComment(comment)
		side := "RIGHT" // New lines in the diff
		comments = append(comments, &github.DraftReviewComment{
			Path: &comment.File,
			Line: &comment.Line,
			Side: &side,
			Body: &body,
		})
	}

	skippedComments := len(skippedFileNotFound) + len(skippedInvalidLine)

	log.Info().
		Int("total_comments", len(result.Comments)).
		Int("valid_comments", len(comments)).
		Int("skipped_file_not_found", len(skippedFileNotFound)).
		Int("skipped_invalid_line", len(skippedInvalidLine)).
		Int("total_skipped", skippedComments).
		Msg("Comment validation complete")

	// Create review body with skipped comments in collapsed sections
	body := s.createReviewBody(result, skippedFileNotFound, skippedInvalidLine)

	// Determine review event
	event := s.determineReviewEvent(result)

	// Check if bot is the PR author - GitHub doesn't allow REQUEST_CHANGES on own PRs
	if event == "REQUEST_CHANGES" {
		pr, _, err := s.client.PullRequests.Get(ctx, owner, repo, prNumber)
		if err == nil && pr.User != nil {
			// Get current user (bot) from token (cache it)
			if s.botLogin == nil {
				currentUser, _, err := s.client.Users.Get(ctx, "")
				if err == nil && currentUser.Login != nil {
					s.botLogin = currentUser.Login
					log.Info().Str("bot_user", *s.botLogin).Msg("Detected bot GitHub username")
				}
			}

			if s.botLogin != nil && pr.User.Login != nil && *pr.User.Login == *s.botLogin {
				// Bot is the PR author, change REQUEST_CHANGES to COMMENT
				log.Info().
					Str("pr_author", *pr.User.Login).
					Str("bot_user", *s.botLogin).
					Msg("Bot is PR author, changing REQUEST_CHANGES to COMMENT")
				event = "COMMENT"
			}
		}
	}

	// Create review
	review := &github.PullRequestReviewRequest{
		CommitID: &headSHA,
		Body:     &body,
		Event:    &event,
		Comments: comments,
	}

	_, _, err = s.client.PullRequests.CreateReview(ctx, owner, repo, prNumber, review)
	if err != nil {
		// If error is about requesting changes on own PR, retry with COMMENT
		if strings.Contains(err.Error(), "Can not request changes on your own pull request") {
			log.Warn().Msg("Cannot request changes on own PR, retrying with COMMENT event")
			event = "COMMENT"
			review.Event = &event
			_, _, err = s.client.PullRequests.CreateReview(ctx, owner, repo, prNumber, review)
		}
		if err != nil {
			return fmt.Errorf("failed to create PR review: %w", err)
		}
	}

	log.Info().
		Str("owner", owner).
		Str("repo", repo).
		Int("pr_number", prNumber).
		Str("event", event).
		Int("comments", len(comments)).
		Int("skipped_file_not_found", len(skippedFileNotFound)).
		Int("skipped_invalid_line", len(skippedInvalidLine)).
		Msg("PR review posted")

	return nil
}

// postReviewWithoutValidation posts review without validating line numbers (fallback)
func (s *GitHubCommentService) postReviewWithoutValidation(
	ctx context.Context,
	owner string,
	repo string,
	prNumber int,
	headSHA string,
	result *types.ReviewResult,
) error {
	// Post as general comment instead of inline comments
	body := s.createReviewBody(result, nil, nil)
	body += "\n\n### All Comments\n\n"
	for _, comment := range result.Comments {
		body += fmt.Sprintf("**%s:%d** - %s\n\n", comment.File, comment.Line, comment.Message)
	}

	event := s.determineReviewEvent(result)
	review := &github.PullRequestReviewRequest{
		CommitID: &headSHA,
		Body:     &body,
		Event:    &event,
	}

	_, _, err := s.client.PullRequests.CreateReview(ctx, owner, repo, prNumber, review)
	return err
}

// parseValidLinesFromPatch extracts valid line numbers from a git patch
func (s *GitHubCommentService) parseValidLinesFromPatch(patch string) map[int]bool {
	validLines := make(map[int]bool)
	if patch == "" {
		return validLines
	}

	lines := strings.Split(patch, "\n")
	var currentNewLine int
	var inHunk bool

	for _, line := range lines {
		// Parse hunk header: @@ -old_start,old_count +new_start,new_count @@
		if strings.HasPrefix(line, "@@") {
			// Extract the +new_start,new_count part
			// Format: @@ -old_start,old_count +new_start,new_count @@
			// After splitting by space: ["@@", "-old_start,old_count", "+new_start,new_count", "@@"]
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "+") && len(part) > 1 && !strings.HasPrefix(part, "+++") {
					// Remove the + prefix
					newPart := part[1:]
					// Split by comma to get start and count
					newParts := strings.Split(newPart, ",")
					if len(newParts) > 0 {
						if start, err := strconv.Atoi(newParts[0]); err == nil {
							currentNewLine = start
							inHunk = true
							break
						}
					}
				}
			}
			continue
		}

		if !inHunk {
			continue
		}

		// Process diff lines
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			// Added line - this is a valid line number for comments
			validLines[currentNewLine] = true
			currentNewLine++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			// Deleted line - don't increment new line counter
			// These lines are not valid for inline comments on the new file
			continue
		} else if strings.HasPrefix(line, " ") {
			// Context line (unchanged) - increment counter but don't mark as valid
			// Comments can't be posted on unchanged lines
			currentNewLine++
		}
		// Ignore other lines (file headers, etc.)
	}

	return validLines
}

// PostComment posts a simple comment to a PR
func (s *GitHubCommentService) PostComment(
	owner string,
	repo string,
	prNumber int,
	body string,
) error {
	ctx := context.Background()

	comment := &github.IssueComment{
		Body: &body,
	}

	_, _, err := s.client.Issues.CreateComment(ctx, owner, repo, prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	log.Info().
		Str("owner", owner).
		Str("repo", repo).
		Int("pr_number", prNumber).
		Msg("Comment posted")

	return nil
}

func (s *GitHubCommentService) formatComment(comment types.ReviewComment) string {
	severityEmoji := map[types.Severity]string{
		types.SeverityError:   "🔴",
		types.SeverityWarning: "🟡",
		types.SeverityInfo:    "💡",
	}

	emoji := severityEmoji[comment.Severity]
	if emoji == "" {
		emoji = "💡"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("%s **%s** | `%s`", emoji, strings.ToUpper(string(comment.Severity)), comment.Category))
	parts = append(parts, "")
	parts = append(parts, comment.Message)

	if comment.Fix != "" {
		parts = append(parts, "")
		parts = append(parts, "**Suggested fix:**")
		parts = append(parts, "```")
		parts = append(parts, comment.Fix)
		parts = append(parts, "```")
	}

	return strings.Join(parts, "\n")
}

func (s *GitHubCommentService) determineReviewEvent(result *types.ReviewResult) string {
	switch result.Recommendation {
	case types.RecommendationApprove:
		return "APPROVE"
	case types.RecommendationRequestChanges:
		return "REQUEST_CHANGES"
	default:
		return "COMMENT"
	}
}

func (s *GitHubCommentService) createReviewBody(result *types.ReviewResult, skippedFileNotFound []types.ReviewComment, skippedInvalidLine []types.ReviewComment) string {
	var parts []string

	parts = append(parts, "## 📊 Review Summary")
	parts = append(parts, "")
	parts = append(parts, fmt.Sprintf("**Recommendation:** %s", result.Recommendation))
	parts = append(parts, "")
	parts = append(parts, "| Category | Count |")
	parts = append(parts, "|----------|-------|")
	parts = append(parts, fmt.Sprintf("| 🔴 Errors | %d |", result.Summary.Errors))
	parts = append(parts, fmt.Sprintf("| 🟡 Warnings | %d |", result.Summary.Warnings))
	parts = append(parts, fmt.Sprintf("| 💡 Suggestions | %d |", result.Summary.Suggestions))
	parts = append(parts, "")

	if len(result.Comments) > 0 {
		parts = append(parts, "### Top Issues")
		for i, comment := range result.Comments {
			if i >= 5 {
				break
			}
			parts = append(parts, fmt.Sprintf("- %s in `%s:%d`", comment.Message, comment.File, comment.Line))
		}
		parts = append(parts, "")
	}

	// Add skipped comments in collapsed sections
	totalSkipped := len(skippedFileNotFound) + len(skippedInvalidLine)
	if totalSkipped > 0 {
		parts = append(parts, "---")
		parts = append(parts, "")
		parts = append(parts, fmt.Sprintf("⚠️ **Note:** %d comment(s) could not be posted as inline comments due to platform limitations.", totalSkipped))
		parts = append(parts, "")

		// Outside diff range comments (file not found)
		if len(skippedFileNotFound) > 0 {
			parts = append(parts, "<details>")
			parts = append(parts, "<summary>⚠️ Outside diff range comments ("+fmt.Sprintf("%d", len(skippedFileNotFound))+")</summary>")
			parts = append(parts, "")
			parts = append(parts, "These comments reference files that are not part of this PR's diff:")
			parts = append(parts, "")
			for _, comment := range skippedFileNotFound {
				severityEmoji := s.getSeverityEmoji(comment.Severity)
				// Format: 🔴 ERROR | security | screens/restaurant/menu/uploadMenu/index.tsx:140
				parts = append(parts, fmt.Sprintf("### %s **%s** | `%s` | `%s:%d`", 
					severityEmoji, 
					strings.ToUpper(string(comment.Severity)), 
					comment.Category,
					comment.File, 
					comment.Line))
				parts = append(parts, "")
				// Add the message/summary of the issue
				if comment.Message != "" {
					parts = append(parts, comment.Message)
					parts = append(parts, "")
				}
				// Add suggested fix if available
				if comment.Fix != "" {
					parts = append(parts, "**Suggested fix:**")
					parts = append(parts, "```")
					parts = append(parts, comment.Fix)
					parts = append(parts, "```")
					parts = append(parts, "")
				}
			}
			parts = append(parts, "</details>")
			parts = append(parts, "")
		}

		// Invalid line number comments
		if len(skippedInvalidLine) > 0 {
			parts = append(parts, "<details>")
			parts = append(parts, "<summary>⚠️ Invalid line number comments ("+fmt.Sprintf("%d", len(skippedInvalidLine))+")</summary>")
			parts = append(parts, "")
			parts = append(parts, "These comments reference line numbers that are not in the diff:")
			parts = append(parts, "")
			for _, comment := range skippedInvalidLine {
				severityEmoji := s.getSeverityEmoji(comment.Severity)
				// Format: 🔴 ERROR | security | screens/restaurant/menu/uploadMenu/index.tsx:140
				parts = append(parts, fmt.Sprintf("### %s **%s** | `%s` | `%s:%d`", 
					severityEmoji, 
					strings.ToUpper(string(comment.Severity)), 
					comment.Category,
					comment.File, 
					comment.Line))
				parts = append(parts, "")
				// Add the message/summary of the issue
				if comment.Message != "" {
					parts = append(parts, comment.Message)
					parts = append(parts, "")
				}
				// Add suggested fix if available
				if comment.Fix != "" {
					parts = append(parts, "**Suggested fix:**")
					parts = append(parts, "```")
					parts = append(parts, comment.Fix)
					parts = append(parts, "```")
					parts = append(parts, "")
				}
			}
			parts = append(parts, "</details>")
			parts = append(parts, "")
		}
	}

	parts = append(parts, "---")
	parts = append(parts, "")
	parts = append(parts, "💬 Reply with `@sherlock help` for available commands")

	return strings.Join(parts, "\n")
}

func (s *GitHubCommentService) getSeverityEmoji(severity types.Severity) string {
	severityEmoji := map[types.Severity]string{
		types.SeverityError:   "🔴",
		types.SeverityWarning: "🟡",
		types.SeverityInfo:    "💡",
	}
	if emoji, ok := severityEmoji[severity]; ok {
		return emoji
	}
	return "💡"
}

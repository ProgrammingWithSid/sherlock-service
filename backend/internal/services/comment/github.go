package comment

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v57/github"
	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/types"
	"golang.org/x/oauth2"
)

type GitHubCommentService struct {
	client *github.Client
	token  string
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

	// Convert comments to GitHub format
	comments := make([]*github.DraftReviewComment, 0, len(result.Comments))
	for _, comment := range result.Comments {
		body := s.formatComment(comment)
		comments = append(comments, &github.DraftReviewComment{
			Path: &comment.File,
			Line: &comment.Line,
			Body: &body,
		})
	}

	// Determine review event
	event := s.determineReviewEvent(result)

	// Create review body
	body := s.createReviewBody(result)

	// Create review
	review := &github.PullRequestReviewRequest{
		CommitID: &headSHA,
		Body:      &body,
		Event:     &event,
		Comments:  comments,
	}

	_, _, err := s.client.PullRequests.CreateReview(ctx, owner, repo, prNumber, review)
	if err != nil {
		return fmt.Errorf("failed to create PR review: %w", err)
	}

	log.Info().
		Str("owner", owner).
		Str("repo", repo).
		Int("pr_number", prNumber).
		Str("event", event).
		Int("comments", len(comments)).
		Msg("PR review posted")

	return nil
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

func (s *GitHubCommentService) createReviewBody(result *types.ReviewResult) string {
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

	parts = append(parts, "---")
	parts = append(parts, "💬 Reply with `@sherlock help` for available commands")

	return strings.Join(parts, "\n")
}



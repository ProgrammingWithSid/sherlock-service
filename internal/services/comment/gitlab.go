package comment

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/types"
	"github.com/xanzy/go-gitlab"
)

type GitLabCommentService struct {
	client *gitlab.Client
}

func NewGitLabCommentService(token string) *GitLabCommentService {
	client, err := gitlab.NewClient(token)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create GitLab client")
	}

	return &GitLabCommentService{
		client: client,
	}
}

// PostReview posts a review to a GitLab MR
func (s *GitLabCommentService) PostReview(
	projectID string,
	mrNumber int,
	result *types.ReviewResult,
) error {
	// GitLab doesn't have the same review concept as GitHub
	// We'll post a summary comment instead
	body := s.createReviewBody(result)

	_, _, err := s.client.Notes.CreateMergeRequestNote(projectID, mrNumber, &gitlab.CreateMergeRequestNoteOptions{
		Body: &body,
	})
	if err != nil {
		return fmt.Errorf("failed to post MR note: %w", err)
	}

	// Post inline comments
	for _, comment := range result.Comments {
		if err := s.postInlineComment(projectID, mrNumber, comment); err != nil {
			log.Warn().Err(err).Msg("Failed to post inline comment")
		}
	}

	log.Info().
		Str("project_id", projectID).
		Int("mr_number", mrNumber).
		Int("comments", len(result.Comments)).
		Msg("MR review posted")

	return nil
}

func (s *GitLabCommentService) postInlineComment(
	projectID string,
	mrNumber int,
	comment types.ReviewComment,
) error {
	body := s.formatComment(comment)

	// GitLab inline comments require position information
	// This is simplified - in production, you'd need to get the diff position
	_, _, err := s.client.Discussions.CreateMergeRequestDiscussion(projectID, mrNumber, &gitlab.CreateMergeRequestDiscussionOptions{
		Body: &body,
		Position: &gitlab.PositionOptions{
			BaseSHA: nil, // Would need actual base SHA
			HeadSHA: nil, // Would need actual head SHA
			StartSHA: nil,
			NewPath: &comment.File,
			OldPath: &comment.File,
			PositionType: gitlab.String("text"),
			NewLine: &comment.Line,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create discussion: %w", err)
	}

	return nil
}

func (s *GitLabCommentService) formatComment(comment types.ReviewComment) string {
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

func (s *GitLabCommentService) createReviewBody(result *types.ReviewResult) string {
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


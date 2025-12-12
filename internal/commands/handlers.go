package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sherlock/service/internal/services/review"
	"github.com/sherlock/service/internal/types"
)

// Handler handles command execution
type Handler interface {
	Handle(cmd Command, context CommandContext) (string, error)
}

// CommandContext provides context for command execution
type CommandContext struct {
	OrgID    string
	RepoID   string
	Repo     types.RepoInfo
	PR       types.PRInfo
	Platform types.Platform
}

// ReviewHandler handles @sherlock review command
type ReviewHandler struct {
	enqueueReview func(job *types.ReviewJob, priority int) (string, error)
}

func NewReviewHandler(enqueueReview func(job *types.ReviewJob, priority int) (string, error)) *ReviewHandler {
	return &ReviewHandler{
		enqueueReview: enqueueReview,
	}
}

func (h *ReviewHandler) Handle(cmd Command, ctx CommandContext) (string, error) {
	job := &types.ReviewJob{
		Type:     "full_review",
		Platform: ctx.Platform,
		OrgID:    ctx.OrgID,
		Repo:     ctx.Repo,
		PR:       ctx.PR,
	}

	jobID, err := h.enqueueReview(job, 10) // Higher priority for manual reviews
	if err != nil {
		return "", fmt.Errorf("failed to queue review: %w", err)
	}

	return fmt.Sprintf("✅ Review queued! Job ID: %s\n\nI'll analyze the code changes and post the results shortly.", jobID), nil
}

// ExplainHandler handles @sherlock explain command
type ExplainHandler struct {
	commandService *review.CommandService
	getWorktree    func(repoPath string, branch string) (string, error)
	getConfig      func(orgID string, repoID string) (review.ReviewConfig, error)
}

func NewExplainHandler(
	commandService *review.CommandService,
	getWorktree func(repoPath string, branch string) (string, error),
	getConfig func(orgID string, repoID string) (review.ReviewConfig, error),
) *ExplainHandler {
	return &ExplainHandler{
		commandService: commandService,
		getWorktree:    getWorktree,
		getConfig:      getConfig,
	}
}

func (h *ExplainHandler) Handle(cmd Command, ctx CommandContext) (string, error) {
	if len(cmd.Args) == 0 {
		return "Please specify what to explain. Example: `@sherlock explain src/utils.ts:45`", nil
	}

	// Parse file:line format
	parts := strings.Split(cmd.Args[0], ":")
	if len(parts) != 2 {
		return "Invalid format. Use: `@sherlock explain src/file.ts:45`", nil
	}

	filePath := parts[0]
	lineNumber, err := strconv.Atoi(parts[1])
	if err != nil {
		return "Invalid line number. Use: `@sherlock explain src/file.ts:45`", nil
	}

	// Get worktree and config
	worktreePath, err := h.getWorktree(ctx.Repo.FullName, ctx.PR.HeadSHA)
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	config, err := h.getConfig(ctx.OrgID, ctx.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	// Run explanation
	req := review.ExplainRequest{
		WorktreePath: worktreePath,
		FilePath:     filePath,
		LineNumber:   lineNumber,
		Config:       config,
	}

	result, err := h.commandService.ExplainCode(req)
	if err != nil {
		return "", fmt.Errorf("explanation failed: %w", err)
	}

	response := fmt.Sprintf("## 📖 Code Explanation\n\n**File:** `%s:%d`\n\n", filePath, lineNumber)
	response += fmt.Sprintf("**Summary:** %s\n\n", result.Summary)
	if len(result.Concepts) > 0 {
		response += fmt.Sprintf("**Key Concepts:** %s\n\n", strings.Join(result.Concepts, ", "))
	}
	response += fmt.Sprintf("**Complexity:** %s\n\n", result.Complexity)
	if result.Details != "" {
		response += fmt.Sprintf("**Details:**\n\n%s", result.Details)
	}

	return response, nil
}

// FixHandler handles @sherlock fix command
type FixHandler struct{}

func NewFixHandler() *FixHandler {
	return &FixHandler{}
}

func (h *FixHandler) Handle(cmd Command, ctx CommandContext) (string, error) {
	// TODO: Implement fix generation logic
	return "🔧 Generating fixes...\n\nThis feature is coming soon!", nil
}

// SecurityHandler handles @sherlock security command
type SecurityHandler struct {
	commandService *review.CommandService
	getWorktree    func(repoPath string, branch string) (string, error)
	getConfig      func(orgID string, repoID string) (review.ReviewConfig, error)
}

func NewSecurityHandler(
	commandService *review.CommandService,
	getWorktree func(repoPath string, branch string) (string, error),
	getConfig func(orgID string, repoID string) (review.ReviewConfig, error),
) *SecurityHandler {
	return &SecurityHandler{
		commandService: commandService,
		getWorktree:    getWorktree,
		getConfig:      getConfig,
	}
}

func (h *SecurityHandler) Handle(cmd Command, ctx CommandContext) (string, error) {
	worktreePath, err := h.getWorktree(ctx.Repo.FullName, ctx.PR.HeadSHA)
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	config, err := h.getConfig(ctx.OrgID, ctx.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	req := review.SecurityRequest{
		WorktreePath: worktreePath,
		TargetBranch: ctx.PR.HeadSHA,
		BaseBranch:   ctx.PR.BaseBranch,
		Config:       config,
	}

	result, err := h.commandService.RunSecurityScan(req)
	if err != nil {
		return "", fmt.Errorf("security scan failed: %w", err)
	}

	response := "## 🔒 Security Scan Results\n\n"
	response += fmt.Sprintf("| Severity | Count |\n")
	response += fmt.Sprintf("|----------|-------|\n")
	response += fmt.Sprintf("| 🔴 Critical | %d |\n", result.Summary.Critical)
	response += fmt.Sprintf("| 🟠 High | %d |\n", result.Summary.High)
	response += fmt.Sprintf("| 🟡 Medium | %d |\n", result.Summary.Medium)
	response += fmt.Sprintf("| ⚪ Low | %d |\n\n", result.Summary.Low)

	if len(result.Issues) > 0 {
		response += "### Top Issues\n\n"
		for i, issue := range result.Issues {
			if i >= 10 {
				break
			}
			response += fmt.Sprintf("- **%s** in `%s:%d`: %s\n", issue.Severity, issue.File, issue.Line, issue.Message)
		}
	}

	response += fmt.Sprintf("\n**Recommendation:** %s", result.Recommendation)

	return response, nil
}

// PerformanceHandler handles @sherlock performance command
type PerformanceHandler struct {
	commandService *review.CommandService
	getWorktree    func(repoPath string, branch string) (string, error)
	getConfig      func(orgID string, repoID string) (review.ReviewConfig, error)
}

func NewPerformanceHandler(
	commandService *review.CommandService,
	getWorktree func(repoPath string, branch string) (string, error),
	getConfig func(orgID string, repoID string) (review.ReviewConfig, error),
) *PerformanceHandler {
	return &PerformanceHandler{
		commandService: commandService,
		getWorktree:    getWorktree,
		getConfig:      getConfig,
	}
}

func (h *PerformanceHandler) Handle(cmd Command, ctx CommandContext) (string, error) {
	worktreePath, err := h.getWorktree(ctx.Repo.FullName, ctx.PR.HeadSHA)
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	config, err := h.getConfig(ctx.OrgID, ctx.RepoID)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	req := review.PerformanceRequest{
		WorktreePath: worktreePath,
		TargetBranch: ctx.PR.HeadSHA,
		BaseBranch:   ctx.PR.BaseBranch,
		Config:       config,
	}

	result, err := h.commandService.RunPerformanceAnalysis(req)
	if err != nil {
		return "", fmt.Errorf("performance analysis failed: %w", err)
	}

	response := "## ⚡ Performance Analysis\n\n"
	response += fmt.Sprintf("**Performance Score:** %d/100\n\n", result.Score)
	response += fmt.Sprintf("| Impact | Count |\n")
	response += fmt.Sprintf("|--------|-------|\n")
	response += fmt.Sprintf("| 🔴 High | %d |\n", result.Summary.High)
	response += fmt.Sprintf("| 🟡 Medium | %d |\n", result.Summary.Medium)
	response += fmt.Sprintf("| ⚪ Low | %d |\n\n", result.Summary.Low)

	if len(result.Issues) > 0 {
		response += "### Top Issues\n\n"
		for i, issue := range result.Issues {
			if i >= 10 {
				break
			}
			response += fmt.Sprintf("- **%s** impact in `%s:%d`: %s\n", issue.Impact, issue.File, issue.Line, issue.Message)
		}
	}

	return response, nil
}

// HelpHandler handles @sherlock help command
type HelpHandler struct {
	parser *Parser
}

func NewHelpHandler(parser *Parser) *HelpHandler {
	return &HelpHandler{
		parser: parser,
	}
}

func (h *HelpHandler) Handle(cmd Command, ctx CommandContext) (string, error) {
	return h.parser.GetHelpMessage(), nil
}

// CommandRouter routes commands to appropriate handlers
type CommandRouter struct {
	handlers map[string]Handler
}

func NewCommandRouter(
	reviewHandler *ReviewHandler,
	explainHandler *ExplainHandler,
	fixHandler *FixHandler,
	securityHandler *SecurityHandler,
	performanceHandler *PerformanceHandler,
	helpHandler *HelpHandler,
) *CommandRouter {
	return &CommandRouter{
		handlers: map[string]Handler{
			"review":     reviewHandler,
			"explain":    explainHandler,
			"fix":        fixHandler,
			"security":   securityHandler,
			"performance": performanceHandler,
			"help":       helpHandler,
		},
	}
}

func (r *CommandRouter) Route(cmd Command, ctx CommandContext) (string, error) {
	handler, exists := r.handlers[cmd.Name]
	if !exists {
		return "", fmt.Errorf("no handler for command: %s", cmd.Name)
	}

	return handler.Handle(cmd, ctx)
}

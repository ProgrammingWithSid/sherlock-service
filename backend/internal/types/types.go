package types

import "time"

// Platform represents the git platform
type Platform string

const (
	PlatformGitHub Platform = "github"
	PlatformGitLab Platform = "gitlab"
)

// ReviewStatus represents the status of a review
type ReviewStatus string

const (
	ReviewStatusPending    ReviewStatus = "pending"
	ReviewStatusProcessing ReviewStatus = "processing"
	ReviewStatusCompleted  ReviewStatus = "completed"
	ReviewStatusFailed     ReviewStatus = "failed"
)

// ReviewRecommendation represents the review recommendation
type ReviewRecommendation string

const (
	RecommendationApprove        ReviewRecommendation = "APPROVE"
	RecommendationRequestChanges ReviewRecommendation = "REQUEST_CHANGES"
	RecommendationComment        ReviewRecommendation = "COMMENT"
)

// Severity represents issue severity
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Category represents issue category
type Category string

const (
	CategoryBugs        Category = "bugs"
	CategorySecurity    Category = "security"
	CategoryPerformance Category = "performance"
	CategoryQuality     Category = "code_quality"
	CategoryArchitecture Category = "architecture"
)

// Plan represents subscription plan
type Plan string

const (
	PlanFree       Plan = "free"
	PlanPro        Plan = "pro"
	PlanTeam       Plan = "team"
	PlanEnterprise Plan = "enterprise"
)

// Role represents user role
type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleOrgAdmin   Role = "org_admin"
)

// User represents a user account
type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Name         string    `json:"name" db:"name"`
	Role         Role      `json:"role" db:"role"`
	OrgID        *string   `json:"org_id,omitempty" db:"org_id"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Organization represents an organization
type Organization struct {
	ID                   string     `json:"id" db:"id"`
	Name                 string     `json:"name" db:"name"`
	Slug                 string     `json:"slug" db:"slug"`
	Plan                 Plan       `json:"plan" db:"plan"`
	StripeCustomerID     *string    `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	StripeSubscriptionID *string    `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	PlanActivatedAt      *time.Time `json:"plan_activated_at,omitempty" db:"plan_activated_at"`
	GlobalRules          string     `json:"global_rules" db:"global_rules"` // JSON array of strings
	ClaimToken           *string    `json:"claim_token,omitempty" db:"claim_token"`
	ClaimTokenExpires    *time.Time `json:"claim_token_expires,omitempty" db:"claim_token_expires"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// Repository represents a connected repository
type Repository struct {
	ID         string    `json:"id" db:"id"`
	OrgID      string    `json:"org_id" db:"org_id"`
	Platform   Platform  `json:"platform" db:"platform"`
	ExternalID string    `json:"external_id" db:"external_id"`
	Name       string    `json:"name" db:"name"`
	FullName   string    `json:"full_name" db:"full_name"`
	IsPrivate  bool      `json:"is_private" db:"is_private"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	Config     string    `json:"config" db:"config"` // JSON string
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Review represents a code review
type Review struct {
	ID             string         `json:"id" db:"id"`
	OrgID          string         `json:"org_id" db:"org_id"`
	RepoID         string         `json:"repo_id" db:"repo_id"`
	PRNumber       int            `json:"pr_number" db:"pr_number"`
	HeadSHA        string         `json:"head_sha" db:"head_sha"`
	Status         ReviewStatus   `json:"status" db:"status"`
	Result         string         `json:"result,omitempty" db:"result"` // JSON string
	CommentsPosted int            `json:"comments_posted" db:"comments_posted"`
	DurationMs     *int           `json:"duration_ms,omitempty" db:"duration_ms"`
	AIProvider     string         `json:"ai_provider" db:"ai_provider"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty" db:"completed_at"`
}

// ReviewComment represents an inline comment
type ReviewComment struct {
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Severity Severity `json:"severity"`
	Category Category `json:"category"`
	Message  string   `json:"message"`
	Fix      string   `json:"fix,omitempty"`
}

// ReviewQualityMetrics represents quality metrics for a review
type ReviewQualityMetrics struct {
	Accuracy     float64 `json:"accuracy"`
	Actionability float64 `json:"actionability"`
	Coverage     float64 `json:"coverage"`
	Precision    float64 `json:"precision"`
	Recall       float64 `json:"recall"`
	OverallScore float64 `json:"overallScore"`
	Confidence   float64 `json:"confidence"`
}

// ReviewResult represents the result of a review
type ReviewResult struct {
	Recommendation    ReviewRecommendation   `json:"recommendation"`
	Summary           ReviewSummary          `json:"summary"`
	Comments          []ReviewComment        `json:"comments"`
	QualityMetrics    *ReviewQualityMetrics  `json:"qualityMetrics,omitempty"`
	NamingSuggestions []NamingSuggestion     `json:"namingSuggestions,omitempty"`
	PRTitleSuggestion *PRTitleSuggestion     `json:"prTitleSuggestion,omitempty"`
}

// NamingSuggestion represents a suggestion for renaming an identifier
type NamingSuggestion struct {
	File          string `json:"file"`
	Line          int    `json:"line"`
	CurrentName   string `json:"currentName"`
	SuggestedName string `json:"suggestedName"`
	Type          string `json:"type"` // function, variable, class, interface, constant
	Reason        string `json:"reason"`
	Severity      Severity `json:"severity"`
}

// PRTitleSuggestion represents a suggestion for PR title
type PRTitleSuggestion struct {
	CurrentTitle   string   `json:"currentTitle,omitempty"`
	SuggestedTitle string   `json:"suggestedTitle"`
	Reason         string   `json:"reason"`
	Alternatives   []string `json:"alternatives,omitempty"`
}

// ReviewSummary represents review statistics
type ReviewSummary struct {
	TotalIssues int `json:"total_issues"`
	Errors      int `json:"errors"`
	Warnings    int `json:"warnings"`
	Suggestions int `json:"suggestions"`
}

// ReviewJob represents a queued review job
type ReviewJob struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"` // "full_review" or "incremental_review"
	Platform    Platform `json:"platform"`
	OrgID       string   `json:"org_id"`
	Repo        RepoInfo `json:"repo"`
	PR          PRInfo   `json:"pr"`
	Config      string   `json:"config,omitempty"` // JSON string
	CreatedAt   time.Time `json:"created_at"`
}

// RepoInfo represents repository information in a job
type RepoInfo struct {
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	CloneURL string `json:"clone_url"`
}

// PRInfo represents PR information in a job
type PRInfo struct {
	Number     int    `json:"number"`
	HeadSHA    string `json:"head_sha"`
	BaseBranch string `json:"base_branch"`
}

// CommandJob represents a command job
type CommandJob struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"` // "command"
	Platform  Platform `json:"platform"`
	OrgID     string   `json:"org_id"`
	Repo      RepoInfo `json:"repo"`
	PR        PRInfo   `json:"pr"`
	Comment   CommentInfo `json:"comment"`
	Command   CommandInfo `json:"command"`
	CreatedAt time.Time `json:"created_at"`
}

// CommentInfo represents comment information
type CommentInfo struct {
	ID     int    `json:"id"`
	Body   string `json:"body"`
	Author string `json:"author"`
}

// CommandInfo represents command information
type CommandInfo struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

// UsageLog represents usage tracking
type UsageLog struct {
	ID        string    `json:"id" db:"id"`
	OrgID     string    `json:"org_id" db:"org_id"`
	EventType string    `json:"event_type" db:"event_type"`
	Metadata  string    `json:"metadata" db:"metadata"` // JSON string
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Session represents a user session
type Session struct {
	Token     string     `json:"token" db:"token"`
	UserID    string     `json:"user_id" db:"user_id"`
	Role      string     `json:"role" db:"role"`
	OrgID     *string    `json:"org_id,omitempty" db:"org_id"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
}

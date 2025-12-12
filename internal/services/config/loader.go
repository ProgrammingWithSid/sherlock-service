package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RepoConfig represents the .sherlock.yml configuration
type RepoConfig struct {
	AI         *AIConfig         `yaml:"ai" json:"ai"`
	Review     *ReviewConfig     `yaml:"review" json:"review"`
	Comments   *CommentsConfig   `yaml:"comments" json:"comments"`
	Focus      *FocusConfig      `yaml:"focus" json:"focus"`
	Rules      []string          `yaml:"rules" json:"rules"`
	Ignore     *IgnoreConfig     `yaml:"ignore" json:"ignore"`
	Security   *SecurityConfig   `yaml:"security" json:"security"`
	Performance *PerformanceConfig `yaml:"performance" json:"performance"`
	Labels     *LabelsConfig     `yaml:"labels" json:"labels"`
}

type AIConfig struct {
	Provider string `yaml:"provider" json:"provider"`
	Model    string `yaml:"model" json:"model"`
}

type ReviewConfig struct {
	Enabled     bool `yaml:"enabled" json:"enabled"`
	OnOpen      bool `yaml:"on_open" json:"on_open"`
	OnPush      bool `yaml:"on_push" json:"on_push"`
	Incremental bool `yaml:"incremental" json:"incremental"`
}

type CommentsConfig struct {
	PostSummary bool `yaml:"post_summary" json:"post_summary"`
	PostInline  bool `yaml:"post_inline" json:"post_inline"`
	MaxComments int  `yaml:"max_comments" json:"max_comments"`
}

type FocusConfig struct {
	Bugs         bool `yaml:"bugs" json:"bugs"`
	Security     bool `yaml:"security" json:"security"`
	Performance  bool `yaml:"performance" json:"performance"`
	CodeQuality  bool `yaml:"code_quality" json:"code_quality"`
	Architecture bool `yaml:"architecture" json:"architecture"`
}

type IgnoreConfig struct {
	Paths []string `yaml:"paths" json:"paths"`
	Files []string `yaml:"files" json:"files"`
}

type SecurityConfig struct {
	Enabled          bool   `yaml:"enabled" json:"enabled"`
	BlockOnCritical  bool   `yaml:"block_on_critical" json:"block_on_critical"`
	MinSeverity      string `yaml:"min_severity" json:"min_severity"`
}

type PerformanceConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	MinScore int `yaml:"min_score" json:"min_score"`
}

type LabelsConfig struct {
	AddOnIssues bool   `yaml:"add_on_issues" json:"add_on_issues"`
	Approved    string `yaml:"approved" json:"approved"`
	NeedsReview string `yaml:"needs_review" json:"needs_review"`
	HasIssues   string `yaml:"has_issues" json:"has_issues"`
}

// Loader loads and merges configuration files
type Loader struct {
	defaultConfig *RepoConfig
}

func NewLoader() *Loader {
	return &Loader{
		defaultConfig: getDefaultConfig(),
	}
}

// LoadFromFile loads configuration from .sherlock.yml file
func (l *Loader) LoadFromFile(repoPath string) (*RepoConfig, error) {
	configPath := filepath.Join(repoPath, ".sherlock.yml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return l.defaultConfig, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config RepoConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Merge with defaults
	return l.mergeConfig(l.defaultConfig, &config), nil
}

// LoadFromJSON loads configuration from JSON string (stored in database)
func (l *Loader) LoadFromJSON(jsonStr string) (*RepoConfig, error) {
	if jsonStr == "" || jsonStr == "{}" {
		return l.defaultConfig, nil
	}

	var config RepoConfig
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return l.mergeConfig(l.defaultConfig, &config), nil
}

// MergeConfig merges two configurations, with src taking precedence
func (l *Loader) mergeConfig(dst *RepoConfig, src *RepoConfig) *RepoConfig {
	result := &RepoConfig{}

	// AI config
	if src.AI != nil {
		result.AI = src.AI
	} else {
		result.AI = dst.AI
	}

	// Review config
	if src.Review != nil {
		result.Review = src.Review
	} else {
		result.Review = dst.Review
	}

	// Comments config
	if src.Comments != nil {
		result.Comments = src.Comments
	} else {
		result.Comments = dst.Comments
	}

	// Focus config
	if src.Focus != nil {
		result.Focus = src.Focus
	} else {
		result.Focus = dst.Focus
	}

	// Rules
	if len(src.Rules) > 0 {
		result.Rules = src.Rules
	} else {
		result.Rules = dst.Rules
	}

	// Ignore config
	if src.Ignore != nil {
		result.Ignore = src.Ignore
	} else {
		result.Ignore = dst.Ignore
	}

	// Security config
	if src.Security != nil {
		result.Security = src.Security
	} else {
		result.Security = dst.Security
	}

	// Performance config
	if src.Performance != nil {
		result.Performance = src.Performance
	} else {
		result.Performance = dst.Performance
	}

	// Labels config
	if src.Labels != nil {
		result.Labels = src.Labels
	} else {
		result.Labels = dst.Labels
	}

	return result
}

// ToJSON converts config to JSON string for storage
func (c *RepoConfig) ToJSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	return string(data), nil
}

func getDefaultConfig() *RepoConfig {
	return &RepoConfig{
		AI: &AIConfig{
			// Provider is empty by default - will use env var AI_PROVIDER
			// Model will be set based on provider in worker
		},
		Review: &ReviewConfig{
			Enabled:     true,
			OnOpen:      true,
			OnPush:      true,
			Incremental: true,
		},
		Comments: &CommentsConfig{
			PostSummary: true,
			PostInline:  true,
			MaxComments: 50,
		},
		Focus: &FocusConfig{
			Bugs:         true,
			Security:     true,
			Performance:  true,
			CodeQuality:  true,
			Architecture: true,
		},
		Rules: []string{},
		Ignore: &IgnoreConfig{
			Paths: []string{"**/*.test.ts", "**/*.spec.ts", "**/fixtures/**", "dist/**", "node_modules/**"},
			Files: []string{"package-lock.json", "yarn.lock", "pnpm-lock.yaml"},
		},
		Security: &SecurityConfig{
			Enabled:         true,
			BlockOnCritical: true,
			MinSeverity:     "warning",
		},
		Performance: &PerformanceConfig{
			Enabled:  true,
			MinScore: 70,
		},
		Labels: &LabelsConfig{
			AddOnIssues: true,
			Approved:    "sherlock:approved",
			NeedsReview: "sherlock:needs-review",
			HasIssues:   "sherlock:has-issues",
		},
	}
}

// ShouldIgnoreFile checks if a file should be ignored based on config
func (c *RepoConfig) ShouldIgnoreFile(filePath string) bool {
	if c.Ignore == nil {
		return false
	}

	// Check file patterns
	for _, pattern := range c.Ignore.Files {
		if strings.HasSuffix(filePath, pattern) {
			return true
		}
	}

	// Check path patterns (simplified - would use glob matching in production)
	for _, pattern := range c.Ignore.Paths {
		if strings.Contains(filePath, pattern) {
			return true
		}
	}

	return false
}

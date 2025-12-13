package review

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

type SherlockService struct {
	nodePath string
}

func NewSherlockService(nodePath string) *SherlockService {
	if nodePath == "" {
		nodePath = "node"
	}
	return &SherlockService{
		nodePath: nodePath,
	}
}

// ReviewConfig represents the configuration for code-sherlock
type ReviewConfig struct {
	AIProvider  string                 `json:"aiProvider"`
	OpenAI      *OpenAIConfig          `json:"openai,omitempty"`
	Claude      *ClaudeConfig          `json:"claude,omitempty"`
	GlobalRules []string               `json:"globalRules"`
	Repository  RepositoryConfig       `json:"repository"`
	PR          PRConfig               `json:"pr"`
	GitHub      *GitHubConfig          `json:"github,omitempty"`
	GitLab      *GitLabConfig          `json:"gitlab,omitempty"`
}

type OpenAIConfig struct {
	APIKey string `json:"apiKey"`
	Model  string `json:"model"`
}

type ClaudeConfig struct {
	APIKey string `json:"apiKey"`
	Model  string `json:"model"`
}

type RepositoryConfig struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	BaseBranch string `json:"baseBranch"`
}

type PRConfig struct {
	Number     int    `json:"number"`
	BaseBranch string `json:"baseBranch,omitempty"`
}

type GitHubConfig struct {
	Token string `json:"token"`
}

type GitLabConfig struct {
	Token     string `json:"token"`
	ProjectID string `json:"projectId"`
}

// ReviewRequest represents a review request
type ReviewRequest struct {
	WorktreePath string
	TargetBranch string
	BaseBranch   string
	Config       ReviewConfig
}

// ReviewResult represents the result from code-sherlock
type ReviewResult struct {
	Summary     string         `json:"summary"`
	Stats       ReviewStats    `json:"stats"`
	Comments   []ReviewComment `json:"comments"`
	Recommendation string      `json:"recommendation"`
}

type ReviewStats struct {
	Errors      int `json:"errors"`
	Warnings    int `json:"warnings"`
	Suggestions int `json:"suggestions"`
}

type ReviewComment struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Severity string `json:"severity"`
	Category string `json:"category"`
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
}

// RunReview executes code-sherlock review via Node.js
func (s *SherlockService) RunReview(req ReviewRequest) (*ReviewResult, error) {
	// Create temporary script file
	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("sherlock-review-%s.js", generateTempID()))
	defer os.Remove(scriptPath)

	// Generate script content
	script := s.generateReviewScript(req)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return nil, fmt.Errorf("failed to write review script: %w", err)
	}

	// Create config file
	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("sherlock-config-%s.json", generateTempID()))
	defer os.Remove(configPath)

	// Log key preview BEFORE marshaling (so we don't modify the actual config)
	if req.Config.OpenAI != nil && len(req.Config.OpenAI.APIKey) > 0 {
		keyPreview := fmt.Sprintf("%s... (len: %d)", req.Config.OpenAI.APIKey[:min(10, len(req.Config.OpenAI.APIKey))], len(req.Config.OpenAI.APIKey))
		log.Info().Str("openai_key_preview", keyPreview).Msg("OpenAI API key loaded")
	}

	// Marshal the actual config (with real API keys) to JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Log config for debugging (without exposing API keys) - create a copy
	debugConfig := req.Config
	if debugConfig.Claude != nil {
		debugConfig.Claude = &ClaudeConfig{
			APIKey: "[REDACTED]",
			Model:  debugConfig.Claude.Model,
		}
	}
	if debugConfig.OpenAI != nil {
		debugConfig.OpenAI = &OpenAIConfig{
			APIKey: "[REDACTED]",
			Model:  debugConfig.OpenAI.Model,
		}
	}
	log.Debug().Interface("config", debugConfig).Msg("Review config")

	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	// Debug: Verify the config file has the real API key (first 10 chars only)
	var verifyConfig map[string]interface{}
	if err := json.Unmarshal(configJSON, &verifyConfig); err == nil {
		if openai, ok := verifyConfig["openai"].(map[string]interface{}); ok {
			if apiKey, ok := openai["apiKey"].(string); ok && len(apiKey) > 0 {
				keyPreview := apiKey[:min(10, len(apiKey))]
				log.Info().Str("config_file_key_preview", keyPreview).Int("key_length", len(apiKey)).Msg("Config file API key verification")
			} else {
				log.Error().Msg("Config file API key is empty or missing!")
			}
		}
	}

	// Execute script
	cmd := exec.Command(s.nodePath, scriptPath, configPath, req.WorktreePath, req.TargetBranch, req.BaseBranch)
	cmd.Dir = req.WorktreePath

	// Capture stderr separately for error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		stderrStr := stderr.String()
		log.Error().Err(err).Str("script", scriptPath).Str("stderr", stderrStr).Msg("Review script failed")
		return nil, fmt.Errorf("review execution failed: %w", err)
	}

	// Filter out non-JSON output (code-sherlock might output progress messages)
	outputStr := string(output)
	// Try to find JSON in the output (look for first { and last })
	startIdx := strings.Index(outputStr, "{")
	endIdx := strings.LastIndex(outputStr, "}")
	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		stderrStr := stderr.String()
		log.Error().Str("output", outputStr).Str("stderr", stderrStr).Msg("No valid JSON found in output")
		return nil, fmt.Errorf("failed to parse review result: no valid JSON found in output")
	}

	jsonOutput := outputStr[startIdx : endIdx+1]

	// Parse result
	var result ReviewResult
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		stderrStr := stderr.String()
		log.Error().Err(err).Str("json_output", jsonOutput).Str("stderr", stderrStr).Msg("Failed to parse JSON")
		return nil, fmt.Errorf("failed to parse review result: %w", err)
	}

	return &result, nil
}

func (s *SherlockService) generateReviewScript(req ReviewRequest) string {
	return `
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

async function runReview() {
  try {
    const configPath = process.argv[2];
    const worktreePath = process.argv[3];
    const targetBranch = process.argv[4];
    const baseBranch = process.argv[5];

    const configData = fs.readFileSync(configPath, 'utf8');
    const config = JSON.parse(configData);

    // Debug: log config structure (without API keys) - create DEEP copy to avoid modifying original
    const debugConfig = JSON.parse(JSON.stringify(config));
    if (debugConfig.claude) {
      const keyLen = debugConfig.claude.apiKey ? debugConfig.claude.apiKey.length : 0;
      debugConfig.claude.apiKey = keyLen > 0 ? '[REDACTED-' + keyLen + 'chars]' : '[MISSING]';
    }
    if (debugConfig.openai) {
      const keyLen = debugConfig.openai.apiKey ? debugConfig.openai.apiKey.length : 0;
      const keyPreview = debugConfig.openai.apiKey ? debugConfig.openai.apiKey.substring(0, 10) : '';
      debugConfig.openai.apiKey = keyLen > 0 ? '[REDACTED-' + keyLen + 'chars-starts:' + keyPreview + ']' : '[MISSING]';
      console.error('OpenAI API key check:', {
        exists: !!config.openai?.apiKey,
        length: config.openai?.apiKey?.length || 0,
        startsWith: config.openai?.apiKey?.substring(0, 10) || 'N/A'
      });
    }
    console.error('Review config:', JSON.stringify(debugConfig, null, 2));

    // Load code-sherlock (CommonJS package)
    let PRReviewer;

    // Strategy 1: Try direct import (works for both ES and CommonJS)
    try {
      const sherlock = await import('code-sherlock');
      PRReviewer = sherlock.PRReviewer;
    } catch (importErr) {
      // Strategy 2: Try to find global installation path and use require
      try {
        const os = require('os');
        const homeDir = os.homedir();
        const possiblePaths = [
          path.join(homeDir, '.nvm/versions/node', process.version, 'lib/node_modules/code-sherlock'),
          path.join('/usr/local/lib/node_modules/code-sherlock'),
          path.join('/opt/homebrew/lib/node_modules/code-sherlock'),
          path.join(process.execPath.replace('bin/node', 'lib/node_modules/code-sherlock')),
        ];

        let found = false;
        for (const pkgPath of possiblePaths) {
          try {
            if (fs.existsSync(pkgPath)) {
              // Try dynamic import with file:// URL
              const fileUrl = 'file://' + pkgPath;
              const sherlock = await import(fileUrl);
              PRReviewer = sherlock.PRReviewer;
              found = true;
              break;
            }
          } catch (e) {
            // Try require as fallback (CommonJS)
            try {
              const sherlock = require(pkgPath);
              PRReviewer = sherlock.PRReviewer;
              found = true;
              break;
            } catch (reqErr) {
              // Continue to next path
            }
          }
        }

        if (!found) {
          throw new Error('Could not find code-sherlock in any expected location');
        }
      } catch (resolveErr) {
        console.error('Failed to import code-sherlock:', importErr.message);
        console.error('Resolve error:', resolveErr.message);
        console.error('Please ensure code-sherlock is installed: npm install -g code-sherlock');
        throw new Error('Failed to import code-sherlock: ' + importErr.message);
      }
    }

    // Create reviewer instance
    const reviewer = new PRReviewer(config, worktreePath);

    // Run review
    const result = await reviewer.reviewPR(targetBranch, false, baseBranch || 'main');

    // Convert to JSON format
    const output = {
      summary: result.summary || '',
      stats: {
        errors: result.stats?.errors || 0,
        warnings: result.stats?.warnings || 0,
        suggestions: result.stats?.suggestions || 0,
      },
      comments: (result.comments || []).map(c => ({
        file: c.file || '',
        line: c.line || 0,
        severity: c.severity || 'info',
        category: c.category || 'code_quality',
        message: c.message || '',
        fix: c.fix || '',
      })),
      recommendation: result.recommendation || 'COMMENT',
    };

    console.log(JSON.stringify(output));
  } catch (error) {
    console.error('Review failed:', error.message);
    if (error.stack) {
      console.error(error.stack);
    }
    process.exit(1);
  }
}

runReview();
`
}

func generateTempID() string {
	return fmt.Sprintf("%d", os.Getpid())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

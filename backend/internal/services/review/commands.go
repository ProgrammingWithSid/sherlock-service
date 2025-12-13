package review

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CommandService handles code-sherlock commands
type CommandService struct {
	nodePath string
}

func NewCommandService(nodePath string) *CommandService {
	if nodePath == "" {
		nodePath = "node"
	}
	return &CommandService{
		nodePath: nodePath,
	}
}

// ExplainRequest represents an explain command request
type ExplainRequest struct {
	WorktreePath string
	FilePath     string
	LineNumber   int
	Config       ReviewConfig
}

// ExplainResult represents the explanation result
type ExplainResult struct {
	Summary    string   `json:"summary"`
	Concepts   []string `json:"concepts"`
	Complexity string   `json:"complexity"`
	Details    string   `json:"details"`
}

// ExplainCode explains code at a specific location
func (s *CommandService) ExplainCode(req ExplainRequest) (*ExplainResult, error) {
	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("sherlock-explain-%s.js", generateTempID()))
	defer os.Remove(scriptPath)

	script := s.generateExplainScript(req)
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("sherlock-config-%s.json", generateTempID()))
	defer os.Remove(configPath)

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	cmd := exec.Command(s.nodePath, scriptPath, configPath, req.WorktreePath, req.FilePath, fmt.Sprintf("%d", req.LineNumber))
	cmd.Dir = req.WorktreePath
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("explain execution failed: %w", err)
	}

	var result ExplainResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return &result, nil
}

// SecurityRequest represents a security scan request
type SecurityRequest struct {
	WorktreePath string
	TargetBranch string
	BaseBranch   string
	Config       ReviewConfig
}

// SecurityResult represents security scan results
type SecurityResult struct {
	Issues      []SecurityIssue `json:"issues"`
	Summary     SecuritySummary `json:"summary"`
	Recommendation string      `json:"recommendation"`
}

type SecurityIssue struct {
	Severity  string `json:"severity"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Message   string `json:"message"`
	Fix       string `json:"fix"`
	Category  string `json:"category"`
}

type SecuritySummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

// RunSecurityScan runs a security-focused scan
func (s *CommandService) RunSecurityScan(req SecurityRequest) (*SecurityResult, error) {
	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("sherlock-security-%s.js", generateTempID()))
	defer os.Remove(scriptPath)

	script := s.generateSecurityScript()
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("sherlock-config-%s.json", generateTempID()))
	defer os.Remove(configPath)

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	cmd := exec.Command(s.nodePath, scriptPath, configPath, req.WorktreePath, req.TargetBranch, req.BaseBranch)
	cmd.Dir = req.WorktreePath
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("security scan failed: %w", err)
	}

	var result SecurityResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return &result, nil
}

// PerformanceRequest represents a performance analysis request
type PerformanceRequest struct {
	WorktreePath string
	TargetBranch string
	BaseBranch   string
	Config       ReviewConfig
}

// PerformanceResult represents performance analysis results
type PerformanceResult struct {
	Score   int                 `json:"score"`
	Issues  []PerformanceIssue  `json:"issues"`
	Summary PerformanceSummary  `json:"summary"`
}

type PerformanceIssue struct {
	Impact   string `json:"impact"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
	Fix      string `json:"fix"`
	Category string `json:"category"`
}

type PerformanceSummary struct {
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
}

// RunPerformanceAnalysis runs a performance analysis
func (s *CommandService) RunPerformanceAnalysis(req PerformanceRequest) (*PerformanceResult, error) {
	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("sherlock-performance-%s.js", generateTempID()))
	defer os.Remove(scriptPath)

	script := s.generatePerformanceScript()
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("sherlock-config-%s.json", generateTempID()))
	defer os.Remove(configPath)

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	cmd := exec.Command(s.nodePath, scriptPath, configPath, req.WorktreePath, req.TargetBranch, req.BaseBranch)
	cmd.Dir = req.WorktreePath
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("performance analysis failed: %w", err)
	}

	var result PerformanceResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return &result, nil
}

func (s *CommandService) generateExplainScript(req ExplainRequest) string {
	return `
const fs = require('fs');
const path = require('path');

async function explainCode() {
  try {
    const configPath = process.argv[2];
    const worktreePath = process.argv[3];
    const filePath = process.argv[4];
    const lineNumber = parseInt(process.argv[5]);

    const configData = fs.readFileSync(configPath, 'utf8');
    const config = JSON.parse(configData);

    const { createCodeExplainer } = await import('code-sherlock');
    const explainer = createCodeExplainer();

    const fullPath = path.join(worktreePath, filePath);
    const code = fs.readFileSync(fullPath, 'utf8');
    const lines = code.split('\n');

    const startLine = Math.max(0, lineNumber - 10);
    const endLine = Math.min(lines.length, lineNumber + 10);
    const relevantCode = lines.slice(startLine, endLine).join('\n');

    const explanation = await explainer.explain(relevantCode, filePath, {
      detailLevel: 'detailed',
      audience: 'intermediate',
    });

    const output = {
      summary: explanation.summary || '',
      concepts: explanation.concepts || [],
      complexity: explanation.complexity?.level || 'medium',
      details: explanation.details || '',
    };

    console.log(JSON.stringify(output));
  } catch (error) {
    console.error('Explain failed:', error.message);
    process.exit(1);
  }
}

explainCode();
`
}

func (s *CommandService) generateSecurityScript() string {
	return `
const fs = require('fs');
const path = require('path');

async function runSecurityScan() {
  try {
    const configPath = process.argv[2];
    const worktreePath = process.argv[3];
    const targetBranch = process.argv[4];
    const baseBranch = process.argv[5];

    const configData = fs.readFileSync(configPath, 'utf8');
    const config = JSON.parse(configData);

    const { createSecurityAnalyzer } = await import('code-sherlock');
    const analyzer = createSecurityAnalyzer({
      minSeverity: 'low',
    });

    // Get changed files
    const { execSync } = require('child_process');
    const changedFiles = execSync(
      'git -C ' + worktreePath + ' diff --name-only ' + baseBranch + '...' + targetBranch,
      { encoding: 'utf-8' }
    ).trim().split('\n').filter(f => f);

    const issues = [];
    for (const file of changedFiles) {
      const fullPath = path.join(worktreePath, file);
      if (fs.existsSync(fullPath)) {
        const content = fs.readFileSync(fullPath, 'utf-8');
        const result = analyzer.analyze(content, file);
        if (result.issues) {
          issues.push(...result.issues.map(i => ({
            severity: i.severity || 'medium',
            file: file,
            line: i.line || 0,
            message: i.message || '',
            fix: i.fix || '',
            category: i.category || 'security',
          })));
        }
      }
    }

    const summary = {
      critical: issues.filter(i => i.severity === 'critical').length,
      high: issues.filter(i => i.severity === 'high').length,
      medium: issues.filter(i => i.severity === 'medium').length,
      low: issues.filter(i => i.severity === 'low').length,
    };

    const output = {
      issues: issues,
      summary: summary,
      recommendation: summary.critical > 0 ? 'BLOCK' : summary.high > 0 ? 'REQUEST_CHANGES' : 'APPROVE',
    };

    console.log(JSON.stringify(output));
  } catch (error) {
    console.error('Security scan failed:', error.message);
    process.exit(1);
  }
}

runSecurityScan();
`
}

func (s *CommandService) generatePerformanceScript() string {
	return `
const fs = require('fs');
const path = require('path');

async function runPerformanceAnalysis() {
  try {
    const configPath = process.argv[2];
    const worktreePath = process.argv[3];
    const targetBranch = process.argv[4];
    const baseBranch = process.argv[5];

    const configData = fs.readFileSync(configPath, 'utf8');
    const config = JSON.parse(configData);

    const { createPerformanceAnalyzer } = await import('code-sherlock');
    const analyzer = createPerformanceAnalyzer({
      focus: 'all',
      minImpact: 'low',
    });

    const { execSync } = require('child_process');
    const changedFiles = execSync(
      'git -C ' + worktreePath + ' diff --name-only ' + baseBranch + '...' + targetBranch,
      { encoding: 'utf-8' }
    ).trim().split('\n').filter(f => f);

    const issues = [];
    let totalScore = 0;
    let fileCount = 0;

    for (const file of changedFiles) {
      const fullPath = path.join(worktreePath, file);
      if (fs.existsSync(fullPath)) {
        const content = fs.readFileSync(fullPath, 'utf-8');
        const result = analyzer.analyze(content);
        if (result.score !== undefined) {
          totalScore += result.score;
          fileCount++;
        }
        if (result.issues) {
          issues.push(...result.issues.map(i => ({
            impact: i.impact || 'medium',
            file: file,
            line: i.line || 0,
            message: i.message || '',
            fix: i.fix || '',
            category: i.category || 'performance',
          })));
        }
      }
    }

    const avgScore = fileCount > 0 ? Math.round(totalScore / fileCount) : 100;
    const summary = {
      high: issues.filter(i => i.impact === 'high').length,
      medium: issues.filter(i => i.impact === 'medium').length,
      low: issues.filter(i => i.impact === 'low').length,
    };

    const output = {
      score: avgScore,
      issues: issues,
      summary: summary,
    };

    console.log(JSON.stringify(output));
  } catch (error) {
    console.error('Performance analysis failed:', error.message);
    process.exit(1);
  }
}

runPerformanceAnalysis();
`
}

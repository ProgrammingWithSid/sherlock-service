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
	Issues         []SecurityIssue `json:"issues"`
	Summary        SecuritySummary `json:"summary"`
	Recommendation string          `json:"recommendation"`
}

type SecurityIssue struct {
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
	Fix      string `json:"fix"`
	Category string `json:"category"`
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
	Score   int                `json:"score"`
	Issues  []PerformanceIssue `json:"issues"`
	Summary PerformanceSummary `json:"summary"`
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

// FixRequest represents a fix generation request
type FixRequest struct {
	WorktreePath string
	TargetBranch string
	BaseBranch   string
	Config       ReviewConfig
	FileFilter   []string `json:"fileFilter,omitempty"` // Optional: only generate fixes for specific files
}

// FixResult represents fix generation results
type FixResult struct {
	Suggestions []FixSuggestion `json:"suggestions"`
	Summary     FixSummary      `json:"summary"`
}

type FixSuggestion struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Description string `json:"description"`
	Fix         string `json:"fix"`
	Confidence  string `json:"confidence"` // "high", "medium", "low"
	Explanation string `json:"explanation"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
}

type FixSummary struct {
	Total            int `json:"total"`
	HighConfidence   int `json:"highConfidence"`
	MediumConfidence int `json:"mediumConfidence"`
	LowConfidence    int `json:"lowConfidence"`
	AutoApplicable   int `json:"autoApplicable"`
}

// TestRequest represents a test generation request
type TestRequest struct {
	WorktreePath string
	FilePath     string
	Config       ReviewConfig
	Framework    string `json:"framework,omitempty"` // "jest", "mocha", "pytest", etc.
}

// TestResult represents test generation results
type TestResult struct {
	Tests     []GeneratedTest `json:"tests"`
	Framework string          `json:"framework"`
	Summary   string          `json:"summary"`
	Coverage  TestCoverage    `json:"coverage"`
}

type GeneratedTest struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Type        string `json:"type"` // "unit", "integration", "e2e"
}

type TestCoverage struct {
	Functions int `json:"functions"`
	Branches  int `json:"branches"`
	Lines     int `json:"lines"`
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

    // Use enhanced SAST integration with all available tools
    const { createSASTIntegration, createSecurityAnalyzer } = await import('code-sherlock');

    // Determine which SAST tools to use based on project type
    const { execSync } = require('child_process');
    const packageJsonPath = path.join(worktreePath, 'package.json');
    const requirementsPath = path.join(worktreePath, 'requirements.txt');
    const goModPath = path.join(worktreePath, 'go.mod');
    const cargoTomlPath = path.join(worktreePath, 'Cargo.toml');
    const gemfilePath = path.join(worktreePath, 'Gemfile');
    const mixPath = path.join(worktreePath, 'mix.exs');

    const tools = [];
    if (fs.existsSync(packageJsonPath)) {
      tools.push('npm-audit', 'snyk');
    }
    if (fs.existsSync(requirementsPath) || fs.existsSync(path.join(worktreePath, 'pyproject.toml'))) {
      tools.push('bandit', 'safety', 'pip-audit');
    }
    if (fs.existsSync(goModPath)) {
      tools.push('gosec');
    }
    if (fs.existsSync(cargoTomlPath)) {
      tools.push('cargo-audit');
    }
    if (fs.existsSync(gemfilePath)) {
      tools.push('brakeman', 'bundler-audit');
    }
    if (fs.existsSync(mixPath)) {
      tools.push('mix-audit');
    }
    // Always include general-purpose tools
    tools.push('semgrep', 'trivy', 'owasp-dependency-check');

    // Remove duplicates
    const uniqueTools = [...new Set(tools)];

    // Create SAST integration with detected tools
    const sastIntegration = createSASTIntegration({
      enabled: true,
      tools: uniqueTools,
      workingDir: worktreePath,
      minSeverity: 'low',
    });

    // Also use pattern-based security analyzer as fallback
    const securityAnalyzer = createSecurityAnalyzer({
      minSeverity: 'low',
    });

    // Get changed files
    const changedFiles = execSync(
      'git -C ' + worktreePath + ' diff --name-only ' + baseBranch + '...' + targetBranch,
      { encoding: 'utf-8' }
    ).trim().split('\n').filter(f => f);

    // Prepare files for SAST analysis
    const filesToAnalyze = [];
    for (const file of changedFiles) {
      const fullPath = path.join(worktreePath, file);
      if (fs.existsSync(fullPath)) {
        const content = fs.readFileSync(fullPath, 'utf-8');
        filesToAnalyze.push({
          path: fullPath,
          content: content,
        });
      }
    }

    // Run SAST analysis
    const sastResult = await sastIntegration.analyze(filesToAnalyze);

    // Also run pattern-based analysis for files not covered by SAST tools
    const patternIssues = [];
    for (const file of filesToAnalyze) {
      const relativePath = path.relative(worktreePath, file.path);
      const result = securityAnalyzer.analyze(file.content, relativePath);
      if (result.issues) {
        patternIssues.push(...result.issues);
      }
    }

    // Combine SAST and pattern-based issues
    const allIssues = [
      ...sastResult.issues.map(i => ({
        severity: i.severity || 'medium',
        file: path.relative(worktreePath, i.file),
        line: i.line || 0,
        message: i.message || '',
        fix: i.fix || '',
        category: i.type || 'security',
        tool: i.tool || 'sast',
      })),
      ...patternIssues.map(i => ({
        severity: i.severity || 'medium',
        file: i.file || '',
        line: i.line || 0,
        message: i.message || '',
        fix: i.fix || '',
        category: i.category || 'security',
        tool: 'pattern-analyzer',
      })),
    ];

    const summary = {
      critical: allIssues.filter(i => i.severity === 'error' || i.severity === 'critical').length,
      high: allIssues.filter(i => i.severity === 'error').length,
      medium: allIssues.filter(i => i.severity === 'warning').length,
      low: allIssues.filter(i => i.severity === 'info' || i.severity === 'suggestion').length,
    };

    const output = {
      issues: allIssues,
      summary: summary,
      recommendation: summary.critical > 0 ? 'BLOCK' : summary.high > 0 ? 'REQUEST_CHANGES' : 'APPROVE',
      toolsUsed: sastResult.toolsUsed || [],
      filesAnalyzed: sastResult.filesAnalyzed || 0,
    };

    console.log(JSON.stringify(output));
  } catch (error) {
    console.error('Security scan failed:', error.message);
    if (error.stack) {
      console.error(error.stack);
    }
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

// RunFixGeneration generates fixes for review comments
func (s *CommandService) RunFixGeneration(req FixRequest) (*FixResult, error) {
	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("sherlock-fix-%s.js", generateTempID()))
	defer os.Remove(scriptPath)

	script := s.generateFixScript()
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

	// Prepare file filter JSON if provided
	fileFilterJSON := "[]"
	if len(req.FileFilter) > 0 {
		filterJSON, err := json.Marshal(req.FileFilter)
		if err == nil {
			fileFilterJSON = string(filterJSON)
		}
	}

	cmd := exec.Command(s.nodePath, scriptPath, configPath, req.WorktreePath, req.TargetBranch, req.BaseBranch, fileFilterJSON)
	cmd.Dir = req.WorktreePath
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("fix generation failed: %w", err)
	}

	var result FixResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return &result, nil
}

// GenerateTests generates tests for a file
func (s *CommandService) GenerateTests(req TestRequest) (*TestResult, error) {
	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("sherlock-test-%s.js", generateTempID()))
	defer os.Remove(scriptPath)

	script := s.generateTestScript()
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

	framework := req.Framework
	if framework == "" {
		framework = "auto" // Auto-detect
	}

	cmd := exec.Command(s.nodePath, scriptPath, configPath, req.WorktreePath, req.FilePath, framework)
	cmd.Dir = req.WorktreePath
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("test generation failed: %w", err)
	}

	var result TestResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return &result, nil
}

func (s *CommandService) generateFixScript() string {
	return `
const fs = require('fs');
const path = require('path');

async function generateFixes() {
  try {
    const configPath = process.argv[2];
    const worktreePath = process.argv[3];
    const targetBranch = process.argv[4];
    const baseBranch = process.argv[5];
    const fileFilterJSON = process.argv[6] || '[]';

    const configData = fs.readFileSync(configPath, 'utf8');
    const config = JSON.parse(configData);
    const fileFilter = JSON.parse(fileFilterJSON);

    const { createAutoFix } = await import('code-sherlock');
    const autoFix = createAutoFix({
      generatorOptions: {
        model: config.aiProvider === 'claude' ? 'claude' : 'openai',
        apiKey: config.aiProvider === 'claude' ? config.claude?.apiKey : config.openai?.apiKey,
        maxFixesPerFile: 10,
        includeAlternatives: true,
      },
    });

    // Get changed files
    const { execSync } = require('child_process');
    const changedFiles = execSync(
      'git -C ' + worktreePath + ' diff --name-only ' + baseBranch + '...' + targetBranch,
      { encoding: 'utf-8' }
    ).trim().split('\n').filter(f => f);

    // Filter files if fileFilter is provided
    const filesToAnalyze = fileFilter.length > 0
      ? changedFiles.filter(f => fileFilter.includes(f))
      : changedFiles;

    // Get review comments by running a quick review or fetching from previous review
    // For now, we'll run a quick review to get comments, or use pattern-based detection
    const reviewComments = [];

    // Try to get comments from a previous review if available
    // Otherwise, run a quick review or use pattern detection
    try {
      // Run a quick review to get actual comments
      const { PRReviewer } = await import('code-sherlock');
      const quickReviewer = new PRReviewer(config, worktreePath);
      const reviewResult = await quickReviewer.reviewPR(targetBranch, false, baseBranch || 'main');

      // Filter comments for files we're analyzing
      if (reviewResult.comments) {
        reviewResult.comments.forEach(comment => {
          if (filesToAnalyze.length === 0 || filesToAnalyze.some(f => comment.file.includes(f) || comment.file.endsWith(f))) {
            reviewComments.push({
              file: comment.file,
              line: comment.line,
              body: comment.body || comment.message || '',
              severity: comment.severity || 'warning',
              category: comment.category || 'code_quality',
              fix: comment.fix || '',
            });
          }
        });
      }
    } catch (reviewErr) {
      // Fallback: Use pattern-based detection for common issues
      console.error('Quick review failed, using pattern detection:', reviewErr.message);
      for (const file of filesToAnalyze) {
        const fullPath = path.join(worktreePath, file);
        if (fs.existsSync(fullPath)) {
          const content = fs.readFileSync(fullPath, 'utf-8');
          const lines = content.split('\n');

          lines.forEach((line, index) => {
            if (line.includes('TODO') || line.includes('FIXME') || line.includes('XXX')) {
              reviewComments.push({
                file: file,
                line: index + 1,
                body: 'Code comment suggests issue: ' + line.trim(),
                severity: 'warning',
                category: 'code_quality',
              });
            }
          });
        }
      }
    }

    // Generate fixes
    const fixContext = {
      comments: reviewComments,
      files: filesToAnalyze.map(file => ({
        path: file,
        content: fs.readFileSync(path.join(worktreePath, file), 'utf-8'),
      })),
    };

    const fixResult = autoFix.generateFixes(fixContext);

    // Convert to output format
    const suggestions = fixResult.suggestions.map(s => ({
      file: s.fix.filePath,
      line: s.fix.startLine,
      description: s.fix.description,
      fix: s.fix.fixedCode,
      confidence: s.fix.confidence,
      explanation: s.explanation,
      category: s.fix.category,
      severity: s.fix.severity,
    }));

    const summary = {
      total: suggestions.length,
      highConfidence: suggestions.filter(s => s.confidence === 'high').length,
      mediumConfidence: suggestions.filter(s => s.confidence === 'medium').length,
      lowConfidence: suggestions.filter(s => s.confidence === 'low').length,
      autoApplicable: suggestions.filter((s, idx) => {
        const fix = fixResult.suggestions[idx]?.fix;
        return fix && fix.isAutoApplicable;
      }).length,
    };

    const output = {
      suggestions: suggestions,
      summary: summary,
    };

    console.log(JSON.stringify(output));
  } catch (error) {
    console.error('Fix generation failed:', error.message);
    if (error.stack) {
      console.error(error.stack);
    }
    process.exit(1);
  }
}

generateFixes();
`
}

func (s *CommandService) generateTestScript() string {
	return `
const fs = require('fs');
const path = require('path');

async function generateTests() {
  try {
    const configPath = process.argv[2];
    const worktreePath = process.argv[3];
    const filePath = process.argv[4];
    const framework = process.argv[5] || 'auto';

    const configData = fs.readFileSync(configPath, 'utf8');
    const config = JSON.parse(configData);

    const { createTestGenerator } = await import('code-sherlock');
    const testGenerator = createTestGenerator();

    const fullPath = path.join(worktreePath, filePath);
    if (!fs.existsSync(fullPath)) {
      throw new Error('File not found: ' + filePath);
    }

    const code = fs.readFileSync(fullPath, 'utf-8');

    // Detect framework from file extension and project structure
    let detectedFramework = framework;
    if (framework === 'auto') {
      if (filePath.endsWith('.ts') || filePath.endsWith('.tsx')) {
        const packageJsonPath = path.join(worktreePath, 'package.json');
        if (fs.existsSync(packageJsonPath)) {
          const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf-8'));
          if (packageJson.devDependencies?.jest) {
            detectedFramework = 'jest';
          } else if (packageJson.devDependencies?.mocha) {
            detectedFramework = 'mocha';
          } else {
            detectedFramework = 'jest'; // Default for TypeScript
          }
        }
      } else if (filePath.endsWith('.py')) {
        detectedFramework = 'pytest';
      } else if (filePath.endsWith('.go')) {
        detectedFramework = 'testing'; // Go testing package
      } else {
        detectedFramework = 'jest'; // Default
      }
    }

    const testResult = await testGenerator.generateTests(code, {
      framework: detectedFramework,
      language: filePath.split('.').pop(),
      detailLevel: 'comprehensive',
      includeEdgeCases: true,
    });

    // Convert to output format
    const tests = testResult.tests.map(t => ({
      name: t.name,
      code: t.code,
      description: t.description,
      type: t.type || 'unit',
    }));

    const output = {
      tests: tests,
      framework: detectedFramework,
      summary: testResult.summary || 'Generated ' + tests.length + ' test(s)',
      coverage: {
        functions: testResult.coverage?.functions || 0,
        branches: testResult.coverage?.branches || 0,
        lines: testResult.coverage?.lines || 0,
      },
    };

    console.log(JSON.stringify(output));
  } catch (error) {
    console.error('Test generation failed:', error.message);
    if (error.stack) {
      console.error(error.stack);
    }
    process.exit(1);
  }
}

generateTests();
`
}

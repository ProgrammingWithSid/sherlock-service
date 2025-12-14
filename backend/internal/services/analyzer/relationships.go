package analyzer

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/services/indexer"
)

// RelationshipIssue represents an issue found in code relationships
type RelationshipIssue struct {
	Type        string // "breaking_change", "missing_update", "inconsistency", "dependency_issue"
	Severity    string // "error", "warning", "info"
	File        string
	Line        int
	Message     string
	RelatedFiles []string
	Fix         string
}

// RelationshipAnalyzer analyzes relationships between changed files
type RelationshipAnalyzer struct {
	indexer *indexer.CodebaseIndexer
}

// NewRelationshipAnalyzer creates a new relationship analyzer
func NewRelationshipAnalyzer(indexer *indexer.CodebaseIndexer) *RelationshipAnalyzer {
	return &RelationshipAnalyzer{
		indexer: indexer,
	}
}

// AnalyzeRelationships analyzes relationships between changed files
func (ra *RelationshipAnalyzer) AnalyzeRelationships(
	ctx context.Context,
	repoID string,
	changedFiles []string,
) ([]RelationshipIssue, error) {
	log.Info().
		Str("repo_id", repoID).
		Int("changed_files", len(changedFiles)).
		Msg("Analyzing code relationships")

	issues := make([]RelationshipIssue, 0)

	// TODO: Implement relationship analysis
	// Will check for:
	// 1. Breaking changes (removed exports, changed signatures)
	// 2. Missing updates (related files that should be updated)
	// 3. Inconsistencies (conflicting changes)
	// 4. Dependency issues (circular dependencies, missing imports)

	// Example: Check for breaking changes
	for _, file := range changedFiles {
		// Check if this file exports symbols used elsewhere
		related, err := ra.indexer.GetRelatedCode(ctx, repoID, file, "")
		if err != nil {
			log.Warn().Err(err).Str("file", file).Msg("Failed to get related code")
			continue
		}

		if len(related) > 0 {
			// Found dependencies - check if changes break them
			issues = append(issues, RelationshipIssue{
				Type:        "dependency_check",
				Severity:    "info",
				File:        file,
				Message:     fmt.Sprintf("File has %d dependent symbols that may need review", len(related)),
				RelatedFiles: extractFilePaths(related),
			})
		}
	}

	log.Info().
		Int("issues_found", len(issues)).
		Msg("Relationship analysis completed")

	return issues, nil
}

// DetectBreakingChanges detects breaking changes in the codebase
func (ra *RelationshipAnalyzer) DetectBreakingChanges(
	ctx context.Context,
	repoID string,
	changedFiles []string,
) ([]RelationshipIssue, error) {
	issues := make([]RelationshipIssue, 0)

	// TODO: Implement breaking change detection
	// Will check for:
	// 1. Removed public APIs
	// 2. Changed function signatures
	// 3. Removed exports
	// 4. Changed type definitions

	return issues, nil
}

// FindRelatedFiles finds files that might need updates based on changes
func (ra *RelationshipAnalyzer) FindRelatedFiles(
	ctx context.Context,
	repoID string,
	changedFiles []string,
) ([]string, error) {
	relatedFiles := make(map[string]bool)

	for _, file := range changedFiles {
		related, err := ra.indexer.GetRelatedCode(ctx, repoID, file, "")
		if err != nil {
			continue
		}

		for _, symbol := range related {
			if symbol.FilePath != file {
				relatedFiles[symbol.FilePath] = true
			}
		}
	}

	result := make([]string, 0, len(relatedFiles))
	for file := range relatedFiles {
		result = append(result, file)
	}

	return result, nil
}

// extractFilePaths extracts unique file paths from symbols
func extractFilePaths(symbols []indexer.CodeSymbol) []string {
	files := make(map[string]bool)
	for _, symbol := range symbols {
		files[symbol.FilePath] = true
	}

	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}

	return result
}

package indexer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// EnhancedIndexer provides enhanced indexing using chunkyyy
type EnhancedIndexer struct {
	indexer  *CodebaseIndexer
	chunkyyy *ChunkyyyService
}

// NewEnhancedIndexer creates an enhanced indexer with chunkyyy integration
func NewEnhancedIndexer(indexer *CodebaseIndexer) *EnhancedIndexer {
	return &EnhancedIndexer{
		indexer:  indexer,
		chunkyyy: indexer.chunkyyy,
	}
}

// IndexFileWithDependencies indexes a file and its dependencies
func (ei *EnhancedIndexer) IndexFileWithDependencies(ctx context.Context, repoID string, filePath string) error {
	log.Info().Str("file", filePath).Msg("Indexing file with dependencies")

	// Extract symbols from file
	symbols, err := ei.chunkyyy.ExtractSymbols(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to extract symbols: %w", err)
	}

	// Extract dependencies
	deps, err := ei.chunkyyy.ExtractDependencies(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to extract dependencies: %w", err)
	}

	log.Info().
		Str("file", filePath).
		Int("symbols", len(symbols)).
		Int("dependencies", len(deps)).
		Msg("File indexed with dependencies")

	// Store symbols (with dependency information)
	for _, symbol := range symbols {
		symbol.RepoID = repoID
		// Add dependency information to symbol
		for _, dep := range deps {
			symbol.Dependencies = append(symbol.Dependencies, dep.Name)
		}
		_ = ei.indexer.storeSymbol(ctx, &symbol)
	}

	return nil
}

// FindDependentFiles finds files that depend on a given symbol
func (ei *EnhancedIndexer) FindDependentFiles(ctx context.Context, repoID string, symbolName string) ([]string, error) {
	// TODO: Query database for files that import/use this symbol
	// For now, return empty
	return []string{}, nil
}

// GetSymbolDependencies gets all dependencies of a symbol
func (ei *EnhancedIndexer) GetSymbolDependencies(ctx context.Context, filePath string, symbolName string) ([]Dependency, error) {
	symbols, err := ei.chunkyyy.ExtractSymbols(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Find the symbol
	for _, symbol := range symbols {
		if symbol.SymbolName == symbolName {
			// Convert []string dependencies to []Dependency
			deps := make([]Dependency, 0, len(symbol.Dependencies))
			for _, depName := range symbol.Dependencies {
				deps = append(deps, Dependency{
					Name: depName,
				})
			}
			return deps, nil
		}
	}

	return []Dependency{}, nil
}

// BuildDependencyGraph builds a dependency graph for the repository
func (ei *EnhancedIndexer) BuildDependencyGraph(ctx context.Context, repoID string, repoPath string) error {
	log.Info().Str("repo_id", repoID).Msg("Building dependency graph")

	// Find all code files
	files, err := ei.indexer.findCodeFiles(repoPath)
	if err != nil {
		return err
	}

	// Index all files with dependencies
	for _, filePath := range files {
		if err := ei.IndexFileWithDependencies(ctx, repoID, filePath); err != nil {
			log.Warn().Err(err).Str("file", filePath).Msg("Failed to index file")
			continue
		}
	}

	log.Info().
		Str("repo_id", repoID).
		Int("files", len(files)).
		Msg("Dependency graph built")

	return nil
}

// GetChunkHashForRange gets the chunkyyy hash for a specific line range
func (ei *EnhancedIndexer) GetChunkHashForRange(ctx context.Context, filePath string, startLine int, endLine int) (string, error) {
	return ei.chunkyyy.GetChunkHash(ctx, filePath, startLine, endLine)
}

// IsCodeFile checks if a file is a code file that chunkyyy can parse
func IsCodeFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	supportedExts := []string{".ts", ".tsx", ".js", ".jsx", ".vue"}
	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

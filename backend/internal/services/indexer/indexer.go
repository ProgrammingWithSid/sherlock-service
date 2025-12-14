package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/database"
	"github.com/sherlock/service/internal/types"
)

// CodeSymbol represents a code symbol (function, class, variable, etc.)
type CodeSymbol struct {
	ID           string
	RepoID       string
	FilePath     string
	SymbolName   string
	SymbolType   string // "function", "class", "method", "variable", "interface", "type"
	LineStart    int
	LineEnd      int
	Signature    string
	Dependencies []string // Array of dependent symbols
	CreatedAt    time.Time
}

// CodebaseIndexer indexes codebase for context-aware analysis
type CodebaseIndexer struct {
	db            *database.DB
	chunkyyy      *ChunkyyyService
	repoPath      string
}

// NewCodebaseIndexer creates a new codebase indexer
func NewCodebaseIndexer(db *database.DB, repoPath string, nodePath string) *CodebaseIndexer {
	return &CodebaseIndexer{
		db:       db,
		chunkyyy: NewChunkyyyService(repoPath, nodePath),
		repoPath: repoPath,
	}
}

// IndexRepository indexes all code symbols in a repository using chunkyyy
func (ci *CodebaseIndexer) IndexRepository(ctx context.Context, repoID string, repoPath string) error {
	log.Info().Str("repo_id", repoID).Str("repo_path", repoPath).Msg("Starting repository indexing with chunkyyy")

	// Update repo path
	ci.repoPath = repoPath
	ci.chunkyyy = NewChunkyyyService(repoPath, "")

	// Find all code files (TypeScript, JavaScript, etc.)
	codeFiles, err := ci.findCodeFiles(repoPath)
	if err != nil {
		return fmt.Errorf("failed to find code files: %w", err)
	}

	log.Info().
		Str("repo_id", repoID).
		Int("files", len(codeFiles)).
		Msg("Found code files to index")

	// Index each file
	totalSymbols := 0
	for _, filePath := range codeFiles {
		symbols, err := ci.chunkyyy.ExtractSymbols(ctx, filePath)
		if err != nil {
			log.Warn().Err(err).Str("file", filePath).Msg("Failed to extract symbols, skipping")
			continue
		}

		// Store symbols in database
		for _, symbol := range symbols {
			symbol.RepoID = repoID
			if err := ci.storeSymbol(ctx, &symbol); err != nil {
				log.Warn().Err(err).Str("symbol", symbol.SymbolName).Msg("Failed to store symbol")
				continue
			}
			totalSymbols++
		}
	}

	log.Info().
		Str("repo_id", repoID).
		Int("symbols", totalSymbols).
		Msg("Repository indexing completed")

	return nil
}

// findCodeFiles finds all code files in the repository
func (ci *CodebaseIndexer) findCodeFiles(repoPath string) ([]string, error) {
	// Supported extensions for chunkyyy
	extensions := []string{".ts", ".tsx", ".js", ".jsx", ".vue"}

	var files []string
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip node_modules, .git, etc.
			if info.Name() == "node_modules" || info.Name() == ".git" || info.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		for _, supportedExt := range extensions {
			if ext == supportedExt {
				relPath, err := filepath.Rel(repoPath, path)
				if err == nil {
					files = append(files, relPath)
				}
				break
			}
		}
		return nil
	})

	return files, err
}

// storeSymbol stores a symbol in the database
func (ci *CodebaseIndexer) storeSymbol(ctx context.Context, symbol *CodeSymbol) error {
	// TODO: Implement database storage
	// Query: INSERT INTO code_symbols (id, repo_id, file_path, symbol_name, symbol_type, line_start, line_end, signature, dependencies)
	// VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	// ON CONFLICT (repo_id, file_path, symbol_name) DO UPDATE SET ...

	log.Debug().
		Str("symbol", symbol.SymbolName).
		Str("type", symbol.SymbolType).
		Str("file", symbol.FilePath).
		Int("deps", len(symbol.Dependencies)).
		Msg("Symbol extracted (ready for storage)")
	return nil
}

// GetRelatedCode finds code related to a given symbol or file using chunkyyy dependencies
func (ci *CodebaseIndexer) GetRelatedCode(ctx context.Context, repoID string, filePath string, symbolName string) ([]CodeSymbol, error) {
	// Extract dependencies from the file using chunkyyy
	deps, err := ci.chunkyyy.ExtractDependencies(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract dependencies: %w", err)
	}

	// Find symbols that match dependencies
	related := make([]CodeSymbol, 0)
	for _, dep := range deps {
		// TODO: Query database for symbols matching dep.Name
		// For now, return empty
	}

	return related, nil
}

// FindUsages finds all usages of a symbol
func (ci *CodebaseIndexer) FindUsages(ctx context.Context, repoID string, symbolName string) ([]CodeSymbol, error) {
	// TODO: Implement usage finding
	return []CodeSymbol{}, nil
}

// GetDependencies returns all dependencies of a symbol
func (ci *CodebaseIndexer) GetDependencies(ctx context.Context, repoID string, symbolName string) ([]CodeSymbol, error) {
	// TODO: Implement dependency lookup
	return []CodeSymbol{}, nil
}

// InvalidateIndex invalidates the index for a repository (when code changes)
func (ci *CodebaseIndexer) InvalidateIndex(ctx context.Context, repoID string, filePath string) error {
	log.Info().
		Str("repo_id", repoID).
		Str("file_path", filePath).
		Msg("Invalidating index for file")

	// TODO: Mark symbols in this file as stale
	// Will trigger re-indexing on next access

	return nil
}

// GetIndexStats returns statistics about the index
func (ci *CodebaseIndexer) GetIndexStats(ctx context.Context, repoID string) (int, error) {
	// TODO: Return count of indexed symbols
	return 0, nil
}

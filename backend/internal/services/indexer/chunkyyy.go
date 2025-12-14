package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
)

// ChunkyyyService provides integration with chunkyyy for AST parsing and symbol extraction
type ChunkyyyService struct {
	nodePath string
	repoPath string
}

// NewChunkyyyService creates a new chunkyyy service
func NewChunkyyyService(repoPath string, nodePath string) *ChunkyyyService {
	if nodePath == "" {
		nodePath = "node"
	}
	return &ChunkyyyService{
		nodePath: nodePath,
		repoPath: repoPath,
	}
}

// ChunkyyyChunk represents a chunk from chunkyyy
type ChunkyyyChunk struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"` // function, class, method, etc.
	Name         string   `json:"name"`
	QualifiedName string  `json:"qualifiedName"`
	FilePath     string   `json:"filePath"`
	StartLine    int      `json:"startLine"`
	EndLine      int      `json:"endLine"`
	Hash         string   `json:"hash"`
	Dependencies []Dependency `json:"dependencies"`
	Exported     bool     `json:"exported"`
	ExportName   string   `json:"exportName,omitempty"`
	Visibility   string   `json:"visibility,omitempty"`
	Async        bool     `json:"async,omitempty"`
	Parameters   []Parameter `json:"parameters,omitempty"`
	ReturnType   string   `json:"returnType,omitempty"`
}

// Dependency represents a code dependency
type Dependency struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Type   string `json:"type,omitempty"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// ExtractSymbols extracts code symbols from a file using chunkyyy
func (cs *ChunkyyyService) ExtractSymbols(ctx context.Context, filePath string) ([]CodeSymbol, error) {
	log.Info().Str("file", filePath).Msg("Extracting symbols using chunkyyy")

	// Create script to call chunkyyy
	script := cs.generateExtractScript(filePath)
	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("chunkyyy-extract-%d.js", time.Now().Unix()))
	defer os.Remove(scriptPath)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return nil, fmt.Errorf("failed to write extract script: %w", err)
	}

	// Execute script
	cmd := exec.CommandContext(ctx, cs.nodePath, scriptPath, cs.repoPath, filePath)
	cmd.Dir = cs.repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error().
			Err(err).
			Str("stderr", stderr.String()).
			Msg("Failed to extract symbols")
		return nil, fmt.Errorf("chunkyyy extraction failed: %w", err)
	}

	// Parse output
	var chunks []ChunkyyyChunk
	if err := json.Unmarshal(stdout.Bytes(), &chunks); err != nil {
		// Try to find JSON in output
		output := stdout.String()
		startIdx := -1
		endIdx := -1
		for i, r := range output {
			if r == '[' && startIdx == -1 {
				startIdx = i
			}
			if r == ']' {
				endIdx = i + 1
				break
			}
		}

		if startIdx >= 0 && endIdx > startIdx {
			jsonOutput := output[startIdx:endIdx]
			if err := json.Unmarshal([]byte(jsonOutput), &chunks); err != nil {
				return nil, fmt.Errorf("failed to parse chunkyyy output: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in chunkyyy output")
		}
	}

	// Convert to CodeSymbol
	symbols := make([]CodeSymbol, 0, len(chunks))
	for _, chunk := range chunks {
		deps := make([]string, 0, len(chunk.Dependencies))
		for _, dep := range chunk.Dependencies {
			deps = append(deps, dep.Name)
		}

		symbol := CodeSymbol{
			ID:           chunk.ID,
			FilePath:     chunk.FilePath,
			SymbolName:   chunk.Name,
			SymbolType:   chunk.Type,
			LineStart:    chunk.StartLine,
			LineEnd:      chunk.EndLine,
			Signature:    cs.buildSignature(chunk),
			Dependencies: deps,
			CreatedAt:    time.Now(),
		}
		symbols = append(symbols, symbol)
	}

	log.Info().
		Str("file", filePath).
		Int("symbols", len(symbols)).
		Msg("Extracted symbols from file")

	return symbols, nil
}

// ExtractDependencies extracts dependency information for a file
func (cs *ChunkyyyService) ExtractDependencies(ctx context.Context, filePath string) ([]Dependency, error) {
	chunks, err := cs.ExtractSymbols(ctx, filePath)
	if err != nil {
		return nil, err
	}

	deps := make(map[string]Dependency)
	for _, symbol := range chunks {
		for _, depName := range symbol.Dependencies {
			if _, exists := deps[depName]; !exists {
				deps[depName] = Dependency{
					Name: depName,
				}
			}
		}
	}

	result := make([]Dependency, 0, len(deps))
	for _, dep := range deps {
		result = append(result, dep)
	}

	return result, nil
}

// GetChunkHash gets the hash for a specific chunk (for caching)
func (cs *ChunkyyyService) GetChunkHash(ctx context.Context, filePath string, startLine int, endLine int) (string, error) {
	chunks, err := cs.ExtractSymbols(ctx, filePath)
	if err != nil {
		return "", err
	}

	// Find chunk that matches the line range
	for _, chunk := range chunks {
		if chunk.StartLine <= startLine && chunk.EndLine >= endLine {
			// Use chunkyyy's hash if available
			script := cs.generateHashScript(filePath, chunk.StartLine, chunk.EndLine)
			scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("chunkyyy-hash-%d.js", time.Now().Unix()))
			defer os.Remove(scriptPath)

			if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
				return "", err
			}

			cmd := exec.CommandContext(ctx, cs.nodePath, scriptPath, cs.repoPath, filePath)
			cmd.Dir = cs.repoPath

			output, err := cmd.Output()
			if err != nil {
				return "", err
			}

			hash := string(bytes.TrimSpace(output))
			if hash != "" {
				return hash, nil
			}
		}
	}

	// Fallback: compute hash from content
	return "", fmt.Errorf("chunk not found for range %d-%d", startLine, endLine)
}

// buildSignature builds a function signature from chunk metadata
func (cs *ChunkyyyService) buildSignature(chunk ChunkyyyChunk) string {
	if chunk.Type != "function" && chunk.Type != "method" {
		return ""
	}

	sig := chunk.Name + "("
	for i, param := range chunk.Parameters {
		if i > 0 {
			sig += ", "
		}
		sig += param.Name
		if param.Type != "" {
			sig += ": " + param.Type
		}
	}
	sig += ")"
	if chunk.ReturnType != "" {
		sig += ": " + chunk.ReturnType
	}

	return sig
}

// generateExtractScript generates a Node.js script to extract symbols using chunkyyy
func (cs *ChunkyyyService) generateExtractScript(filePath string) string {
	return fmt.Sprintf(`
const { Chunkyyy } = require('chunkyyy');
const path = require('path');

async function extractSymbols() {
  try {
    const repoPath = process.argv[2];
    const filePath = process.argv[3];
    const fullPath = path.join(repoPath, filePath);

    const chunkyyy = new Chunkyyy();
    const chunks = await chunkyyy.chunkFile(fullPath);

    // Convert to simplified format for Go
    const symbols = chunks.map(chunk => ({
      id: chunk.id || chunk.metadata?.id || '',
      type: chunk.type || chunk.metadata?.type || 'unknown',
      name: chunk.name || chunk.metadata?.name || 'unnamed',
      qualifiedName: chunk.qualifiedName || chunk.metadata?.qualifiedName || '',
      filePath: filePath,
      startLine: chunk.startLine || chunk.metadata?.startLine || chunk.range?.start?.line || 1,
      endLine: chunk.endLine || chunk.metadata?.endLine || chunk.range?.end?.line || 1,
      hash: chunk.hash || chunk.metadata?.hash || '',
      dependencies: (chunk.dependencies || chunk.metadata?.dependencies || []).map(dep => ({
        name: typeof dep === 'string' ? dep : (dep.name || dep),
        source: dep.source || '',
        type: dep.type || ''
      })),
      exported: chunk.exported || chunk.metadata?.exported || false,
      exportName: chunk.exportName || chunk.metadata?.exportName || '',
      visibility: chunk.visibility || chunk.metadata?.visibility || '',
      async: chunk.async || chunk.metadata?.async || false,
      parameters: (chunk.parameters || chunk.metadata?.parameters || []).map(p => ({
        name: p.name || p,
        type: p.type || ''
      })),
      returnType: chunk.returnType || chunk.metadata?.returnType || ''
    }));

    console.log(JSON.stringify(symbols));
  } catch (error) {
    console.error('Error extracting symbols:', error.message);
    process.exit(1);
  }
}

extractSymbols();
`)
}

// generateHashScript generates a script to get chunk hash
func (cs *ChunkyyyService) generateHashScript(filePath string, startLine int, endLine int) string {
	return fmt.Sprintf(`
const { Chunkyyy } = require('chunkyyy');
const path = require('path');

async function getChunkHash() {
  try {
    const repoPath = process.argv[2];
    const filePath = process.argv[3];
    const fullPath = path.join(repoPath, filePath);

    const chunkyyy = new Chunkyyy();
    const chunks = await chunkyyy.chunkFile(fullPath);

    // Find chunk matching the range
    for (const chunk of chunks) {
      const start = chunk.startLine || chunk.metadata?.startLine || chunk.range?.start?.line || 1;
      const end = chunk.endLine || chunk.metadata?.endLine || chunk.range?.end?.line || 1;
      
      if (start <= %d && end >= %d) {
        console.log(chunk.hash || chunk.metadata?.hash || '');
        return;
      }
    }

    console.log('');
  } catch (error) {
    console.error('Error getting chunk hash:', error.message);
    process.exit(1);
  }
}

getChunkHash();
`, startLine, endLine)
}


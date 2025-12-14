package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// RustIndexerService provides integration with Rust indexer microservice
type RustIndexerService struct {
	baseURL    string
	httpClient *http.Client
	enabled    bool
}

// NewRustIndexerService creates a new Rust indexer service client
func NewRustIndexerService(baseURL string) *RustIndexerService {
	if baseURL == "" {
		return &RustIndexerService{enabled: false}
	}

	return &RustIndexerService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		enabled: true,
	}
}

// RustSymbol represents a symbol from Rust indexer
type RustSymbol struct {
	ID           string   `json:"id"`
	SymbolName   string   `json:"symbol_name"`
	SymbolType   string   `json:"symbol_type"`
	FilePath     string   `json:"file_path"`
	LineStart    int      `json:"line_start"`
	LineEnd      int      `json:"line_end"`
	Signature    string   `json:"signature,omitempty"`
	Dependencies []string `json:"dependencies"`
	Exported     bool     `json:"exported"`
	Visibility   string   `json:"visibility,omitempty"`
}

// RustExtractRequest represents extraction request
type RustExtractRequest struct {
	StartLine int `json:"start_line,omitempty"`
	EndLine   int `json:"end_line,omitempty"`
}

// RustExtractResponse represents extraction response
type RustExtractResponse struct {
	Symbols []RustSymbol `json:"symbols"`
	Success bool         `json:"success"`
}

// ExtractSymbols extracts symbols using Rust service
func (rs *RustIndexerService) ExtractSymbols(ctx context.Context, repoPath string, filePath string) ([]CodeSymbol, error) {
	if !rs.enabled {
		return nil, fmt.Errorf("Rust indexer not enabled")
	}

	url := fmt.Sprintf("%s/extract/%s/%s", rs.baseURL, repoPath, filePath)
	
	reqBody := RustExtractRequest{}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := rs.httpClient.Do(req)
	if err != nil {
		log.Warn().Err(err).Str("file", filePath).Msg("Rust indexer request failed, falling back to chunkyyy")
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Rust indexer returned status %d", resp.StatusCode)
	}

	var rustResp RustExtractResponse
	if err := json.NewDecoder(resp.Body).Decode(&rustResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert Rust symbols to CodeSymbol
	symbols := make([]CodeSymbol, 0, len(rustResp.Symbols))
	for _, rs := range rustResp.Symbols {
		symbols = append(symbols, CodeSymbol{
			ID:           rs.ID,
			SymbolName:   rs.SymbolName,
			SymbolType:   rs.SymbolType,
			FilePath:     rs.FilePath,
			LineStart:    rs.LineStart,
			LineEnd:      rs.LineEnd,
			Signature:    rs.Signature,
			Dependencies: rs.Dependencies,
			CreatedAt:    time.Now(),
		})
	}

	return symbols, nil
}

// ExtractDependencies extracts dependencies using Rust service
func (rs *RustIndexerService) ExtractDependencies(ctx context.Context, repoPath string, filePath string) ([]Dependency, error) {
	if !rs.enabled {
		return nil, fmt.Errorf("Rust indexer not enabled")
	}

	url := fmt.Sprintf("%s/extract-deps/%s/%s", rs.baseURL, repoPath, filePath)
	
	reqBody := RustExtractRequest{}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := rs.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Rust indexer returned status %d", resp.StatusCode)
	}

	var rustResp RustExtractResponse
	if err := json.NewDecoder(resp.Body).Decode(&rustResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to dependencies
	deps := make([]Dependency, 0, len(rustResp.Symbols))
	for _, s := range rustResp.Symbols {
		deps = append(deps, Dependency{
			Name: s.SymbolName,
		})
	}

	return deps, nil
}

// GetChunkHash gets chunk hash using Rust service
func (rs *RustIndexerService) GetChunkHash(ctx context.Context, repoPath string, filePath string, startLine int, endLine int) (string, error) {
	if !rs.enabled {
		return "", fmt.Errorf("Rust indexer not enabled")
	}

	url := fmt.Sprintf("%s/hash/%s/%s", rs.baseURL, repoPath, filePath)
	
	reqBody := RustExtractRequest{
		StartLine: startLine,
		EndLine:   endLine,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := rs.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Rust indexer returned status %d", resp.StatusCode)
	}

	var hashResp struct {
		Hash    string `json:"hash"`
		Success bool   `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hashResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return hashResp.Hash, nil
}

// IsEnabled returns whether Rust indexer is enabled
func (rs *RustIndexerService) IsEnabled() bool {
	return rs.enabled
}


-- Migration: Add codebase indexing tables for context-aware analysis
-- Created: 2024-12-13
-- Description: Adds tables for indexing code symbols, dependencies, and relationships

-- Table for code symbols (functions, classes, methods, etc.)
CREATE TABLE IF NOT EXISTS code_symbols (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    symbol_name TEXT NOT NULL,
    symbol_type TEXT NOT NULL, -- 'function', 'class', 'method', 'variable', 'interface', 'type'
    line_start INT NOT NULL,
    line_end INT NOT NULL,
    signature TEXT,
    dependencies TEXT[], -- Array of dependent symbol IDs
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for code_symbols
CREATE INDEX IF NOT EXISTS idx_code_symbols_repo ON code_symbols(repo_id);
CREATE INDEX IF NOT EXISTS idx_code_symbols_file ON code_symbols(repo_id, file_path);
CREATE INDEX IF NOT EXISTS idx_code_symbols_name ON code_symbols(repo_id, symbol_name);
CREATE INDEX IF NOT EXISTS idx_code_symbols_type ON code_symbols(symbol_type);
CREATE INDEX IF NOT EXISTS idx_code_symbols_deps ON code_symbols USING GIN(dependencies);

-- Table for review feedback (for learning system)
CREATE TABLE IF NOT EXISTS review_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    comment_id TEXT NOT NULL, -- Comment identifier
    file_path TEXT NOT NULL, -- File path where comment was made
    line_number INT NOT NULL, -- Line number of the comment
    feedback TEXT NOT NULL, -- 'accepted', 'dismissed', 'fixed'
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(review_id, comment_id)
);

-- Indexes for review_feedback
CREATE INDEX IF NOT EXISTS idx_review_feedback_review ON review_feedback(review_id);
CREATE INDEX IF NOT EXISTS idx_review_feedback_user ON review_feedback(user_id);
CREATE INDEX IF NOT EXISTS idx_review_feedback_org ON review_feedback(org_id);
CREATE INDEX IF NOT EXISTS idx_review_feedback_file_line ON review_feedback(file_path, line_number);
CREATE INDEX IF NOT EXISTS idx_review_feedback_feedback ON review_feedback(feedback);
CREATE INDEX IF NOT EXISTS idx_review_feedback_created ON review_feedback(created_at);

-- Table for review cache (for incremental reviews)
CREATE TABLE IF NOT EXISTS review_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    chunk_hash TEXT NOT NULL, -- SHA256 hash of code chunk
    review_result JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    UNIQUE(repo_id, file_path, chunk_hash)
);

-- Indexes for review_cache
CREATE INDEX IF NOT EXISTS idx_review_cache_repo_file ON review_cache(repo_id, file_path);
CREATE INDEX IF NOT EXISTS idx_review_cache_hash ON review_cache(chunk_hash);
CREATE INDEX IF NOT EXISTS idx_review_cache_expires ON review_cache(expires_at);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to update updated_at on code_symbols
CREATE TRIGGER update_code_symbols_updated_at
    BEFORE UPDATE ON code_symbols
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger to update updated_at on review_feedback
CREATE TRIGGER update_review_feedback_updated_at
    BEFORE UPDATE ON review_feedback
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Cleanup function for expired cache entries
CREATE OR REPLACE FUNCTION cleanup_expired_cache()
RETURNS void AS $$
BEGIN
    DELETE FROM review_cache WHERE expires_at < NOW();
END;
$$ language 'plpgsql';

-- Add comment to tables
COMMENT ON TABLE code_symbols IS 'Stores code symbols (functions, classes, etc.) for codebase indexing';
COMMENT ON TABLE review_feedback IS 'Stores user feedback on reviews for learning system';
COMMENT ON TABLE review_cache IS 'Caches review results for incremental reviews';

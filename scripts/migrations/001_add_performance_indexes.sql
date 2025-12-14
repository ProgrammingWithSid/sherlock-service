-- Migration: Add performance indexes for Code-Sherlock improvements
-- Created: 2024-12-13
-- Description: Adds indexes to improve query performance for reviews, repositories, and usage tracking

-- Indexes for reviews table
CREATE INDEX IF NOT EXISTS idx_reviews_org_status ON reviews(org_id, status);
CREATE INDEX IF NOT EXISTS idx_reviews_repo_pr ON reviews(repo_id, pr_number);
CREATE INDEX IF NOT EXISTS idx_reviews_status_created ON reviews(status, created_at);
CREATE INDEX IF NOT EXISTS idx_reviews_head_sha ON reviews(head_sha);

-- Indexes for repositories table
CREATE INDEX IF NOT EXISTS idx_repositories_org_active ON repositories(org_id, is_active);
CREATE INDEX IF NOT EXISTS idx_repositories_full_name ON repositories(full_name);
CREATE INDEX IF NOT EXISTS idx_repositories_platform_external ON repositories(platform, external_id);

-- Indexes for usage_logs table
CREATE INDEX IF NOT EXISTS idx_usage_org_created ON usage_logs(org_id, created_at);
CREATE INDEX IF NOT EXISTS idx_usage_event_type ON usage_logs(event_type, created_at);

-- Indexes for organizations table
CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);
CREATE INDEX IF NOT EXISTS idx_organizations_plan ON organizations(plan);

-- Indexes for users table
CREATE INDEX IF NOT EXISTS idx_users_org ON users(org_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Indexes for sessions table (if exists)
CREATE INDEX IF NOT EXISTS idx_sessions_user_expires ON sessions(user_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);

-- Composite index for common review queries
CREATE INDEX IF NOT EXISTS idx_reviews_org_repo_status ON reviews(org_id, repo_id, status);

-- Index for finding recent reviews
CREATE INDEX IF NOT EXISTS idx_reviews_completed_at ON reviews(completed_at) WHERE completed_at IS NOT NULL;

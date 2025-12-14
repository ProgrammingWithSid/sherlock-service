package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type DB struct {
	conn *sql.DB
}

func New(databaseURL string) (*DB, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Ping() error {
	return db.conn.Ping()
}

// Conn returns the underlying sql.DB connection for advanced queries
func (db *DB) Conn() *sql.DB {
	return db.conn
}

func (db *DB) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS organizations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			plan VARCHAR(50) DEFAULT 'free',
			stripe_customer_id VARCHAR(255),
			stripe_subscription_id VARCHAR(255),
			plan_activated_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS repositories (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			platform VARCHAR(50) NOT NULL,
			external_id VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			full_name VARCHAR(255) NOT NULL,
			is_private BOOLEAN DEFAULT false,
			is_active BOOLEAN DEFAULT true,
			config TEXT DEFAULT '{}',
			created_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(org_id, platform, external_id)
		)`,
		`CREATE TABLE IF NOT EXISTS reviews (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			repo_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
			pr_number INTEGER NOT NULL,
			head_sha VARCHAR(40) NOT NULL,
			status VARCHAR(50) DEFAULT 'pending',
			result TEXT,
			comments_posted INTEGER DEFAULT 0,
			duration_ms INTEGER,
			ai_provider VARCHAR(50),
			created_at TIMESTAMP DEFAULT NOW(),
			completed_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS usage_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			event_type VARCHAR(50) NOT NULL,
			metadata TEXT DEFAULT '{}',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_org_created ON reviews(org_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_repo_pr ON reviews(repo_id, pr_number)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_org_month ON usage_logs(org_id, DATE_TRUNC('month', created_at))`,
		`CREATE TABLE IF NOT EXISTS github_installations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			installation_id BIGINT NOT NULL UNIQUE,
			token TEXT,
			token_expires TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			role VARCHAR(50) DEFAULT 'org_admin',
			org_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token VARCHAR(255) PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role VARCHAR(50) NOT NULL,
			org_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_org ON users(org_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_installations_org ON github_installations(org_id)`,
		`CREATE INDEX IF NOT EXISTS idx_installations_id ON github_installations(installation_id)`,
		// Review feedback table for learning system
		`CREATE TABLE IF NOT EXISTS review_feedback (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
			comment_id TEXT NOT NULL,
			file_path TEXT NOT NULL,
			line_number INT NOT NULL,
			feedback TEXT NOT NULL,
			user_id UUID REFERENCES users(id) ON DELETE SET NULL,
			org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(review_id, comment_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_review_feedback_review ON review_feedback(review_id)`,
		`CREATE INDEX IF NOT EXISTS idx_review_feedback_user ON review_feedback(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_review_feedback_org ON review_feedback(org_id)`,
		`CREATE INDEX IF NOT EXISTS idx_review_feedback_file_line ON review_feedback(file_path, line_number)`,
		`CREATE INDEX IF NOT EXISTS idx_review_feedback_feedback ON review_feedback(feedback)`,
		`CREATE INDEX IF NOT EXISTS idx_review_feedback_created ON review_feedback(created_at)`,
		// Function to update updated_at timestamp
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ language 'plpgsql'`,
		// Trigger to update updated_at on review_feedback
		`DROP TRIGGER IF EXISTS update_review_feedback_updated_at ON review_feedback`,
		`CREATE TRIGGER update_review_feedback_updated_at
		BEFORE UPDATE ON review_feedback
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column()`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			log.Error().Err(err).Str("query", query).Msg("Migration failed")
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	log.Info().Msg("Database migrations completed")
	return nil
}

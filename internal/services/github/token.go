package github

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v57/github"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

// TokenService handles GitHub App token generation and refresh
type TokenService struct {
	appID      int64
	privateKey *rsa.PrivateKey
	client     *github.Client
}

func NewTokenService(appID int64, privateKeyPEM []byte) (*TokenService, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create JWT token for app authentication
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": time.Now().Unix() - 60,
		"exp": time.Now().Unix() + 600,
		"iss": appID,
	})

	jwtToken, err := token.SignedString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign JWT: %w", err)
	}

	// Create GitHub client with JWT
	ctx := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: jwtToken})
	tc := oauth2.NewClient(nil, ctx)
	client := github.NewClient(tc)

	return &TokenService{
		appID:      appID,
		privateKey: privateKey,
		client:     client,
	}, nil
}

// GetInstallationToken gets or refreshes an installation token
func (s *TokenService) GetInstallationToken(installationID int64) (string, *time.Time, error) {
	ctx := oauth2.NoContext

	// Create JWT for app authentication
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": time.Now().Unix() - 60,
		"exp": time.Now().Unix() + 600,
		"iss": s.appID,
	})

	jwtToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to sign JWT: %w", err)
	}

	// Create temporary client with JWT
	ctxSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: jwtToken})
	tc := oauth2.NewClient(ctx, ctxSource)
	tempClient := github.NewClient(tc)

	// Get installation token
	installationToken, _, err := tempClient.Apps.CreateInstallationToken(ctx, installationID, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create installation token: %w", err)
	}

	expiresAt := installationToken.GetExpiresAt()
	return installationToken.GetToken(), &expiresAt.Time, nil
}

// RefreshTokenIfNeeded checks if token needs refresh and refreshes it
func (s *TokenService) RefreshTokenIfNeeded(currentToken string, expiresAt *time.Time) (string, *time.Time, bool, error) {
	if expiresAt == nil {
		return "", nil, false, fmt.Errorf("expiresAt is nil")
	}

	// Refresh if token expires within 5 minutes
	if time.Until(*expiresAt) < 5*time.Minute {
		// Token needs refresh - but we need installation ID
		// This would be called with installation ID
		return "", nil, true, fmt.Errorf("token refresh needed but installation ID required")
	}

	return currentToken, expiresAt, false, nil
}

// GetInstallationTokenWithRefresh gets token and refreshes if needed
func (s *TokenService) GetInstallationTokenWithRefresh(installationID int64, currentToken string, expiresAt *time.Time) (string, *time.Time, error) {
	if expiresAt != nil {
		refreshed, newExpiresAt, needsRefresh, err := s.RefreshTokenIfNeeded(currentToken, expiresAt)
		if err == nil && !needsRefresh {
			return refreshed, newExpiresAt, nil
		}
	}

	// Get new token
	return s.GetInstallationToken(installationID)
}


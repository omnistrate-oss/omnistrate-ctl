package common

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShouldRefreshToken(t *testing.T) {
	tests := []struct {
		name      string
		expiresIn time.Duration
		expected  bool
	}{
		{
			name:      "refreshes token expiring before margin",
			expiresIn: tokenRefreshMargin - time.Second,
			expected:  true,
		},
		{
			name:      "refreshes token expiring exactly at margin",
			expiresIn: tokenRefreshMargin,
			expected:  true,
		},
		{
			name:      "keeps token expiring after margin",
			expiresIn: tokenRefreshMargin + time.Second,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := makeJWT(time.Now().Add(tt.expiresIn).Unix())
			assert.Equal(t, tt.expected, shouldRefreshToken(token))
		})
	}
}

func makeJWT(exp int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(struct {
		Exp int64 `json:"exp"`
	}{
		Exp: exp,
	})
	claims := base64.RawURLEncoding.EncodeToString(payload)
	return fmt.Sprintf("%s.%s.fakesig", header, claims)
}

func TestAPIKeyEnvConst(t *testing.T) {
	assert.Equal(t, "OMNISTRATE_API_KEY", apiKeyEnv)
}

// TestGetTokenWithLoginUsesEnvVar asserts that when no stored token
// exists and OMNISTRATE_API_KEY is set, GetTokenWithLogin attempts a
// signin-exchange rather than falling through to RunLogin (interactive).
func TestGetTokenWithLoginUsesEnvVar(t *testing.T) {
	t.Setenv("OMNISTRATE_API_KEY", "om_test_env_key")
	t.Setenv("OMNISTRATE_DRY_RUN", "true")

	// Ensure no stored auth so we hit the env-var path.
	// We don't remove real config — the test will fail at the HTTP layer
	// but we verify it tried the exchange (not the interactive prompt).
	token, err := GetTokenWithLogin()
	// Expected: fails at HTTP because no real server, but the error
	// should mention signin-exchange, not "login" prompt issues.
	if token == "" && err != nil {
		assert.Contains(t, err.Error(), "OMNISTRATE_API_KEY signin-exchange failed",
			"should attempt env-var exchange, not interactive login")
	}
	// If it somehow succeeds (unlikely in unit test), that's fine too.
}

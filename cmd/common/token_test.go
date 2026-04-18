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

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
	t.Run("refreshes token expiring within five minutes", func(t *testing.T) {
		token := makeJWT(time.Now().Add(4 * time.Minute).Unix())
		assert.True(t, shouldRefreshToken(token))
	})

	t.Run("keeps token with more than five minutes remaining", func(t *testing.T) {
		token := makeJWT(time.Now().Add(10 * time.Minute).Unix())
		assert.False(t, shouldRefreshToken(token))
	})
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

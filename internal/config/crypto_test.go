package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptToken(t *testing.T) {
	require := require.New(t)

	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test-refresh-token" // #nosec G101 -- test data, not a real credential

	encrypted, err := EncryptToken(token)
	require.NoError(err)
	require.NotEmpty(encrypted)
	require.NotEqual(token, encrypted, "encrypted should differ from plaintext")

	decrypted, err := DecryptToken(encrypted)
	require.NoError(err)
	require.Equal(token, decrypted)
}

func TestEncryptTokenProducesDifferentCiphertexts(t *testing.T) {
	require := require.New(t)

	token := "same-token-value"

	enc1, err := EncryptToken(token)
	require.NoError(err)

	enc2, err := EncryptToken(token)
	require.NoError(err)

	require.NotEqual(enc1, enc2, "each encryption should use a unique nonce")

	// Both should decrypt to the same value
	dec1, err := DecryptToken(enc1)
	require.NoError(err)
	dec2, err := DecryptToken(enc2)
	require.NoError(err)
	require.Equal(dec1, dec2)
}

func TestDecryptTokenInvalidData(t *testing.T) {
	require := require.New(t)

	_, err := DecryptToken("not-hex")
	require.Error(err)

	_, err = DecryptToken("deadbeef")
	require.Error(err)
}

func TestEncryptDecryptEmptyString(t *testing.T) {
	require := require.New(t)

	encrypted, err := EncryptToken("")
	require.NoError(err)

	decrypted, err := DecryptToken(encrypted)
	require.NoError(err)
	require.Equal("", decrypted)
}

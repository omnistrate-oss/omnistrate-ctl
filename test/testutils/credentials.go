package testutils

import (
	"crypto/rand"
	"math/big"
	"strings"
	"sync"
)

// TestDBPasswordPlaceholder is the token that service specs use in place of a
// database password. Tests replace it with TestDBPassword() before sending a
// spec to Omnistrate, so committed specs never carry a fixed password.
const TestDBPasswordPlaceholder = "__OMNISTRATE_TEST_DB_PASSWORD__" //nolint:gosec // placeholder token, not a credential

const (
	testDBPasswordLength   = 32
	testDBPasswordAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var (
	testDBPasswordOnce  sync.Once
	testDBPasswordValue string
)

// TestDBPassword returns a random password that is generated once per test
// process. Every rendered spec in the same process receives the same value.
func TestDBPassword() string {
	testDBPasswordOnce.Do(func() {
		testDBPasswordValue = RandomTestPassword()
	})
	return testDBPasswordValue
}

// RandomTestPassword returns a cryptographically random password suitable for
// ephemeral service instances. The alphanumeric alphabet avoids escaping in
// JSON, YAML, shell arguments, and connection URLs.
func RandomTestPassword() string {
	alphabetSize := big.NewInt(int64(len(testDBPasswordAlphabet)))
	var b strings.Builder
	b.Grow(testDBPasswordLength)
	for range testDBPasswordLength {
		n, err := rand.Int(rand.Reader, alphabetSize)
		if err != nil {
			panic("crypto/rand failed while generating a test database password: " + err.Error())
		}
		b.WriteByte(testDBPasswordAlphabet[n.Int64()])
	}
	return b.String()
}

// RenderTestCredentials replaces every database credential placeholder in a
// test spec with the random value for the current test process.
func RenderTestCredentials(spec string) string {
	if !strings.Contains(spec, TestDBPasswordPlaceholder) {
		return spec
	}
	return strings.ReplaceAll(spec, TestDBPasswordPlaceholder, TestDBPassword())
}

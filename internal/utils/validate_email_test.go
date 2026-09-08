package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateEmail(t *testing.T) {
	for name, tc := range map[string]struct {
		email   string
		wantErr bool
	}{
		"capitalized address is accepted": {
			email: "Customer@Example.com",
		},
		"dot-cloud TLD is accepted": {
			email: "customer@acme.cloud",
		},
		"dot-online TLD is accepted": {
			email: "customer@acme.online",
		},
		"plus tag is accepted": {
			email: "customer+tag@example.com",
		},
		"org-scoped address is accepted": {
			email: "customer+org-abc123@example.com",
		},
		"missing at sign is rejected": {
			email:   "customer.example.com",
			wantErr: true,
		},
		"two at signs are rejected": {
			email:   "customer@example@com",
			wantErr: true,
		},
		"empty local part is rejected": {
			email:   "@example.com",
			wantErr: true,
		},
		"empty domain is rejected": {
			email:   "customer@",
			wantErr: true,
		},
		"domain without a dot is rejected": {
			email:   "customer@example",
			wantErr: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateEmail(tc.email)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

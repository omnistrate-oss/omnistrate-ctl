package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// scopedOrgIDRegex matches an organization ID. It mirrors the server-side definition so the
// CLI and the platform agree on what the trailing segment of a scoped address looks like.
var scopedOrgIDRegex = regexp.MustCompile(`^org-[a-zA-Z0-9][a-zA-Z0-9._\-]*$`)

// FormatEmailWithScopedOrg returns the org-scoped form of an address, which is how the
// platform stores an end customer's identity inside a service provider's organization:
//
//	("customer@example.com", "org-abc123") -> "customer+org-abc123@example.com"
//
// Any plus tag already present is preserved, with the organization ID appended after it.
func FormatEmailWithScopedOrg(email, orgID string) (string, error) {
	localPart, domain, err := splitEmail(email)
	if err != nil {
		return "", err
	}
	if !scopedOrgIDRegex.MatchString(orgID) {
		return "", fmt.Errorf("invalid organization ID %q", orgID)
	}
	if localPartHasScopedOrg(localPart) {
		return "", fmt.Errorf("email %q is already scoped to an organization", email)
	}

	return fmt.Sprintf("%s+%s@%s", localPart, orgID, domain), nil
}

// EmailHasScopedOrg reports whether an address already carries an organization ID as the
// last segment of its local part.
func EmailHasScopedOrg(email string) bool {
	localPart, _, err := splitEmail(email)
	if err != nil {
		return false
	}
	return localPartHasScopedOrg(localPart)
}

// IsProductionEnvironmentType reports whether an environment type is a production one.
// Org-scoped customer identities only exist in production environments.
func IsProductionEnvironmentType(environmentType string) bool {
	switch strings.ToUpper(strings.TrimSpace(environmentType)) {
	case "PROD", "PRODUCTION":
		return true
	default:
		return false
	}
}

func splitEmail(email string) (localPart, domain string, err error) {
	parts := strings.Split(strings.TrimSpace(email), "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid email address %q", email)
	}
	return parts[0], parts[1], nil
}

func localPartHasScopedOrg(localPart string) bool {
	segments := strings.Split(localPart, "+")
	if len(segments) < 2 {
		return false
	}
	return scopedOrgIDRegex.MatchString(segments[len(segments)-1])
}

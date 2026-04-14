package instance

import "testing"

func TestResolveBreakpointResourceIDAndKey(t *testing.T) {
	keyByID := map[string]string{
		"r-1": "writer",
	}
	idByKey := map[string]string{
		"writer": "r-1",
	}

	tests := []struct {
		name        string
		idOrKey     string
		expectedID  string
		expectedKey string
	}{
		{
			name:        "id to key mapping",
			idOrKey:     "r-1",
			expectedID:  "r-1",
			expectedKey: "writer",
		},
		{
			name:        "key to id mapping",
			idOrKey:     "writer",
			expectedID:  "r-1",
			expectedKey: "writer",
		},
		{
			name:        "unknown value fallback",
			idOrKey:     "custom",
			expectedID:  "custom",
			expectedKey: "custom",
		},
		{
			name:        "empty value fallback",
			idOrKey:     "   ",
			expectedID:  "unknown",
			expectedKey: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, key := resolveBreakpointResourceIDAndKey(tt.idOrKey, keyByID, idByKey)
			if id != tt.expectedID {
				t.Fatalf("expected ID %q, got %q", tt.expectedID, id)
			}
			if key != tt.expectedKey {
				t.Fatalf("expected key %q, got %q", tt.expectedKey, key)
			}
		})
	}
}

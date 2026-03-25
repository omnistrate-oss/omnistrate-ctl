package instance

import "testing"

func TestParseOperationAttemptParts(t *testing.T) {
	tests := []struct {
		name               string
		operationID        string
		expectedGeneration string
		expectedNonce      string
		expectedCanonical  string
	}{
		{
			name:               "generation and nonce",
			operationID:        "genhash123.attempt456",
			expectedGeneration: "genhash123",
			expectedNonce:      "attempt456",
			expectedCanonical:  "genhash123.attempt456",
		},
		{
			name:               "legacy operation id without nonce",
			operationID:        "legacy-op-id",
			expectedGeneration: "legacy-op-id",
			expectedNonce:      "(legacy)",
			expectedCanonical:  "legacy-op-id",
		},
		{
			name:               "empty operation id",
			operationID:        "",
			expectedGeneration: "(unknown)",
			expectedNonce:      "(unknown)",
			expectedCanonical:  "(unknown)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generationID, nonce, canonicalID := parseOperationAttemptParts(tt.operationID)
			if generationID != tt.expectedGeneration {
				t.Fatalf("expected generation %q, got %q", tt.expectedGeneration, generationID)
			}
			if nonce != tt.expectedNonce {
				t.Fatalf("expected nonce %q, got %q", tt.expectedNonce, nonce)
			}
			if canonicalID != tt.expectedCanonical {
				t.Fatalf("expected canonical %q, got %q", tt.expectedCanonical, canonicalID)
			}
		})
	}
}

func TestBuildTimelineSections_GroupsByGenerationAndAttempts(t *testing.T) {
	history := []TerraformHistoryEntry{
		{
			Operation:   "init",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:00Z",
			CompletedAt: "2026-03-03T10:00:10Z",
			OperationID: "genA.nonceA1",
		},
		{
			Operation:   "apply",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:11Z",
			CompletedAt: "2026-03-03T10:00:30Z",
			OperationID: "genA.nonceA1",
		},
		{
			Operation:   "init",
			Status:      "completed",
			StartedAt:   "2026-03-03T11:00:00Z",
			CompletedAt: "2026-03-03T11:00:10Z",
			OperationID: "genA.nonceA2",
		},
		{
			Operation:   "apply",
			Status:      "completed",
			StartedAt:   "2026-03-03T11:00:11Z",
			CompletedAt: "2026-03-03T11:00:40Z",
			OperationID: "genA.nonceA2",
		},
		{
			Operation:   "output",
			Status:      "completed",
			StartedAt:   "2026-03-03T11:00:41Z",
			CompletedAt: "2026-03-03T11:00:45Z",
			OperationID: "genA.nonceA2",
		},
		{
			Operation:   "init",
			Status:      "completed",
			StartedAt:   "2026-03-03T12:00:00Z",
			CompletedAt: "2026-03-03T12:00:08Z",
			OperationID: "genB.nonceB1",
		},
		{
			Operation:   "apply",
			Status:      "completed",
			StartedAt:   "2026-03-03T12:00:09Z",
			CompletedAt: "2026-03-03T12:00:25Z",
			OperationID: "genB.nonceB1",
		},
	}

	sections := buildTimelineSections(history)
	if len(sections) != 1 {
		t.Fatalf("expected 1 date section, got %d", len(sections))
	}

	groups := sections[0].groups
	if len(groups) != 2 {
		t.Fatalf("expected 2 generation groups, got %d", len(groups))
	}

	// Latest generation (genB) should come first.
	if groups[0].generationID != "genB" {
		t.Fatalf("expected first generation to be genB, got %q", groups[0].generationID)
	}
	if len(groups[0].attempts) != 1 {
		t.Fatalf("expected genB to have 1 attempt, got %d", len(groups[0].attempts))
	}
	if groups[0].attempts[0].nonce != "nonceB1" {
		t.Fatalf("expected genB attempt nonce nonceB1, got %q", groups[0].attempts[0].nonce)
	}

	// genA should include two attempts, with newest attempt first.
	if groups[1].generationID != "genA" {
		t.Fatalf("expected second generation to be genA, got %q", groups[1].generationID)
	}
	if len(groups[1].attempts) != 2 {
		t.Fatalf("expected genA to have 2 attempts, got %d", len(groups[1].attempts))
	}
	if groups[1].attempts[0].nonce != "nonceA2" {
		t.Fatalf("expected latest genA attempt nonceA2 first, got %q", groups[1].attempts[0].nonce)
	}
	if groups[1].attempts[1].nonce != "nonceA1" {
		t.Fatalf("expected older genA attempt nonceA1 second, got %q", groups[1].attempts[1].nonce)
	}

	if groups[1].attempts[0].summary != "init → apply → output" {
		t.Fatalf("unexpected summary for genA nonceA2: %q", groups[1].attempts[0].summary)
	}
	if groups[1].attempts[1].summary != "init → apply" {
		t.Fatalf("unexpected summary for genA nonceA1: %q", groups[1].attempts[1].summary)
	}
}

func TestFlattenTimeline_IncludesAttemptRows(t *testing.T) {
	history := []TerraformHistoryEntry{
		{
			Operation:   "init",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:00Z",
			CompletedAt: "2026-03-03T10:00:10Z",
			OperationID: "genA.nonceA1",
		},
		{
			Operation:   "apply",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:11Z",
			CompletedAt: "2026-03-03T10:00:30Z",
			OperationID: "genA.nonceA1",
		},
		{
			Operation:   "init",
			Status:      "completed",
			StartedAt:   "2026-03-03T11:00:00Z",
			CompletedAt: "2026-03-03T11:00:10Z",
			OperationID: "genA.nonceA2",
		},
		{
			Operation:   "apply",
			Status:      "completed",
			StartedAt:   "2026-03-03T11:00:11Z",
			CompletedAt: "2026-03-03T11:00:40Z",
			OperationID: "genA.nonceA2",
		},
		{
			Operation:   "output",
			Status:      "completed",
			StartedAt:   "2026-03-03T11:00:41Z",
			CompletedAt: "2026-03-03T11:00:45Z",
			OperationID: "genA.nonceA2",
		},
		{
			Operation:   "init",
			Status:      "completed",
			StartedAt:   "2026-03-03T12:00:00Z",
			CompletedAt: "2026-03-03T12:00:08Z",
			OperationID: "genB.nonceB1",
		},
		{
			Operation:   "apply",
			Status:      "completed",
			StartedAt:   "2026-03-03T12:00:09Z",
			CompletedAt: "2026-03-03T12:00:25Z",
			OperationID: "genB.nonceB1",
		},
	}

	sections := buildTimelineSections(history)
	if len(sections) != 1 {
		t.Fatalf("expected one section, got %d", len(sections))
	}

	sections[0].expanded = true
	for gi := range sections[0].groups {
		sections[0].groups[gi].expanded = true
		for ai := range sections[0].groups[gi].attempts {
			sections[0].groups[gi].attempts[ai].expanded = true
		}
	}

	rows := flattenTimeline(sections)
	dateHeaders := 0
	groupHeaders := 0
	attemptHeaders := 0
	entryRows := 0
	for _, row := range rows {
		if row.isDateHeader {
			dateHeaders++
			continue
		}
		if row.isGroupHeader {
			groupHeaders++
			continue
		}
		if row.isAttemptHeader {
			attemptHeaders++
			continue
		}
		if row.entry != nil {
			entryRows++
		}
	}

	if dateHeaders != 1 {
		t.Fatalf("expected 1 date header row, got %d", dateHeaders)
	}
	if groupHeaders != 2 {
		t.Fatalf("expected 2 generation header rows, got %d", groupHeaders)
	}
	if attemptHeaders != 3 {
		t.Fatalf("expected 3 attempt header rows, got %d", attemptHeaders)
	}
	if entryRows != 7 {
		t.Fatalf("expected 7 entry rows, got %d", entryRows)
	}
}

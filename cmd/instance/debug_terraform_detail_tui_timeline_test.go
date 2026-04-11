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

	rows := flattenTimeline(sections, nil, nil)
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

func TestFlattenTimeline_PlanPreviewRowBeforeApply(t *testing.T) {
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
			Operation:   "output",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:31Z",
			CompletedAt: "2026-03-03T10:00:35Z",
			OperationID: "genA.nonceA1",
		},
	}

	sections := buildTimelineSections(history)
	sections[0].expanded = true
	sections[0].groups[0].expanded = true
	sections[0].groups[0].attempts[0].expanded = true

	previews := map[string]string{
		"genA.nonceA1": `{"planned_values":{}}`,
	}

	rows := flattenTimeline(sections, previews, nil)
	var previewRows int
	var previewBeforeApply bool
	for i, row := range rows {
		if row.isPlanPreviewRow {
			previewRows++
			if row.planPreviewOpID != "genA.nonceA1" {
				t.Fatalf("expected plan preview opID genA.nonceA1, got %q", row.planPreviewOpID)
			}
			// Next row should be the "apply" entry
			if i+1 < len(rows) && rows[i+1].entry != nil && rows[i+1].entry.Operation == "apply" {
				previewBeforeApply = true
			}
		}
	}
	if previewRows != 1 {
		t.Fatalf("expected 1 plan preview row, got %d", previewRows)
	}
	if !previewBeforeApply {
		t.Fatal("expected plan preview row to appear before 'apply' entry")
	}
}

func TestFlattenTimeline_NoPlanPreviewWithoutData(t *testing.T) {
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
	}

	sections := buildTimelineSections(history)
	sections[0].expanded = true
	sections[0].groups[0].expanded = true
	sections[0].groups[0].attempts[0].expanded = true

	rows := flattenTimeline(sections, nil, nil)
	for _, row := range rows {
		if row.isPlanPreviewRow {
			t.Fatal("expected no plan preview row when no preview data exists")
		}
	}
}

func TestFlattenTimeline_PlanPreviewErrorRow(t *testing.T) {
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
			Status:      "failed",
			StartedAt:   "2026-03-03T10:00:11Z",
			CompletedAt: "2026-03-03T10:00:30Z",
			OperationID: "genA.nonceA1",
		},
	}

	sections := buildTimelineSections(history)
	sections[0].expanded = true
	sections[0].groups[0].expanded = true
	sections[0].groups[0].attempts[0].expanded = true

	previewErrors := map[string]string{
		"genA.nonceA1": "Error: Failed to refresh state",
	}

	rows := flattenTimeline(sections, nil, previewErrors)
	var previewRows int
	for _, row := range rows {
		if row.isPlanPreviewRow {
			previewRows++
		}
	}
	if previewRows != 1 {
		t.Fatalf("expected 1 plan preview row for error, got %d", previewRows)
	}
}

func TestFlattenTimeline_PlanPreviewAtEndWithoutApply(t *testing.T) {
	history := []TerraformHistoryEntry{
		{
			Operation:   "init",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:00Z",
			CompletedAt: "2026-03-03T10:00:10Z",
			OperationID: "genA.nonceA1",
		},
		{
			Operation:   "output",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:11Z",
			CompletedAt: "2026-03-03T10:00:20Z",
			OperationID: "genA.nonceA1",
		},
	}

	sections := buildTimelineSections(history)
	sections[0].expanded = true
	sections[0].groups[0].expanded = true
	sections[0].groups[0].attempts[0].expanded = true

	previews := map[string]string{
		"genA.nonceA1": `{"planned_values":{}}`,
	}

	rows := flattenTimeline(sections, previews, nil)
	var previewRows int
	for i, row := range rows {
		if row.isPlanPreviewRow {
			previewRows++
			// Should be the last child row
			if !row.isLastChild {
				t.Fatal("expected plan preview row to be last child when no apply entry exists")
			}
			// Should be the last non-header row (at the end)
			if i != len(rows)-1 {
				t.Fatalf("expected plan preview to be last row, but it's at index %d of %d", i, len(rows))
			}
		}
	}
	if previewRows != 1 {
		t.Fatalf("expected 1 plan preview row, got %d", previewRows)
	}
}

// TestFlattenTimeline_PlanPreviewMatchesByGenerationID verifies that plan previews
// keyed by generation ID only (no nonce) match attempts whose operationID is
// "generationID.nonce". This mirrors real ConfigMap names like
// tf-plan-tf-r-xxx-instance-yyy-{generationID} where the suffix is only the generation.
func TestFlattenTimeline_PlanPreviewMatchesByGenerationID(t *testing.T) {
	history := []TerraformHistoryEntry{
		{
			Operation:   "init",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:00Z",
			CompletedAt: "2026-03-03T10:00:10Z",
			OperationID: "a4b6ca139fecb81e1804.69965f91ff",
		},
		{
			Operation:   "apply",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:11Z",
			CompletedAt: "2026-03-03T10:00:30Z",
			OperationID: "a4b6ca139fecb81e1804.69965f91ff",
		},
		{
			Operation:   "output",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:31Z",
			CompletedAt: "2026-03-03T10:00:35Z",
			OperationID: "a4b6ca139fecb81e1804.69965f91ff",
		},
	}

	sections := buildTimelineSections(history)
	sections[0].expanded = true
	sections[0].groups[0].expanded = true
	sections[0].groups[0].attempts[0].expanded = true

	// Key is the generation ID only (no nonce) — this is how dedicated tf-plan-* CMs work
	previews := map[string]string{
		"a4b6ca139fecb81e1804": `{"planned_values":{"root_module":{}}}`,
	}

	rows := flattenTimeline(sections, previews, nil)
	var previewRows int
	var previewBeforeApply bool
	for i, row := range rows {
		if row.isPlanPreviewRow {
			previewRows++
			// Next row should be the "apply" entry
			if i+1 < len(rows) && rows[i+1].entry != nil && rows[i+1].entry.Operation == "apply" {
				previewBeforeApply = true
			}
		}
	}
	if previewRows != 1 {
		t.Fatalf("expected 1 plan preview row with generation-only key, got %d", previewRows)
	}
	if !previewBeforeApply {
		t.Fatal("expected plan preview row to appear before 'apply' entry")
	}
}

// TestFlattenTimeline_PlanPreviewGenerationKeyMultipleAttempts verifies that when a
// generation-only preview key exists, ALL attempts within that generation see the preview.
func TestFlattenTimeline_PlanPreviewGenerationKeyMultipleAttempts(t *testing.T) {
	history := []TerraformHistoryEntry{
		// First attempt
		{
			Operation:   "output",
			Status:      "failed",
			StartedAt:   "2026-03-03T10:00:00Z",
			CompletedAt: "2026-03-03T10:00:05Z",
			OperationID: "genX.nonce1",
		},
		{
			Operation:   "init",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:06Z",
			CompletedAt: "2026-03-03T10:00:10Z",
			OperationID: "genX.nonce1",
		},
		{
			Operation:   "output",
			Status:      "completed",
			StartedAt:   "2026-03-03T10:00:11Z",
			CompletedAt: "2026-03-03T10:00:15Z",
			OperationID: "genX.nonce1",
		},
		// Second attempt (has apply)
		{
			Operation:   "init",
			Status:      "completed",
			StartedAt:   "2026-03-03T09:00:00Z",
			CompletedAt: "2026-03-03T09:00:10Z",
			OperationID: "genX.nonce2",
		},
		{
			Operation:   "apply",
			Status:      "completed",
			StartedAt:   "2026-03-03T09:00:11Z",
			CompletedAt: "2026-03-03T09:00:30Z",
			OperationID: "genX.nonce2",
		},
		{
			Operation:   "output",
			Status:      "completed",
			StartedAt:   "2026-03-03T09:00:31Z",
			CompletedAt: "2026-03-03T09:00:35Z",
			OperationID: "genX.nonce2",
		},
	}

	sections := buildTimelineSections(history)
	sections[0].expanded = true
	for gi := range sections[0].groups {
		sections[0].groups[gi].expanded = true
		for ai := range sections[0].groups[gi].attempts {
			sections[0].groups[gi].attempts[ai].expanded = true
		}
	}

	// Generation-only key — matches all attempts in this generation
	previews := map[string]string{
		"genX": `{"planned_values":{}}`,
	}

	rows := flattenTimeline(sections, previews, nil)
	var previewRows int
	for _, row := range rows {
		if row.isPlanPreviewRow {
			previewRows++
		}
	}
	// Both attempts should get a plan preview row
	if previewRows != 2 {
		t.Fatalf("expected 2 plan preview rows (one per attempt), got %d", previewRows)
	}
}

func TestPlanPreviewLookupKeys(t *testing.T) {
	tests := []struct {
		name     string
		opID     string
		expected []string
	}{
		{
			name:     "canonical with nonce",
			opID:     "genA.nonceA1",
			expected: []string{"genA.nonceA1", "genA"},
		},
		{
			name:     "generation only (legacy or already short)",
			opID:     "genA",
			expected: []string{"genA"},
		},
		{
			name:     "real-world generation hash with nonce",
			opID:     "a4b6ca139fecb81e1804.69965f91ff",
			expected: []string{"a4b6ca139fecb81e1804.69965f91ff", "a4b6ca139fecb81e1804"},
		},
		{
			name:     "empty",
			opID:     "",
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := planPreviewLookupKeys(tt.opID)
			if len(keys) != len(tt.expected) {
				t.Fatalf("expected %d keys, got %d: %v", len(tt.expected), len(keys), keys)
			}
			for i, k := range keys {
				if k != tt.expected[i] {
					t.Fatalf("key[%d]: expected %q, got %q", i, tt.expected[i], k)
				}
			}
		})
	}
}

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyDir(t *testing.T) {
	// Create a temporary source directory with test files
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create test files and subdirectories
	testFiles := map[string]string{
		"file1.txt":         "content1",
		"file2.txt":         "content2",
		"subdir/file3.txt":  "content3",
		"subdir/file4.txt":  "content4",
		"subdir2/file5.txt": "content5",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(srcDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Copy directory
	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Verify all files were copied
	for path, expectedContent := range testFiles {
		dstPath := filepath.Join(dstDir, path)
		content, err := os.ReadFile(dstPath)
		if err != nil {
			t.Errorf("Failed to read copied file %s: %v", path, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s: got %q, want %q", path, string(content), expectedContent)
		}
	}
}

func TestMergeMarkdownFile_CreateNew(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "src.md")
	dstPath := filepath.Join(tmpDir, "dst.md")

	srcContent := "This is source content"
	if err := os.WriteFile(srcPath, []byte(srcContent), 0600); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Merge into non-existent destination
	if err := mergeMarkdownFile(srcPath, dstPath); err != nil {
		t.Fatalf("mergeMarkdownFile failed: %v", err)
	}

	// Verify destination was created with header
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	expectedContent := omnistrateSectionHeader + srcContent
	if string(dstContent) != expectedContent {
		t.Errorf("Content mismatch: got %q, want %q", string(dstContent), expectedContent)
	}
}

func TestMergeMarkdownFile_AppendToExisting(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "src.md")
	dstPath := filepath.Join(tmpDir, "dst.md")

	srcContent := "Omnistrate content"
	existingContent := "# Existing Content\n\nSome existing text"

	if err := os.WriteFile(srcPath, []byte(srcContent), 0600); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	if err := os.WriteFile(dstPath, []byte(existingContent), 0600); err != nil {
		t.Fatalf("Failed to create destination file: %v", err)
	}

	// Merge into existing destination
	if err := mergeMarkdownFile(srcPath, dstPath); err != nil {
		t.Fatalf("mergeMarkdownFile failed: %v", err)
	}

	// Verify destination has both contents
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if !strings.Contains(string(dstContent), existingContent) {
		t.Errorf("Destination should contain existing content")
	}
	if !strings.Contains(string(dstContent), omnistrateSectionHeader) {
		t.Errorf("Destination should contain Omnistrate section header")
	}
	if !strings.Contains(string(dstContent), srcContent) {
		t.Errorf("Destination should contain source content")
	}
}

func TestMergeMarkdownFile_ReplaceExistingSection(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "src.md")
	dstPath := filepath.Join(tmpDir, "dst.md")

	srcContent := "New Omnistrate content"
	existingContent := "# Existing Content\n\n" + omnistrateSectionHeader + "Old Omnistrate content\n\n## Other Section\n\nOther content"

	if err := os.WriteFile(srcPath, []byte(srcContent), 0600); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	if err := os.WriteFile(dstPath, []byte(existingContent), 0600); err != nil {
		t.Fatalf("Failed to create destination file: %v", err)
	}

	// Merge into existing destination with Omnistrate section
	if err := mergeMarkdownFile(srcPath, dstPath); err != nil {
		t.Fatalf("mergeMarkdownFile failed: %v", err)
	}

	// Verify destination has new content and preserves other sections
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	contentStr := string(dstContent)

	if !strings.Contains(contentStr, "# Existing Content") {
		t.Errorf("Destination should preserve existing content before Omnistrate section")
	}
	if !strings.Contains(contentStr, omnistrateSectionHeader) {
		t.Errorf("Destination should contain Omnistrate section header")
	}
	if !strings.Contains(contentStr, srcContent) {
		t.Errorf("Destination should contain new source content")
	}
	if strings.Contains(contentStr, "Old Omnistrate content") {
		t.Errorf("Destination should not contain old Omnistrate content")
	}
	if !strings.Contains(contentStr, "## Other Section") {
		t.Errorf("Destination should preserve content after Omnistrate section")
	}
	if !strings.Contains(contentStr, "Other content") {
		t.Errorf("Destination should preserve content after Omnistrate section")
	}
}

func TestMergeMarkdownFile_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "src.md")
	dstPath := filepath.Join(tmpDir, "dst.md")

	srcContent := "Omnistrate content"
	existingContent := "# Existing Content\n\nSome text"

	if err := os.WriteFile(srcPath, []byte(srcContent), 0600); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	if err := os.WriteFile(dstPath, []byte(existingContent), 0600); err != nil {
		t.Fatalf("Failed to create destination file: %v", err)
	}

	// First merge
	if err := mergeMarkdownFile(srcPath, dstPath); err != nil {
		t.Fatalf("First mergeMarkdownFile failed: %v", err)
	}

	firstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file after first merge: %v", err)
	}

	// Second merge with same content (should be idempotent)
	if err := mergeMarkdownFile(srcPath, dstPath); err != nil {
		t.Fatalf("Second mergeMarkdownFile failed: %v", err)
	}

	secondContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file after second merge: %v", err)
	}

	// Content should be identical after second merge
	if string(firstContent) != string(secondContent) {
		t.Errorf("Merge is not idempotent:\nFirst:  %q\nSecond: %q", string(firstContent), string(secondContent))
	}
}

func TestSkillInstallation_PreservesOtherSkills(t *testing.T) {
	tmpDir := t.TempDir()
	destSkillsDir := filepath.Join(tmpDir, "dest-skills")
	srcSkillsDir := filepath.Join(tmpDir, "src-skills")

	// Create destination skills directory with existing skills
	existingSkills := map[string]map[string]string{
		"omnistrate-fde": {
			"SKILL.md": "FDE skill content",
		},
		"other-skill": {
			"README.md": "Other skill content",
			"config.md": "Other skill config",
		},
	}

	for skillName, files := range existingSkills {
		skillDir := filepath.Join(destSkillsDir, skillName)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("Failed to create skill directory: %v", err)
		}
		for fileName, content := range files {
			filePath := filepath.Join(skillDir, fileName)
			if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
				t.Fatalf("Failed to create skill file: %v", err)
			}
		}
	}

	// Create source skills directory with new version of one skill
	newSkills := map[string]map[string]string{
		"omnistrate-fde": {
			"SKILL.md": "Updated FDE skill content",
		},
	}

	for skillName, files := range newSkills {
		skillDir := filepath.Join(srcSkillsDir, skillName)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("Failed to create source skill directory: %v", err)
		}
		for fileName, content := range files {
			filePath := filepath.Join(skillDir, fileName)
			if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
				t.Fatalf("Failed to create source skill file: %v", err)
			}
		}
	}

	// Simulate skill installation process
	srcSkills, err := os.ReadDir(srcSkillsDir)
	if err != nil {
		t.Fatalf("Failed to read source skills: %v", err)
	}

	// Remove existing skill directories with the same name
	for _, skill := range srcSkills {
		if skill.IsDir() {
			destSkillPath := filepath.Join(destSkillsDir, skill.Name())
			if _, err := os.Stat(destSkillPath); err == nil {
				if err := os.RemoveAll(destSkillPath); err != nil {
					t.Fatalf("Failed to remove existing skill: %v", err)
				}
			}
		}
	}

	// Copy skills
	if err := copyDir(srcSkillsDir, destSkillsDir); err != nil {
		t.Fatalf("Failed to copy skills: %v", err)
	}

	// Verify omnistrate-fde was updated
	fdePath := filepath.Join(destSkillsDir, "omnistrate-fde", "SKILL.md")
	fdeContent, err := os.ReadFile(fdePath)
	if err != nil {
		t.Fatalf("Failed to read updated FDE skill: %v", err)
	}
	if string(fdeContent) != "Updated FDE skill content" {
		t.Errorf("FDE skill was not updated correctly")
	}

	// Verify other-skill was preserved
	otherSkillPath := filepath.Join(destSkillsDir, "other-skill")
	if _, err := os.Stat(otherSkillPath); os.IsNotExist(err) {
		t.Errorf("other-skill was removed but should have been preserved")
	}

	otherSkillReadme := filepath.Join(otherSkillPath, "README.md")
	otherContent, err := os.ReadFile(otherSkillReadme)
	if err != nil {
		t.Fatalf("Failed to read preserved other-skill: %v", err)
	}
	if string(otherContent) != "Other skill content" {
		t.Errorf("other-skill content was modified")
	}
}

func TestSkillInstallation_RemovesExtraFilesOnReinstall(t *testing.T) {
	tmpDir := t.TempDir()
	destSkillsDir := filepath.Join(tmpDir, "dest-skills")
	srcSkillsDir := filepath.Join(tmpDir, "src-skills")

	// Create destination skill with extra files
	skillDir := filepath.Join(destSkillsDir, "omnistrate-fde")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	oldFiles := map[string]string{
		"SKILL.md":       "Old skill content",
		"REFERENCE.md":   "Old reference",
		"extra-file.txt": "This should be removed",
	}

	for fileName, content := range oldFiles {
		filePath := filepath.Join(skillDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Create source skill with only expected files
	srcSkillDir := filepath.Join(srcSkillsDir, "omnistrate-fde")
	if err := os.MkdirAll(srcSkillDir, 0755); err != nil {
		t.Fatalf("Failed to create source skill directory: %v", err)
	}

	newFiles := map[string]string{
		"SKILL.md":     "New skill content",
		"REFERENCE.md": "New reference",
	}

	for fileName, content := range newFiles {
		filePath := filepath.Join(srcSkillDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}
	}

	// Simulate reinstall process
	srcSkills, err := os.ReadDir(srcSkillsDir)
	if err != nil {
		t.Fatalf("Failed to read source skills: %v", err)
	}

	for _, skill := range srcSkills {
		if skill.IsDir() {
			destSkillPath := filepath.Join(destSkillsDir, skill.Name())
			if _, err := os.Stat(destSkillPath); err == nil {
				if err := os.RemoveAll(destSkillPath); err != nil {
					t.Fatalf("Failed to remove existing skill: %v", err)
				}
			}
		}
	}

	if err := copyDir(srcSkillsDir, destSkillsDir); err != nil {
		t.Fatalf("Failed to copy skills: %v", err)
	}

	// Verify extra file was removed
	extraFilePath := filepath.Join(destSkillsDir, "omnistrate-fde", "extra-file.txt")
	if _, err := os.Stat(extraFilePath); !os.IsNotExist(err) {
		t.Errorf("extra-file.txt should have been removed but still exists")
	}

	// Verify expected files exist with new content
	for fileName, expectedContent := range newFiles {
		filePath := filepath.Join(destSkillsDir, "omnistrate-fde", fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", fileName, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s: got %q, want %q", fileName, string(content), expectedContent)
		}
	}
}

func TestSkillInstallation_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	destSkillsDir := filepath.Join(tmpDir, "dest-skills")
	srcSkillsDir := filepath.Join(tmpDir, "src-skills")

	// Create source skill
	srcSkillDir := filepath.Join(srcSkillsDir, "omnistrate-fde")
	if err := os.MkdirAll(srcSkillDir, 0755); err != nil {
		t.Fatalf("Failed to create source skill directory: %v", err)
	}

	skillFiles := map[string]string{
		"SKILL.md":     "Skill content",
		"REFERENCE.md": "Reference content",
	}

	for fileName, content := range skillFiles {
		filePath := filepath.Join(srcSkillDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}
	}

	// Helper function to perform install
	doInstall := func() error {
		srcSkills, err := os.ReadDir(srcSkillsDir)
		if err != nil {
			return err
		}

		for _, skill := range srcSkills {
			if skill.IsDir() {
				destSkillPath := filepath.Join(destSkillsDir, skill.Name())
				if _, err := os.Stat(destSkillPath); err == nil {
					if err := os.RemoveAll(destSkillPath); err != nil {
						return err
					}
				}
			}
		}

		return copyDir(srcSkillsDir, destSkillsDir)
	}

	// First install
	if err := doInstall(); err != nil {
		t.Fatalf("First install failed: %v", err)
	}

	// Read content after first install
	firstInstallContents := make(map[string]string)
	for fileName := range skillFiles {
		filePath := filepath.Join(destSkillsDir, "omnistrate-fde", fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file after first install: %v", err)
		}
		firstInstallContents[fileName] = string(content)
	}

	// Second install (should be idempotent)
	if err := doInstall(); err != nil {
		t.Fatalf("Second install failed: %v", err)
	}

	// Verify content is identical after second install
	for fileName, firstContent := range firstInstallContents {
		filePath := filepath.Join(destSkillsDir, "omnistrate-fde", fileName)
		secondContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file after second install: %v", err)
		}
		if string(secondContent) != firstContent {
			t.Errorf("Install is not idempotent for %s:\nFirst:  %q\nSecond: %q", fileName, firstContent, string(secondContent))
		}
	}
}

func TestMergeMarkdownFileWithContent_UpdatesExistingPaths(t *testing.T) {
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test.md")

	// Create existing file with old skill paths
	existingContent := "# My Skills\n\n**Location**: " + "`skills/omnistrate-fde/`\n\nSome content"
	if err := os.WriteFile(destPath, []byte(existingContent), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Merge with new content
	newContent := "New Omnistrate content with **Location**: " + "`.claude/skills/omnistrate-sa/`"
	if err := mergeMarkdownFileWithContent(newContent, destPath); err != nil {
		t.Fatalf("mergeMarkdownFileWithContent failed: %v", err)
	}

	// Read result
	result, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	resultStr := string(result)

	// Verify old paths were updated in existing content
	if strings.Contains(resultStr, "`skills/omnistrate-fde/`") {
		t.Errorf("Old skill path was not updated in existing content")
	}
	if !strings.Contains(resultStr, "`.claude/skills/omnistrate-fde/`") {
		t.Errorf("Existing content should have updated skill path")
	}

	// Verify new content is present with correct paths
	if !strings.Contains(resultStr, omnistrateSectionHeader) {
		t.Errorf("Omnistrate section header not found")
	}
	if !strings.Contains(resultStr, "`.claude/skills/omnistrate-sa/`") {
		t.Errorf("New content should contain updated skill path")
	}
}

func TestUpdateSkillPaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single skill location",
			input:    "**Location**: " + "`skills/omnistrate-fde/`",
			expected: "**Location**: " + "`.claude/skills/omnistrate-fde/`",
		},
		{
			name:     "Multiple skill locations",
			input:    "**Location**: " + "`skills/omnistrate-fde/`\n\nSome content here\n\n**Location**: " + "`skills/omnistrate-sa/`\n\nMore content\n\n**Location**: " + "`skills/omnistrate-sre/`",
			expected: "**Location**: " + "`.claude/skills/omnistrate-fde/`\n\nSome content here\n\n**Location**: " + "`.claude/skills/omnistrate-sa/`\n\nMore content\n\n**Location**: " + "`.claude/skills/omnistrate-sre/`",
		},
		{
			name:     "No skill locations",
			input:    "This content has no skill locations",
			expected: "This content has no skill locations",
		},
		{
			name:     "Already updated paths",
			input:    "**Location**: " + "`.claude/skills/omnistrate-fde/`",
			expected: "**Location**: " + "`.claude/skills/omnistrate-fde/`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updateSkillPaths(tt.input)
			if result != tt.expected {
				t.Errorf("updateSkillPaths() =\n%q\nwant\n%q", result, tt.expected)
			}
		})
	}
}

func TestReplaceOmnistrateSection(t *testing.T) {
	tests := []struct {
		name        string
		destContent string
		srcContent  string
		want        string
	}{
		{
			name:        "Replace section in middle",
			destContent: "# Header\n\n" + omnistrateSectionHeader + "Old content\n\n## Next Section\n\nMore content",
			srcContent:  "New content",
			want:        "# Header\n\n" + omnistrateSectionHeader + "New content\n\n## Next Section\n\nMore content",
		},
		{
			name:        "Replace section at end",
			destContent: "# Header\n\nSome text\n\n" + omnistrateSectionHeader + "Old content",
			srcContent:  "New content",
			want:        "# Header\n\nSome text\n\n" + omnistrateSectionHeader + "New content",
		},
		{
			name:        "Section not found",
			destContent: "# Header\n\nSome text",
			srcContent:  "New content",
			want:        "# Header\n\nSome text\n\n" + omnistrateSectionHeader + "New content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceOmnistrateSection(tt.destContent, tt.srcContent)
			if got != tt.want {
				t.Errorf("replaceOmnistrateSection() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

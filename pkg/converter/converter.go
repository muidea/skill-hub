package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
	"skill-hub/pkg/validator"
)

// Fix represents a single fix that can be applied to a skill
type Fix struct {
	Description string
	Apply       func(content string) (string, error)
	CanFix      bool
}

// ConversionResult represents the result of converting a skill
type ConversionResult struct {
	SkillID      string
	Original     string
	Modified     string
	AppliedFixes []string
	Errors       []string
	Warnings     []string
	BackupPath   string
}

// Converter handles automatic fixing of skill files
type Converter struct {
	validator *validator.Validator
	backupDir string
}

// NewConverter creates a new converter
func NewConverter() (*Converter, error) {
	v := validator.NewValidator()

	// Create backup directory in temp
	backupDir := filepath.Join(os.TempDir(), "skill-hub-backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &Converter{
		validator: v,
		backupDir: backupDir,
	}, nil
}

// ConvertSkill attempts to fix a skill file
func (c *Converter) ConvertSkill(skillPath string, options validator.ValidationOptions) (*ConversionResult, error) {
	// Read the skill file
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	original := string(content)
	skillID := filepath.Base(filepath.Dir(skillPath))

	// Create backup
	backupPath, err := c.createBackup(skillPath, original)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Validate the skill first
	result, err := c.validator.ValidateWithOptions(skillPath, options)
	if err != nil {
		return nil, fmt.Errorf("failed to validate skill: %w", err)
	}

	// If no issues or only warnings that can't be fixed, return early
	if !result.HasErrors() && (!result.HasWarnings() || !options.StrictMode) {
		return &ConversionResult{
			SkillID:    skillID,
			Original:   original,
			Modified:   original,
			BackupPath: backupPath,
		}, nil
	}

	// Apply fixes
	modified := original
	appliedFixes := []string{}
	errors := []string{}
	warnings := []string{}

	// Get available fixes
	fixes := c.getAvailableFixes(result)

	for _, fix := range fixes {
		if fix.CanFix {
			newContent, err := fix.Apply(modified)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to apply fix '%s': %v", fix.Description, err))
				continue
			}

			if newContent != modified {
				modified = newContent
				appliedFixes = append(appliedFixes, fix.Description)
			}
		}
	}

	// Validate again after fixes
	// Write temporary file for validation
	tempPath := filepath.Join(os.TempDir(), "skill-hub-temp-"+skillID+".md")
	if err := os.WriteFile(tempPath, []byte(modified), 0644); err != nil {
		errors = append(errors, fmt.Sprintf("failed to write temp file for validation: %v", err))
	} else {
		defer os.Remove(tempPath)

		postFixResult, err := c.validator.ValidateWithOptions(tempPath, options)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to validate after fixes: %v", err))
		} else {
			// Collect remaining errors and warnings
			for _, err := range postFixResult.Errors {
				errors = append(errors, err.Message)
			}
			for _, warn := range postFixResult.Warnings {
				warnings = append(warnings, warn.Message)
			}
		}
	}

	return &ConversionResult{
		SkillID:      skillID,
		Original:     original,
		Modified:     modified,
		AppliedFixes: appliedFixes,
		Errors:       errors,
		Warnings:     warnings,
		BackupPath:   backupPath,
	}, nil
}

// PreviewConversion shows what changes would be made without actually applying them
func (c *Converter) PreviewConversion(skillPath string, options validator.ValidationOptions) (*ConversionResult, error) {
	// Read the skill file
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	original := string(content)
	skillID := filepath.Base(filepath.Dir(skillPath))

	// Validate the skill
	result, err := c.validator.ValidateWithOptions(skillPath, options)
	if err != nil {
		return nil, fmt.Errorf("failed to validate skill: %w", err)
	}

	// If no issues, return early
	if !result.HasErrors() && (!result.HasWarnings() || !options.StrictMode) {
		return &ConversionResult{
			SkillID:  skillID,
			Original: original,
			Modified: original,
		}, nil
	}

	// Apply fixes to a copy for preview
	modified := original
	appliedFixes := []string{}
	errors := []string{}
	warnings := []string{}

	// Get available fixes
	fixes := c.getAvailableFixes(result)

	for _, fix := range fixes {
		if fix.CanFix {
			newContent, err := fix.Apply(modified)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to apply fix '%s': %v", fix.Description, err))
				continue
			}

			if newContent != modified {
				modified = newContent
				appliedFixes = append(appliedFixes, fix.Description)
			}
		}
	}

	return &ConversionResult{
		SkillID:      skillID,
		Original:     original,
		Modified:     modified,
		AppliedFixes: appliedFixes,
		Errors:       errors,
		Warnings:     warnings,
	}, nil
}

// RestoreBackup restores a skill from backup
func (c *Converter) RestoreBackup(skillPath, backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("no backup path provided")
	}

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Read backup content
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Write backup content to skill file
	if err := os.WriteFile(skillPath, backupContent, 0644); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	// Remove backup file
	if err := os.Remove(backupPath); err != nil {
		// Don't fail if we can't remove the backup
		fmt.Printf("warning: failed to remove backup file: %v\n", err)
	}

	return nil
}

// createBackup creates a backup of the original skill file
func (c *Converter) createBackup(skillPath, content string) (string, error) {
	skillName := filepath.Base(filepath.Dir(skillPath))
	backupName := fmt.Sprintf("%s-%d.md", skillName, os.Getpid())
	backupPath := filepath.Join(c.backupDir, backupName)

	if err := os.WriteFile(backupPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

// getAvailableFixes returns fixes based on validation issues
func (c *Converter) getAvailableFixes(result *validator.ValidationResult) []Fix {
	var fixes []Fix

	// Check errors
	for _, err := range result.Errors {
		switch err.Code {
		case validator.ErrMissingName:
			fixes = append(fixes, Fix{
				Description: "Add missing name field",
				Apply:       c.fixMissingName,
				CanFix:      true,
			})
		case validator.ErrNameInvalidFormat:
			fixes = append(fixes, Fix{
				Description: "Fix name format (convert to Title Case)",
				Apply:       c.fixNameFormat,
				CanFix:      true,
			})
		case validator.ErrMissingDescription:
			fixes = append(fixes, Fix{
				Description: "Add placeholder description",
				Apply:       c.fixMissingDescription,
				CanFix:      true,
			})
		}
	}

	// Check warnings for compatibility format issues
	for _, warn := range result.Warnings {
		if warn.Code == validator.WarnCompatObjectFormat {
			fixes = append(fixes, Fix{
				Description: "Convert compatibility object to string format",
				Apply:       c.fixCompatibilityFormat,
				CanFix:      true,
			})
		}
	}

	// Always check for missing metadata fields
	fixes = append(fixes, Fix{
		Description: "Add default version (1.0.0) if missing",
		Apply:       c.fixMissingVersion,
		CanFix:      true,
	})

	fixes = append(fixes, Fix{
		Description: "Add default author (unknown) if missing",
		Apply:       c.fixMissingAuthor,
		CanFix:      true,
	})

	return fixes
}

// fixMissingName adds a missing name field
func (c *Converter) fixMissingName(content string) (string, error) {
	return c.addFrontmatterField(content, "name", "Untitled Skill")
}

// fixNameFormat converts name to Title Case
func (c *Converter) fixNameFormat(content string) (string, error) {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		if strings.HasPrefix(line, "name:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentName := strings.TrimSpace(parts[1])
				// Simple title case conversion
				fixedName := toTitleCase(strings.ToLower(currentName))
				lines[i] = "name: " + fixedName
				break
			}
		}
	}

	return strings.Join(lines, "\n"), nil
}

// toTitleCase converts a string to title case
func toTitleCase(s string) string {
	if s == "" {
		return s
	}

	// Convert first character to uppercase
	result := []rune(s)
	result[0] = unicode.ToUpper(result[0])

	// Convert the rest to lowercase
	for i := 1; i < len(result); i++ {
		result[i] = unicode.ToLower(result[i])
	}

	return string(result)
}

// fixMissingDescription adds a placeholder description
func (c *Converter) fixMissingDescription(content string) (string, error) {
	return c.addFrontmatterField(content, "description", "A skill for AI coding assistants")
}

// fixCompatibilityFormat converts compatibility object to string format
func (c *Converter) fixCompatibilityFormat(content string) (string, error) {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	frontmatterEnd := -1

	// Find frontmatter boundaries
	for i, line := range lines {
		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
			} else {
				frontmatterEnd = i
				break
			}
		}
	}

	if frontmatterEnd == -1 {
		return content, fmt.Errorf("invalid frontmatter format")
	}

	// Parse frontmatter
	frontmatterLines := lines[1:frontmatterEnd]
	frontmatterContent := strings.Join(frontmatterLines, "\n")

	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatterContent), &data); err != nil {
		return content, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Check if compatibility is an object
	if compatObj, ok := data["compatibility"].(map[string]interface{}); ok {
		var compatList []string

		// Convert object to string list
		if cursorVal, ok := compatObj["cursor"].(bool); ok && cursorVal {
			compatList = append(compatList, "Cursor")
		}
		if claudeVal, ok := compatObj["claude_code"].(bool); ok && claudeVal {
			compatList = append(compatList, "Claude Code")
		}
		if openCodeVal, ok := compatObj["open_code"].(bool); ok && openCodeVal {
			compatList = append(compatList, "OpenCode")
		}
		if shellVal, ok := compatObj["shell"].(bool); ok && shellVal {
			compatList = append(compatList, "Shell")
		}

		// Create new compatibility string
		var compatString string
		if len(compatList) > 0 {
			compatString = "Designed for " + strings.Join(compatList, ", ") + " (or similar AI coding assistants)"
		} else {
			compatString = ""
		}

		// Update the compatibility field in data
		data["compatibility"] = compatString

		// Re-serialize YAML
		newYaml, err := yaml.Marshal(data)
		if err != nil {
			return content, fmt.Errorf("failed to marshal updated frontmatter: %w", err)
		}

		// Reconstruct the file
		newLines := []string{"---"}
		newLines = append(newLines, strings.Split(strings.TrimSpace(string(newYaml)), "\n")...)
		newLines = append(newLines, "---")
		newLines = append(newLines, lines[frontmatterEnd+1:]...)

		return strings.Join(newLines, "\n"), nil
	}

	return content, nil
}

// fixMissingVersion adds a default version
func (c *Converter) fixMissingVersion(content string) (string, error) {
	return c.addFrontmatterField(content, "version", "1.0.0")
}

// fixMissingAuthor adds a default author
func (c *Converter) fixMissingAuthor(content string) (string, error) {
	return c.addFrontmatterField(content, "source", "unknown")
}

// addFrontmatterField adds a field to the frontmatter
func (c *Converter) addFrontmatterField(content, field, value string) (string, error) {
	lines := strings.Split(content, "\n")

	// Check if frontmatter exists
	if len(lines) < 2 || lines[0] != "---" {
		// No frontmatter, create one
		newLines := []string{"---", fmt.Sprintf("%s: %s", field, value), "---"}
		if len(lines) > 0 {
			newLines = append(newLines, lines...)
		}
		return strings.Join(newLines, "\n"), nil
	}

	// Find end of frontmatter
	frontmatterEnd := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			frontmatterEnd = i
			break
		}
	}

	if frontmatterEnd == -1 {
		// Malformed frontmatter, add at the beginning
		newLines := []string{"---", fmt.Sprintf("%s: %s", field, value), "---"}
		newLines = append(newLines, lines...)
		return strings.Join(newLines, "\n"), nil
	}

	// Check if field already exists
	for i := 1; i < frontmatterEnd; i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), field+":") {
			// Field exists, don't add it
			return content, nil
		}
	}

	// Add field before the closing ---
	newLines := make([]string, len(lines)+1)
	copy(newLines, lines[:frontmatterEnd])
	newLines[frontmatterEnd] = fmt.Sprintf("%s: %s", field, value)
	copy(newLines[frontmatterEnd+1:], lines[frontmatterEnd:])

	return strings.Join(newLines, "\n"), nil
}

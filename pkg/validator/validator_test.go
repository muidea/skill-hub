package validator

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidator_ValidateFile(t *testing.T) {
	tests := []struct {
		name         string
		skillPath    string
		wantErrors   int
		wantWarnings int
		wantValid    bool
	}{
		{
			name:         "valid skill",
			skillPath:    "testdata/valid-skill/SKILL.md",
			wantErrors:   0,
			wantWarnings: 0,
			wantValid:    true,
		},
		{
			name:         "missing name field",
			skillPath:    "testdata/missing-name/SKILL.md",
			wantErrors:   1, // MISSING_NAME
			wantWarnings: 0,
			wantValid:    false,
		},
		{
			name:         "invalid name format",
			skillPath:    "testdata/invalid-name-format/SKILL.md",
			wantErrors:   1, // NAME_INVALID_FORMAT
			wantWarnings: 1, // DIRECTORY_MISMATCH_WARNING
			wantValid:    false,
		},
		{
			name:         "object compatibility format",
			skillPath:    "testdata/object-compatibility/SKILL.md",
			wantErrors:   0,
			wantWarnings: 1, // COMPAT_OBJECT_FORMAT
			wantValid:    true,
		},
	}

	v := NewValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 获取测试文件的绝对路径
			// 首先尝试使用相对路径
			absPath := tt.skillPath
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				// 如果相对路径不存在，尝试从测试文件所在目录查找
				_, filename, _, _ := runtime.Caller(0)
				testDir := filepath.Dir(filename)
				absPath = filepath.Join(testDir, tt.skillPath)
			}

			result, err := v.ValidateFile(absPath)
			if err != nil {
				t.Fatalf("ValidateFile() 错误 = %v, 文件路径: %s", err, absPath)
			}

			if len(result.Errors) != tt.wantErrors {
				t.Errorf("ValidateFile() 错误数量 = %v, 期望 %v", len(result.Errors), tt.wantErrors)
				for _, err := range result.Errors {
					t.Logf("错误: %s - %s", err.Code, err.Message)
				}
			}

			if len(result.Warnings) != tt.wantWarnings {
				t.Errorf("ValidateFile() 警告数量 = %v, 期望 %v", len(result.Warnings), tt.wantWarnings)
				for _, warn := range result.Warnings {
					t.Logf("警告: %s - %s", warn.Code, warn.Message)
				}
			}

			if result.IsValid != tt.wantValid {
				t.Errorf("ValidateFile() IsValid = %v, 期望 %v", result.IsValid, tt.wantValid)
			}
		})
	}
}

func TestValidator_ValidateSkill(t *testing.T) {
	tests := []struct {
		name         string
		skillName    string
		frontmatter  map[string]interface{}
		wantErrors   int
		wantWarnings int
		wantValid    bool
	}{
		{
			name:      "valid skill",
			skillName: "test-skill",
			frontmatter: map[string]interface{}{
				"name":        "test-skill",
				"description": "A valid test skill with proper formatting. This description is long enough.",
			},
			wantErrors:   0,
			wantWarnings: 1, // DIRECTORY_MISMATCH_WARNING (因为skillName是"test-skill"但路径为空)
			wantValid:    true,
		},
		{
			name:      "missing description",
			skillName: "test-skill",
			frontmatter: map[string]interface{}{
				"name": "test-skill",
			},
			wantErrors:   1, // MISSING_DESCRIPTION
			wantWarnings: 1, // DIRECTORY_MISMATCH_WARNING
			wantValid:    false,
		},
		{
			name:      "object compatibility",
			skillName: "test-skill",
			frontmatter: map[string]interface{}{
				"name":        "test-skill",
				"description": "A test skill with a proper description.",
				"compatibility": map[string]interface{}{
					"cursor":      true,
					"claude_code": false,
				},
			},
			wantErrors:   0,
			wantWarnings: 2, // DIRECTORY_MISMATCH_WARNING + COMPAT_OBJECT_FORMAT
			wantValid:    true,
		},
	}

	v := NewValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.ValidateSkill(tt.skillName, tt.frontmatter)

			if len(result.Errors) != tt.wantErrors {
				t.Errorf("ValidateSkill() 错误数量 = %v, 期望 %v", len(result.Errors), tt.wantErrors)
				for _, err := range result.Errors {
					t.Logf("错误: %s - %s", err.Code, err.Message)
				}
			}

			if len(result.Warnings) != tt.wantWarnings {
				t.Errorf("ValidateSkill() 警告数量 = %v, 期望 %v", len(result.Warnings), tt.wantWarnings)
				for _, warn := range result.Warnings {
					t.Logf("警告: %s - %s", warn.Code, warn.Message)
				}
			}

			if result.IsValid != tt.wantValid {
				t.Errorf("ValidateSkill() IsValid = %v, 期望 %v", result.IsValid, tt.wantValid)
			}
		})
	}
}

func TestValidationResult_Methods(t *testing.T) {
	result := NewValidationResult("/test/path/SKILL.md")

	if result.HasErrors() {
		t.Error("新结果不应该有错误")
	}
	if result.HasWarnings() {
		t.Error("新结果不应该有警告")
	}
	if !result.IsValid {
		t.Error("新结果应该是有效的")
	}

	error1 := NewError(ErrMissingName, "name", true)
	warning1 := NewWarning(WarnDescTooShort, "description", true)

	result.AddError(error1)
	result.AddWarning(warning1)

	if !result.HasErrors() {
		t.Error("添加错误后应该有错误")
	}
	if !result.HasWarnings() {
		t.Error("添加警告后应该有警告")
	}
	if result.IsValid {
		t.Error("有错误时应该是无效的")
	}

	fixableErrors := result.GetFixableErrors()
	if len(fixableErrors) != 1 {
		t.Errorf("GetFixableErrors() = %v, 期望 1", len(fixableErrors))
	}

	fixableWarnings := result.GetFixableWarnings()
	if len(fixableWarnings) != 1 {
		t.Errorf("GetFixableWarnings() = %v, 期望 1", len(fixableWarnings))
	}

	summary := result.Summary()
	expected := "❌ 1个错误, ⚠️  1个警告"
	if summary != expected {
		t.Errorf("Summary() = %v, 期望 %v", summary, expected)
	}
}

func TestValidator_ValidateWithOptions(t *testing.T) {
	// 获取测试文件的绝对路径
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	skillPath := filepath.Join(testDir, "testdata", "object-compatibility", "SKILL.md")

	v := NewValidator()

	t.Run("default options", func(t *testing.T) {
		result, err := v.ValidateWithOptions(skillPath, ValidationOptions{})
		if err != nil {
			t.Fatalf("ValidateWithOptions() 错误 = %v", err)
		}

		if !result.IsValid {
			t.Error("默认选项下，有警告的技能应该是有效的")
		}
		if len(result.Warnings) == 0 {
			t.Error("应该检测到警告")
		}
	})

	t.Run("strict mode", func(t *testing.T) {
		result, err := v.ValidateWithOptions(skillPath, ValidationOptions{StrictMode: true})
		if err != nil {
			t.Fatalf("ValidateWithOptions() 错误 = %v", err)
		}

		if result.IsValid {
			t.Error("严格模式下，有警告的技能应该是无效的")
		}
	})

	t.Run("ignore warnings", func(t *testing.T) {
		result, err := v.ValidateWithOptions(skillPath, ValidationOptions{IgnoreWarnings: true})
		if err != nil {
			t.Fatalf("ValidateWithOptions() 错误 = %v", err)
		}

		if len(result.Warnings) > 0 {
			t.Error("忽略警告选项应该过滤掉警告")
		}
	})
}

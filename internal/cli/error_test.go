package cli

import (
	"testing"

	"skill-hub/pkg/errors"
)

func TestErrorHandling(t *testing.T) {
	t.Run("CheckInitDependency returns structured error", func(t *testing.T) {
		// 这个测试需要未初始化的环境
		// 在实际测试中，我们会设置环境变量或使用测试工具
		// 这里我们只是验证错误类型
		err := CheckInitDependency()

		// 如果返回错误，应该是AppError类型
		if err != nil {
			if !errors.IsCode(err, errors.ErrConfigNotFound) {
				t.Errorf("Expected ErrConfigNotFound, got: %v", err)
			}

			// 验证错误详情
			code := errors.Code(err)
			if code != errors.ErrConfigNotFound {
				t.Errorf("Expected error code %s, got %s", errors.ErrConfigNotFound, code)
			}

			op := errors.Operation(err)
			if op != "CheckInitDependency" {
				t.Errorf("Expected operation 'CheckInitDependency', got %s", op)
			}
		}
	})

	t.Run("Error wrapping preserves context", func(t *testing.T) {
		// 模拟一个底层错误
		originalErr := errors.New("original error")

		// 包装错误
		wrappedErr := errors.Wrap(originalErr, "middle layer")
		doubleWrappedErr := errors.Wrap(wrappedErr, "outer layer")

		// 验证错误链
		if errors.Message(doubleWrappedErr) != "original error" {
			t.Errorf("Error message not preserved: %v", errors.Message(doubleWrappedErr))
		}

		// 验证操作上下文
		if errors.Operation(doubleWrappedErr) != "outer layer" {
			t.Errorf("Operation context not preserved: %s", errors.Operation(doubleWrappedErr))
		}
	})

	t.Run("SkillNotFound error helper", func(t *testing.T) {
		err := errors.SkillNotFound("TestOperation", "test-skill")

		if !errors.IsCode(err, errors.ErrSkillNotFound) {
			t.Errorf("Expected ErrSkillNotFound, got: %v", err)
		}

		msg := errors.Message(err)
		expectedMsg := "技能未找到: test-skill"
		if msg != expectedMsg {
			t.Errorf("Expected message '%s', got '%s'", expectedMsg, msg)
		}
	})

	t.Run("Error details can be added", func(t *testing.T) {
		err := errors.NewWithCode("TestOperation", errors.ErrValidation, "validation failed")
		appErr, ok := err.(*errors.AppError)
		if !ok {
			t.Fatal("Expected AppError type")
		}

		// 添加详情
		appErr.WithDetails(map[string]interface{}{
			"field":   "username",
			"reason":  "too short",
			"min_len": 3,
		})

		details := errors.Details(err)
		if details == nil {
			t.Fatal("Expected error details")
		}

		if details["field"] != "username" {
			t.Errorf("Expected field 'username', got %v", details["field"])
		}
	})
}

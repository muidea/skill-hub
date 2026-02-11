package errors

import (
	"fmt"
	"testing"
)

// BenchmarkErrorCreation 测试错误创建性能
func BenchmarkErrorCreation(b *testing.B) {
	b.Run("NewWithCode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewWithCode("BenchmarkOperation", ErrSystem, "test error message")
		}
	})

	b.Run("NewWithCodef", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewWithCodef("BenchmarkOperation", ErrSystem, "test error %d", i)
		}
	})

	b.Run("New", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = New(fmt.Sprintf("test error %d", i))
		}
	})
}

// BenchmarkErrorWrapping 测试错误包装性能
func BenchmarkErrorWrapping(b *testing.B) {
	baseErr := New("base error")

	b.Run("Wrap", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Wrap(baseErr, "operation context")
		}
	})

	b.Run("Wrapf", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Wrapf(baseErr, "operation context %d", i)
		}
	})

	b.Run("WrapWithCode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = WrapWithCode(baseErr, "operation", ErrSystem, "wrapped error")
		}
	})
}

// BenchmarkErrorChecking 测试错误检查性能
func BenchmarkErrorChecking(b *testing.B) {
	err := NewWithCode("TestOperation", ErrFileNotFound, "file not found")

	b.Run("IsCode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = IsCode(err, ErrFileNotFound)
		}
	})

	b.Run("Code", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Code(err)
		}
	})

	b.Run("Message", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Message(err)
		}
	})

	b.Run("Operation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Operation(err)
		}
	})
}

// BenchmarkErrorChaining 测试错误链性能
func BenchmarkErrorChaining(b *testing.B) {
	baseErr := New("level 0 error")

	b.Run("DeepChain", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := baseErr
			for j := 0; j < 10; j++ {
				err = Wrapf(err, "level %d", j+1)
			}
			_ = err
		}
	})

	b.Run("WideChain", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err1 := NewWithCode("op1", ErrSystem, "error 1")
			err2 := NewWithCode("op2", ErrFileNotFound, "error 2")
			err3 := NewWithCode("op3", ErrConfigInvalid, "error 3")
			_ = Combine(err1, err2, err3)
		}
	})
}

// BenchmarkAppErrorMethods 测试AppError方法性能
func BenchmarkAppErrorMethods(b *testing.B) {
	err := &AppError{
		Code:    ErrValidation,
		Message: "validation failed",
		Op:      "BenchmarkOperation",
		Err:     New("underlying error"),
		Details: map[string]interface{}{
			"field":   "username",
			"reason":  "too short",
			"min_len": 3,
		},
	}

	b.Run("Error", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = err.Error()
		}
	})

	b.Run("Unwrap", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = err.Unwrap()
		}
	})

	b.Run("IsCode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = err.IsCode(ErrValidation)
		}
	})

	b.Run("WithDetails", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = err.WithDetails(map[string]interface{}{
				"additional": "detail",
				"iteration":  i,
			})
		}
	})
}

// BenchmarkMultiError 测试MultiError性能
func BenchmarkMultiError(b *testing.B) {
	b.Run("MultiErrorAdd", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			multiErr := NewMultiError()
			for j := 0; j < 10; j++ {
				multiErr.Add(New(fmt.Sprintf("error %d", j)))
			}
			_ = multiErr
		}
	})

	b.Run("MultiErrorError", func(b *testing.B) {
		multiErr := NewMultiError()
		for j := 0; j < 10; j++ {
			multiErr.Add(New(fmt.Sprintf("error %d", j)))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = multiErr.Error()
		}
	})

	b.Run("Combine", func(b *testing.B) {
		errs := make([]error, 10)
		for j := 0; j < 10; j++ {
			errs[j] = New(fmt.Sprintf("error %d", j))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Combine(errs...)
		}
	})
}

// BenchmarkConcurrentErrorOperations 测试并发错误操作性能
func BenchmarkConcurrentErrorOperations(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			counter++
			// 混合错误操作
			err := NewWithCode("ConcurrentOperation", ErrSystem, fmt.Sprintf("error %d", counter))
			wrapped := Wrap(err, "wrapped")
			_ = IsCode(wrapped, ErrSystem)
			_ = Code(wrapped)
			_ = Message(wrapped)
			_ = Operation(wrapped)
		}
	})
}

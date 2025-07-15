package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationError(t *testing.T) {
	t.Run("With field", func(t *testing.T) {
		err := &ValidationError{
			Field:   "email",
			Message: "must be a valid email address",
		}

		expected := "validation error on field 'email': must be a valid email address"
		assert.Equal(t, expected, err.Error())
		assert.True(t, errors.Is(err, ErrValidation))
	})

	t.Run("Without field", func(t *testing.T) {
		err := &ValidationError{
			Message: "input is invalid",
		}

		expected := "validation error: input is invalid"
		assert.Equal(t, expected, err.Error())
		assert.True(t, errors.Is(err, ErrValidation))
	})

	t.Run("Empty field", func(t *testing.T) {
		err := &ValidationError{
			Field:   "",
			Message: "general validation error",
		}

		expected := "validation error: general validation error"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Unwrap returns ErrValidation", func(t *testing.T) {
		err := &ValidationError{
			Field:   "test",
			Message: "test error",
		}

		assert.Equal(t, ErrValidation, err.Unwrap())
	})
}

func TestNotFoundError(t *testing.T) {
	t.Run("With ID", func(t *testing.T) {
		err := &NotFoundError{
			Resource: "user",
			ID:       "123",
		}

		expected := "user with ID '123' not found"
		assert.Equal(t, expected, err.Error())
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("Without ID", func(t *testing.T) {
		err := &NotFoundError{
			Resource: "user",
		}

		expected := "user not found"
		assert.Equal(t, expected, err.Error())
		assert.True(t, errors.Is(err, ErrNotFound))
	})

	t.Run("Empty ID", func(t *testing.T) {
		err := &NotFoundError{
			Resource: "user",
			ID:       "",
		}

		expected := "user not found"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Unwrap returns ErrNotFound", func(t *testing.T) {
		err := &NotFoundError{
			Resource: "test",
			ID:       "1",
		}

		assert.Equal(t, ErrNotFound, err.Unwrap())
	})
}

func TestConflictError(t *testing.T) {
	t.Run("With field and value", func(t *testing.T) {
		err := &ConflictError{
			Resource: "user",
			Field:    "email",
			Value:    "test@example.com",
		}

		expected := "user already exists with email='test@example.com'"
		assert.Equal(t, expected, err.Error())
		assert.True(t, errors.Is(err, ErrConflict))
	})

	t.Run("Without field and value", func(t *testing.T) {
		err := &ConflictError{
			Resource: "user",
		}

		expected := "user already exists"
		assert.Equal(t, expected, err.Error())
		assert.True(t, errors.Is(err, ErrConflict))
	})

	t.Run("With field but no value", func(t *testing.T) {
		err := &ConflictError{
			Resource: "user",
			Field:    "email",
		}

		expected := "user already exists"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("With value but no field", func(t *testing.T) {
		err := &ConflictError{
			Resource: "user",
			Value:    "test@example.com",
		}

		expected := "user already exists"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Unwrap returns ErrConflict", func(t *testing.T) {
		err := &ConflictError{
			Resource: "test",
		}

		assert.Equal(t, ErrConflict, err.Unwrap())
	})
}

func TestDatabaseError(t *testing.T) {
	t.Run("With cause", func(t *testing.T) {
		cause := errors.New("connection failed")
		err := &DatabaseError{
			Operation: "create user",
			Cause:     cause,
		}

		expected := "database error during create user: connection failed"
		assert.Equal(t, expected, err.Error())
		assert.True(t, errors.Is(err, ErrDatabase))
	})

	t.Run("Without cause", func(t *testing.T) {
		err := &DatabaseError{
			Operation: "create user",
		}

		expected := "database error during create user"
		assert.Equal(t, expected, err.Error())
		assert.True(t, errors.Is(err, ErrDatabase))
	})

	t.Run("Unwrap returns ErrDatabase", func(t *testing.T) {
		err := &DatabaseError{
			Operation: "test",
		}

		assert.Equal(t, ErrDatabase, err.Unwrap())
	})
}

func TestWrapValidationError(t *testing.T) {
	t.Run("With field", func(t *testing.T) {
		err := WrapValidationError("email", "must be valid")

		var validationErr *ValidationError
		assert.True(t, errors.As(err, &validationErr))
		assert.Equal(t, "email", validationErr.Field)
		assert.Equal(t, "must be valid", validationErr.Message)
		assert.True(t, errors.Is(err, ErrValidation))
	})

	t.Run("Without field", func(t *testing.T) {
		err := WrapValidationError("", "general error")

		var validationErr *ValidationError
		assert.True(t, errors.As(err, &validationErr))
		assert.Equal(t, "", validationErr.Field)
		assert.Equal(t, "general error", validationErr.Message)
	})
}

func TestWrapNotFoundError(t *testing.T) {
	err := WrapNotFoundError("user", "123")

	var notFoundErr *NotFoundError
	assert.True(t, errors.As(err, &notFoundErr))
	assert.Equal(t, "user", notFoundErr.Resource)
	assert.Equal(t, "123", notFoundErr.ID)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestWrapConflictError(t *testing.T) {
	err := WrapConflictError("user", "email", "test@example.com")

	var conflictErr *ConflictError
	assert.True(t, errors.As(err, &conflictErr))
	assert.Equal(t, "user", conflictErr.Resource)
	assert.Equal(t, "email", conflictErr.Field)
	assert.Equal(t, "test@example.com", conflictErr.Value)
	assert.True(t, errors.Is(err, ErrConflict))
}

func TestWrapDatabaseError(t *testing.T) {
	t.Run("With cause", func(t *testing.T) {
		cause := errors.New("connection failed")
		err := WrapDatabaseError("create user", cause)

		var dbErr *DatabaseError
		assert.True(t, errors.As(err, &dbErr))
		assert.Equal(t, "create user", dbErr.Operation)
		assert.Equal(t, cause, dbErr.Cause)
		assert.True(t, errors.Is(err, ErrDatabase))
	})

	t.Run("Without cause", func(t *testing.T) {
		err := WrapDatabaseError("create user", nil)

		var dbErr *DatabaseError
		assert.True(t, errors.As(err, &dbErr))
		assert.Equal(t, "create user", dbErr.Operation)
		assert.Nil(t, dbErr.Cause)
	})
}

func TestIsValidationError(t *testing.T) {
	t.Run("True for ValidationError", func(t *testing.T) {
		err := WrapValidationError("field", "message")
		assert.True(t, IsValidationError(err))
	})

	t.Run("False for other errors", func(t *testing.T) {
		err := WrapNotFoundError("resource", "id")
		assert.False(t, IsValidationError(err))
	})

	t.Run("False for nil", func(t *testing.T) {
		assert.False(t, IsValidationError(nil))
	})

	t.Run("False for generic error", func(t *testing.T) {
		err := errors.New("generic error")
		assert.False(t, IsValidationError(err))
	})
}

func TestIsNotFoundError(t *testing.T) {
	t.Run("True for NotFoundError", func(t *testing.T) {
		err := WrapNotFoundError("resource", "id")
		assert.True(t, IsNotFoundError(err))
	})

	t.Run("False for other errors", func(t *testing.T) {
		err := WrapValidationError("field", "message")
		assert.False(t, IsNotFoundError(err))
	})

	t.Run("False for nil", func(t *testing.T) {
		assert.False(t, IsNotFoundError(nil))
	})
}

func TestIsConflictError(t *testing.T) {
	t.Run("True for ConflictError", func(t *testing.T) {
		err := WrapConflictError("resource", "field", "value")
		assert.True(t, IsConflictError(err))
	})

	t.Run("False for other errors", func(t *testing.T) {
		err := WrapValidationError("field", "message")
		assert.False(t, IsConflictError(err))
	})

	t.Run("False for nil", func(t *testing.T) {
		assert.False(t, IsConflictError(nil))
	})
}

func TestIsDatabaseError(t *testing.T) {
	t.Run("True for DatabaseError", func(t *testing.T) {
		err := WrapDatabaseError("operation", errors.New("cause"))
		assert.True(t, IsDatabaseError(err))
	})

	t.Run("False for other errors", func(t *testing.T) {
		err := WrapValidationError("field", "message")
		assert.False(t, IsDatabaseError(err))
	})

	t.Run("False for nil", func(t *testing.T) {
		assert.False(t, IsDatabaseError(nil))
	})
}

func TestToMCPError(t *testing.T) {
	t.Run("Nil error returns nil", func(t *testing.T) {
		result := ToMCPError(nil)
		assert.Nil(t, result)
	})

	t.Run("Returns error as-is for now", func(t *testing.T) {
		originalErr := errors.New("test error")
		result := ToMCPError(originalErr)
		assert.Equal(t, originalErr, result)
	})

	t.Run("ValidationError returns error as-is", func(t *testing.T) {
		originalErr := WrapValidationError("field", "message")
		result := ToMCPError(originalErr)
		assert.Equal(t, originalErr, result)
	})

	t.Run("NotFoundError returns error as-is", func(t *testing.T) {
		originalErr := WrapNotFoundError("resource", "id")
		result := ToMCPError(originalErr)
		assert.Equal(t, originalErr, result)
	})

	t.Run("ConflictError returns error as-is", func(t *testing.T) {
		originalErr := WrapConflictError("resource", "field", "value")
		result := ToMCPError(originalErr)
		assert.Equal(t, originalErr, result)
	})

	t.Run("DatabaseError returns error as-is", func(t *testing.T) {
		originalErr := WrapDatabaseError("operation", errors.New("cause"))
		result := ToMCPError(originalErr)
		assert.Equal(t, originalErr, result)
	})
}

func TestRequiredFieldError(t *testing.T) {
	err := RequiredFieldError("email")

	var validationErr *ValidationError
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "email", validationErr.Field)
	assert.Equal(t, "field is required", validationErr.Message)
	assert.True(t, IsValidationError(err))

	expectedMessage := "validation error on field 'email': field is required"
	assert.Equal(t, expectedMessage, err.Error())
}

func TestInvalidFieldError(t *testing.T) {
	err := InvalidFieldError("age", "must be a positive number")

	var validationErr *ValidationError
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "age", validationErr.Field)
	assert.Equal(t, "must be a positive number", validationErr.Message)
	assert.True(t, IsValidationError(err))

	expectedMessage := "validation error on field 'age': must be a positive number"
	assert.Equal(t, expectedMessage, err.Error())
}

func TestErrorUnwrapping(t *testing.T) {
	t.Run("ValidationError unwraps to ErrValidation", func(t *testing.T) {
		err := WrapValidationError("field", "message")
		assert.True(t, errors.Is(err, ErrValidation))
		assert.False(t, errors.Is(err, ErrNotFound))
		assert.False(t, errors.Is(err, ErrConflict))
		assert.False(t, errors.Is(err, ErrDatabase))
	})

	t.Run("NotFoundError unwraps to ErrNotFound", func(t *testing.T) {
		err := WrapNotFoundError("resource", "id")
		assert.True(t, errors.Is(err, ErrNotFound))
		assert.False(t, errors.Is(err, ErrValidation))
		assert.False(t, errors.Is(err, ErrConflict))
		assert.False(t, errors.Is(err, ErrDatabase))
	})

	t.Run("ConflictError unwraps to ErrConflict", func(t *testing.T) {
		err := WrapConflictError("resource", "field", "value")
		assert.True(t, errors.Is(err, ErrConflict))
		assert.False(t, errors.Is(err, ErrValidation))
		assert.False(t, errors.Is(err, ErrNotFound))
		assert.False(t, errors.Is(err, ErrDatabase))
	})

	t.Run("DatabaseError unwraps to ErrDatabase", func(t *testing.T) {
		err := WrapDatabaseError("operation", errors.New("cause"))
		assert.True(t, errors.Is(err, ErrDatabase))
		assert.False(t, errors.Is(err, ErrValidation))
		assert.False(t, errors.Is(err, ErrNotFound))
		assert.False(t, errors.Is(err, ErrConflict))
	})
}

func TestErrorChaining(t *testing.T) {
	t.Run("Wrapped errors maintain chain", func(t *testing.T) {
		cause := errors.New("original error")
		dbErr := WrapDatabaseError("create user", cause)

		// Should be able to unwrap to find the original error
		var originalErr *DatabaseError
		assert.True(t, errors.As(dbErr, &originalErr))
		assert.Equal(t, "create user", originalErr.Operation)
		assert.Equal(t, cause, originalErr.Cause)
	})

	t.Run("Multiple error types can be checked", func(t *testing.T) {
		err := WrapValidationError("email", "invalid format")

		// Should match ValidationError and ErrValidation
		var validationErr *ValidationError
		assert.True(t, errors.As(err, &validationErr))
		assert.True(t, errors.Is(err, ErrValidation))

		// Should not match other types
		var notFoundErr *NotFoundError
		assert.False(t, errors.As(err, &notFoundErr))
		assert.False(t, errors.Is(err, ErrNotFound))
	})
}

func TestErrorMessageFormatting(t *testing.T) {
	t.Run("ValidationError formats correctly", func(t *testing.T) {
		tests := []struct {
			field    string
			message  string
			expected string
		}{
			{
				field:    "email",
				message:  "must be valid",
				expected: "validation error on field 'email': must be valid",
			},
			{
				field:    "",
				message:  "general error",
				expected: "validation error: general error",
			},
		}

		for _, tt := range tests {
			err := WrapValidationError(tt.field, tt.message)
			assert.Equal(t, tt.expected, err.Error())
		}
	})

	t.Run("NotFoundError formats correctly", func(t *testing.T) {
		tests := []struct {
			resource string
			id       string
			expected string
		}{
			{
				resource: "user",
				id:       "123",
				expected: "user with ID '123' not found",
			},
			{
				resource: "user",
				id:       "",
				expected: "user not found",
			},
		}

		for _, tt := range tests {
			err := WrapNotFoundError(tt.resource, tt.id)
			assert.Equal(t, tt.expected, err.Error())
		}
	})

	t.Run("ConflictError formats correctly", func(t *testing.T) {
		tests := []struct {
			resource string
			field    string
			value    string
			expected string
		}{
			{
				resource: "user",
				field:    "email",
				value:    "test@example.com",
				expected: "user already exists with email='test@example.com'",
			},
			{
				resource: "user",
				field:    "",
				value:    "",
				expected: "user already exists",
			},
		}

		for _, tt := range tests {
			err := WrapConflictError(tt.resource, tt.field, tt.value)
			assert.Equal(t, tt.expected, err.Error())
		}
	})

	t.Run("DatabaseError formats correctly", func(t *testing.T) {
		tests := []struct {
			operation string
			cause     error
			expected  string
		}{
			{
				operation: "create user",
				cause:     errors.New("connection failed"),
				expected:  "database error during create user: connection failed",
			},
			{
				operation: "create user",
				cause:     nil,
				expected:  "database error during create user",
			},
		}

		for _, tt := range tests {
			err := WrapDatabaseError(tt.operation, tt.cause)
			assert.Equal(t, tt.expected, err.Error())
		}
	})
}
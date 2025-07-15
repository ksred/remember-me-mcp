package utils

import (
	"errors"
	"fmt"

	// "github.com/mark3labs/mcp-go/mcp"
)

// Custom error types
var (
	// ErrValidation is returned when input validation fails
	ErrValidation = errors.New("validation error")
	
	// ErrNotFound is returned when a requested resource is not found
	ErrNotFound = errors.New("not found")
	
	// ErrConflict is returned when there's a conflict with existing data
	ErrConflict = errors.New("conflict")
	
	// ErrDatabase is returned when there's a database operation error
	ErrDatabase = errors.New("database error")
)

// ValidationError represents an error that occurs during input validation
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

// NotFoundError represents an error when a resource is not found
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s with ID '%s' not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

func (e *NotFoundError) Unwrap() error {
	return ErrNotFound
}

// ConflictError represents an error when there's a conflict with existing data
type ConflictError struct {
	Resource string
	Field    string
	Value    string
}

func (e *ConflictError) Error() string {
	if e.Field != "" && e.Value != "" {
		return fmt.Sprintf("%s already exists with %s='%s'", e.Resource, e.Field, e.Value)
	}
	return fmt.Sprintf("%s already exists", e.Resource)
}

func (e *ConflictError) Unwrap() error {
	return ErrConflict
}

// DatabaseError represents an error that occurs during database operations
type DatabaseError struct {
	Operation string
	Cause     error
}

func (e *DatabaseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("database error during %s: %v", e.Operation, e.Cause)
	}
	return fmt.Sprintf("database error during %s", e.Operation)
}

func (e *DatabaseError) Unwrap() error {
	return ErrDatabase
}

// Error wrapping functions

// WrapValidationError wraps an error as a validation error
func WrapValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// WrapNotFoundError wraps an error as a not found error
func WrapNotFoundError(resource, id string) error {
	return &NotFoundError{
		Resource: resource,
		ID:       id,
	}
}

// WrapConflictError wraps an error as a conflict error
func WrapConflictError(resource, field, value string) error {
	return &ConflictError{
		Resource: resource,
		Field:    field,
		Value:    value,
	}
}

// WrapDatabaseError wraps an error as a database error
func WrapDatabaseError(operation string, cause error) error {
	return &DatabaseError{
		Operation: operation,
		Cause:     cause,
	}
}

// Error checking functions

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	return errors.Is(err, ErrValidation)
}

// IsNotFoundError checks if an error is a not found error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsConflictError checks if an error is a conflict error
func IsConflictError(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsDatabaseError checks if an error is a database error
func IsDatabaseError(err error) bool {
	return errors.Is(err, ErrDatabase)
}

// ToMCPError converts our custom errors to appropriate MCP error responses
func ToMCPError(err error) error {
	if err == nil {
		return nil
	}

	// Temporarily return the error as is until MCP package is properly configured
	return err
	
	// TODO: Uncomment when MCP package functions are available
	// Check for specific error types and convert to appropriate MCP errors
	// switch {
	// case IsValidationError(err):
	// 	var validationErr *ValidationError
	// 	if errors.As(err, &validationErr) {
	// 		return mcp.NewError(
	// 			mcp.ErrInvalidParams,
	// 			validationErr.Error(),
	// 			nil,
	// 		)
	// 	}
	// 	return mcp.NewError(
	// 		mcp.ErrInvalidParams,
	// 		err.Error(),
	// 		nil,
	// 	)

	// case IsNotFoundError(err):
	// 	var notFoundErr *NotFoundError
	// 	if errors.As(err, &notFoundErr) {
	// 		return mcp.NewError(
	// 			mcp.ErrResourceNotFound,
	// 			notFoundErr.Error(),
	// 			nil,
	// 		)
	// 	}
	// 	return mcp.NewError(
	// 		mcp.ErrResourceNotFound,
	// 		err.Error(),
	// 		nil,
	// 	)

	// case IsConflictError(err):
	// 	var conflictErr *ConflictError
	// 	if errors.As(err, &conflictErr) {
	// 		return mcp.NewError(
	// 			mcp.ErrResourceAlreadyExists,
	// 			conflictErr.Error(),
	// 			nil,
	// 		)
	// 	}
	// 	return mcp.NewError(
	// 		mcp.ErrResourceAlreadyExists,
	// 		err.Error(),
	// 		nil,
	// 	)

	// case IsDatabaseError(err):
	// 	var dbErr *DatabaseError
	// 	if errors.As(err, &dbErr) {
	// 		return mcp.NewError(
	// 			mcp.ErrInternalError,
	// 			fmt.Sprintf("Internal server error: %s", dbErr.Operation),
	// 			nil,
	// 		)
	// 	}
	// 	return mcp.NewError(
	// 		mcp.ErrInternalError,
	// 		"Internal server error",
	// 		nil,
	// 	)

	// default:
	// 	// For any other errors, return a generic internal error
	// 	return mcp.NewError(
	// 		mcp.ErrInternalError,
	// 		"Internal server error",
	// 		nil,
	// 	)
	// }
}

// Helper function to create a validation error for required fields
func RequiredFieldError(field string) error {
	return WrapValidationError(field, "field is required")
}

// Helper function to create a validation error for invalid field values
func InvalidFieldError(field, reason string) error {
	return WrapValidationError(field, reason)
}
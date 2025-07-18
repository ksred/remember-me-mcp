package api

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation successful"`
}
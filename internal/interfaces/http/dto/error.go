package dto

// ErrorBody describes a single API error.
type ErrorBody struct {
	Code    string `json:"code" example:"validation_error"`
	Message string `json:"message" example:"task title is required"`
}

// ErrorResponse is the uniform error envelope.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

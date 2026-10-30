package dto

// ErrorBody describes a single API error.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse is the uniform error envelope.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

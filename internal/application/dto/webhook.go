package dto

// WebhookResponse represents the response from webhook processing.
type WebhookResponse struct {
	Status    string `json:"status" example:"processed"`
	ErrorCode string `json:"error_code,omitempty" example:"DISPATCH_FAILED"`
	Message   string `json:"message,omitempty" example:""`
	RetrySafe bool   `json:"retry_safe,omitempty" example:"false"`
}

// WebhookHealthResponse represents the webhook health check response.
type WebhookHealthResponse struct {
	Status             string   `json:"status" example:"healthy"`
	IdempotencyEnabled bool     `json:"idempotency_enabled" example:"true"`
	RegisteredHandlers []string `json:"registered_handlers" example:"issues,issue_comment"`
}
package models

type HealthResponse struct {
	Status string `json:"status,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

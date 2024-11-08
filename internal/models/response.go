package models

// APIResponse defines the standard structure for all API responses
type APIResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Info    interface{} `json:"info,omitempty"` // Info can hold any type of data (e.g., list or object)
}

package public

import "net/http"

// Handler struct for public-related endpoints
type Handler struct{}

// NewHandler creates a new public handler
func NewHandler() *Handler {
	return &Handler{}
}

// Endpoint1 is an example public endpoint
func (h *Handler) Endpoint1(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Public endpoint response"))
}

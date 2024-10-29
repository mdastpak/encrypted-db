package user

import "net/http"

// Handler struct for user-related endpoints
type Handler struct{}

// NewHandler creates a new user handler
func NewHandler() *Handler {
	return &Handler{}
}

// GetProfile handles GET requests to fetch user profile
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User profile data"))
}

// UpdateProfile handles PUT requests to update user profile
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User profile updated"))
}

// Package http provides HTTP response helpers.
package http

import (
	"encoding/json"
	"net/http"
)

// RespondJSON writes a JSON response
func RespondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// RespondError writes a JSON error response
func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, map[string]string{"detail": message})
}

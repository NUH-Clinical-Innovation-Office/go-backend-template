package http

import (
	"encoding/json"
	"errors"
	"net/http"
)

// ErrBodyTooLarge is returned when a request body exceeds MaxBytesReader's cap.
var ErrBodyTooLarge = errors.New("request body too large")

// DecodeJSON reads at most max bytes from r.Body and decodes JSON into dst.
// Returns ErrBodyTooLarge on size overflow; the caller is expected to map
// it to http.StatusRequestEntityTooLarge.
func DecodeJSON(w http.ResponseWriter, r *http.Request, max int64, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, max)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return ErrBodyTooLarge
		}
		return err
	}
	return nil
}

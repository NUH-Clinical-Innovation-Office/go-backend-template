//go:build integration

package integration

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/require"
	"github.com/your-org/go-backend-template/internal/api"
)

var (
	contractOnce   sync.Once
	contractRouter routers.Router
	contractErr    error
)

// loadContractRouter parses the embedded api/openapi.yaml once per package run.
func loadContractRouter(t *testing.T) routers.Router {
	t.Helper()
	contractOnce.Do(func() {
		doc, err := api.GetSpec()
		if err != nil {
			contractErr = err
			return
		}
		if err = doc.Validate(context.Background()); err != nil {
			contractErr = err
			return
		}
		contractRouter, contractErr = gorillamux.NewRouter(doc)
	})
	require.NoError(t, contractErr, "load api/openapi.yaml")
	return contractRouter
}

// withResponseContract checks every /api/v1 response the router writes against
// api/openapi.yaml, so a handler that drifts from the published contract fails
// the test that exercised it. Undocumented status codes fail too.
//
// Only responses are checked. The router's own request validator already
// checks requests, and tests send malformed requests on purpose to prove they
// are rejected. A request to an /api/v1 route the spec does not describe is
// reported, since consumers cannot generate a client for it.
func withResponseContract(t *testing.T, next http.Handler) http.Handler {
	t.Helper()
	specRouter := loadContractRouter(t)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Operational routes (/, /health, /openapi.yaml) are outside the spec.
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
			next.ServeHTTP(w, r)
			return
		}

		var reqBody []byte
		if r.Body != nil {
			reqBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(reqBody))
		}

		tee := &teeResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(tee, r)

		probe := r.Clone(context.Background())
		probe.Body = io.NopCloser(bytes.NewReader(reqBody))
		route, pathParams, err := specRouter.FindRoute(probe)
		if err != nil {
			t.Errorf("contract: %s %s is not described by api/openapi.yaml: %v", r.Method, r.URL.Path, err)
			return
		}

		input := &openapi3filter.ResponseValidationInput{
			RequestValidationInput: &openapi3filter.RequestValidationInput{
				Request:    probe,
				PathParams: pathParams,
				Route:      route,
			},
			Status: tee.status,
			Header: tee.Header(),
			Body:   io.NopCloser(bytes.NewReader(tee.body.Bytes())),
			Options: &openapi3filter.Options{
				IncludeResponseStatus: true,
				MultiError:            true,
				AuthenticationFunc:    openapi3filter.NoopAuthenticationFunc,
			},
		}
		if err := openapi3filter.ValidateResponse(context.Background(), input); err != nil {
			t.Errorf("contract: %s %s -> %d does not match api/openapi.yaml: %v",
				r.Method, route.Path, tee.status, err)
		}
	})
}

// teeResponseWriter copies the status and body while passing both through.
type teeResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	body        bytes.Buffer
}

func (w *teeResponseWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.status = status
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *teeResponseWriter) Write(b []byte) (int, error) {
	w.wroteHeader = true
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

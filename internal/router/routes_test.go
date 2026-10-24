package router

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/your-org/go-backend-template/internal/api"
	"github.com/your-org/go-backend-template/internal/config"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

// apiPrefix is where the spec's operations are mounted. It must match the
// single server URL in api/openapi.yaml.
const apiPrefix = "/api/v1"

// TestSpecMatchesRouter fails when a /api/v1 route is mounted in chi but not
// described by api/openapi.yaml, or described but not mounted. Routes are wired
// to the generated wrapper by hand to keep per-group middleware, so a typo or
// a forgotten route still compiles; this test is what catches it.
func TestSpecMatchesRouter(t *testing.T) {
	swagger, err := api.GetSpec()
	if err != nil {
		t.Fatalf("GetSpec: %v", err)
	}
	if len(swagger.Servers) != 1 || swagger.Servers[0].URL != apiPrefix {
		t.Fatalf("spec servers = %v, want exactly %q", swagger.Servers, apiPrefix)
	}

	var specRoutes []string
	for path, item := range swagger.Paths.Map() {
		for method := range item.Operations() {
			specRoutes = append(specRoutes, method+" "+apiPrefix+path)
		}
	}

	r := New(
		zap.NewNop(),
		noop.NewTracerProvider().Tracer("test"),
		nil, nil, nil,
		&config.CORSConfig{},
		config.RateLimitConfig{},
		nil,
		false,
		nil,
	)

	var chiRoutes []string
	err = chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(route, apiPrefix+"/") {
			chiRoutes = append(chiRoutes, method+" "+route)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	for _, route := range chiRoutes {
		if !slices.Contains(specRoutes, route) {
			t.Errorf("%s is mounted in the router but not described by api/openapi.yaml", route)
		}
	}
	for _, route := range specRoutes {
		if !slices.Contains(chiRoutes, route) {
			t.Errorf("%s is described by api/openapi.yaml but not mounted in the router", route)
		}
	}
}

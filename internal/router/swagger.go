//go:build swagger

package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/your-org/go-backend-template/docs/swagger" // generated swagger docs
)

// mountSwagger registers the /swagger UI routes when built with the
// "swagger" build tag. Use the SWAGGER_BUILD docker build arg to enable.
func mountSwagger(r chi.Router, enabled bool) {
	if !enabled {
		return
	}

	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.RequestURI+"/", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
}

package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HealthChecker interface {
	Health(ctx context.Context) error
}

func Run(checkers ...HealthChecker) {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		for _, checker := range checkers {
			if err := checker.Health(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				render.JSON(w, r, map[string]string{"status": "unhealthy", "error": err.Error()})
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "ok"})

	})

	r.Handle("/metrics", promhttp.Handler())

	if err := http.ListenAndServe(":8000", r); err != nil {
		log.Fatal(err)
	}
}

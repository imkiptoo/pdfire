package server

import (
	"bytes"
	"net/http"

	"github.com/modernice/pdfire"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/unrolled/render"
)

// New returns a new PDFire server.
func New() *chi.Mux {
	router := chi.NewRouter()

	router.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
	)

	router.Post("/conversions", func(w http.ResponseWriter, r *http.Request) {
		render := render.New()
		options, err := pdfire.NewConversionOptionsFromJSON(r.Body)

		if err != nil {
			render.JSON(w, 400, map[string]interface{}{
				"error": err.Error(),
			})

			return
		}

		buf := bytes.NewBuffer(make([]byte, 0))
		err = pdfire.Convert(r.Context(), buf, options)

		if err != nil {
			render.JSON(w, 400, map[string]interface{}{
				"error": err.Error(),
			})

			return
		}

		render.Data(w, 201, buf.Bytes())
	})

	return router
}

package server

import (
	"net/http"

	"github.com/a-h/templ"
)

func renderComponent(w http.ResponseWriter, r *http.Request, component templ.Component) {
	if component == nil {
		panic("renderComponent without a component")
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

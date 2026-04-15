package httpserver

import (
"net/http"

"github.com/go-chi/chi/v5"
)

type Router = http.Handler

func NewRouter() Router {
r := chi.NewRouter()

r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
_, _ = w.Write([]byte(`{"status":"ok"}`))
})

return r
}

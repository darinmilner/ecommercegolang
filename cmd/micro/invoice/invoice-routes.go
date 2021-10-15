package main

import (
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer)

	mux.Use(middleware.Logger)

	mux.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Vary", "Authorization", "Content-Type", "X-CSRF-Token", "XMLHttpRequest", "Access-Control-Allow-Origin", "Origin"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	mux.Get("/", app.HealthCheck)
	mux.Post("/invoice/create-and-send", app.CreateAndSendInvoice)

	return mux

}

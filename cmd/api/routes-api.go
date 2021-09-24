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

	mux.Post("/api/payment-intent", app.GetPaymentIntent)
	mux.Post("/api/create-customer-and-subscribe", app.CreateCustomerAndSubscribe)

	mux.Get("/api/widget/{id}", app.GetWidgetByID)

	//Auth Routes
	mux.Post("/api/authenticate", app.CreateAuthToken)
	mux.Post("/api/is-authenticated", app.CheckAuthenticated)
	mux.Post("/api/register", app.Register)
	mux.Post("/api/forgot-password", app.SendPasswordResetEmail)
	mux.Post("/api/reset-password", app.ResetPassword)

	mux.Route("/api/admin", func(mux chi.Router) {
		mux.Use(app.Auth)

		mux.Post("/virtual-terminal-succeeded", app.VirtualTerminalPaymentSucceeded)
		mux.Post("/all-sales", app.AllSales)
		mux.Post("/all-subscriptions", app.AllSubscriptions)

		mux.Post("/get-sale/{id}", app.GetSale)
	})
	return mux

}

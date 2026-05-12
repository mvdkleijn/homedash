/*
	HomeDash - A simple, automated dashboard for home labs.
	Copyright (C) 2023-2024  Martijn van der Kleijn

	This file is part of HomeDash.

	This Source Code Form is subject to the terms of the Mozilla Public
	License, v. 2.0. If a copy of the MPL was not distributed with this
	file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package main

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mvdkleijn/homedash/internal/config"
	c "github.com/mvdkleijn/homedash/internal/config"
	"github.com/mvdkleijn/homedash/internal/routes"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

//go:embed static/*
var staticFS embed.FS

// LoggingMiddleware is a custom middleware that uses our global Logger
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// We use a custom ResponseWriter to "intercept" the status code
		// so we can log it. Standard http.ResponseWriter doesn't let us see it.
		wrappedWriter := &responseWriterInterceptor{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler in the chain
		next.ServeHTTP(wrappedWriter, r)

		// After the request is finished, log the details using our global Logger
		duration := time.Since(start)

		c.Logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrappedWriter.statusCode).
			Dur("duration", duration).
			Str("remote_addr", r.RemoteAddr).
			Msg("request processed")
	})
}

// responseWriterInterceptor is a helper to capture the HTTP status code
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterInterceptor) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	config.Setup()

	r := mux.NewRouter()

	// Panic recovery
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	})

	r.Use(LoggingMiddleware)
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   c.Config.Cors.AllowedOrigins,
		AllowedMethods:   c.Config.Cors.AllowedMethods,
		AllowedHeaders:   c.Config.Cors.AllowedHeaders,
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: c.Config.Cors.AllowCredentials,
		MaxAge:           int(time.Duration.Seconds(12 * time.Hour)),
	}).Handler)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/", http.FileServer(http.FS(staticFS))))

	api := r.PathPrefix("/api").Subrouter()
	v1 := &routes.V1{}
	v1.AddRoutes(api)

	r.HandleFunc("/icons/{filename}", routes.ServeIcon).Methods("GET")
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		indexHtml, err := staticFS.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(indexHtml)
	})

	// Check for old data and clean up every X minutes
	go func() {
		for {
			time.Sleep(time.Duration(c.Config.CleanCheckInterval) * time.Minute)
			routes.DataStore.CleanupOutdatedEntries(c.Config.MaxAgeBeforeCleanup)
		}
	}()

	address := fmt.Sprintf("%s:%s", c.Config.Server.Address, c.Config.Server.Port)
	c.Logger.Info().Str("address", address).Msg("starting server")

	server := &http.Server{
		Addr:    address,
		Handler: r,
	}

	// Channel to listen for interrupt signals (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Run server in a goroutine so it doesn't block
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			config.Logger.Fatal().Err(err).Msg("error trying to serve data")
		}
	}()

	// Wait for interrupt signal
	<-quit
	config.Logger.Info().Msg("shutting down server...")

	// Create a context with a timeout for the shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		config.Logger.Fatal().Err(err).Msg("server forced to shutdown")
	}

	config.Logger.Info().Msg("server exiting")
}

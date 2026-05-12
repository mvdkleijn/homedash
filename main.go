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
	"strconv"
	"strings"
	"syscall"
	"time"

	c "github.com/mvdkleijn/homedash/internal/config"
	"github.com/mvdkleijn/homedash/internal/routes"
)

//go:embed static
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

// SimpleCorsMiddleware replaces github.com/rs/cors
func SimpleCorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//TODO add check: if allow credentials are true then ensure origin is not *
		// Set CORS headers based on config
		w.Header().Set("Access-Control-Allow-Origin", strings.Join(c.Config.Cors.AllowedOrigins, ","))
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.Config.Cors.AllowedMethods, ","))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.Config.Cors.AllowedHeaders, ","))
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		w.Header().Set("Access-Control-Allow-Credentials", strconv.FormatBool(c.Config.Cors.AllowCredentials))

		//TODO make this settable?
		maxAge := uint(12 * time.Hour / time.Second)
		w.Header().Set("Access-Control-Max-Age", strconv.FormatUint(uint64(maxAge), 10))

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware replaces the r.Use(func...) logic
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				c.Logger.Error().Err(fmt.Errorf("%v", err)).Msg("panic recovered")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func main() {
	c.Setup()

	// Create the base mux
	mux := http.NewServeMux()

	// Define the API routes (v1)
	v1 := &routes.V1{}
	// We pass the mux itself, and V1 will register routes with "GET /v1/..."
	if err := v1.AddRoutes(mux); err != nil {
		c.Logger.Fatal().Err(err).Msg("failed to initialize routes")
	}

	// Define Static Assets
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("GET /static/", fileServer)

	// Define Icon route
	mux.HandleFunc("GET /icons/{filename}", routes.ServeIcon)

	// Define Index route
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		indexHtml, err := staticFS.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(indexHtml)
	})

	// Wrap the entire mux with Middleware (The "Chain")
	// The order is: Recovery -> Logging -> CORS -> Mux
	var handler http.Handler = mux
	handler = SimpleCorsMiddleware(handler)
	handler = LoggingMiddleware(handler)
	handler = RecoveryMiddleware(handler)

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
		Handler: handler,
	}

	// Channel to listen for interrupt signals (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Run server in a goroutine so it doesn't block
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.Logger.Fatal().Err(err).Msg("error trying to serve data")
		}
	}()

	// Wait for interrupt signal
	<-quit
	c.Logger.Info().Msg("shutting down server...")

	// Create a context with a timeout for the shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		c.Logger.Fatal().Err(err).Msg("server forced to shutdown")
	}

	c.Logger.Info().Msg("server exiting")
}

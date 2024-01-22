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
	"embed"
	"fmt"
	"net/http"
	"os"
	"time"

	c "github.com/mvdkleijn/homedash/internal/config"
	"github.com/mvdkleijn/homedash/internal/routes"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
)

//go:embed static/*
var staticFS embed.FS

// Logger is a middleware function that logs each request to the given Logrus logger instance
func Logger(logger *zerolog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info().Str("remoteaddr", r.RemoteAddr).Str("protocol", r.Proto).Str("method", r.Method).Str("url", r.URL.String()).Msg("request received")

			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

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

	r.Use(Logger(&log))
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
			time.Sleep(time.Duration(c.Config.Global.CleanCheckInterval) * time.Minute)
			routes.DataStore.CleanupOutdatedEntries(c.Config.Global.MaxAgeBeforeCleanup)
		}
	}()

	address := fmt.Sprintf("%s:%s", c.Config.Global.ServerAddress, c.Config.Global.ServerPort)
	log.Info().Str("address", address).Msg("starting server")
	err := http.ListenAndServe(address, r)
	if err != nil {
		log.Debug().Err(err).Msg("error trying to serve data")
	}
}

/*
	HomeDash - A simple, automated dashboard for home labs.
	Copyright (C) 2023  Martijn van der Kleijn

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
	"time"

	c "github.com/mvdkleijn/homedash/internal/config"
	"github.com/mvdkleijn/homedash/internal/routes"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

//go:embed static/*
var staticFS embed.FS

// Logger is a middleware function that logs each request to the given Logrus logger instance
func Logger(logger *logrus.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.WithFields(logrus.Fields{
				"remoteaddr": r.RemoteAddr,
				"protocol":   r.Proto,
				"method":     r.Method,
				"url":        r.URL.String(),
			}).Info("request received")

			next.ServeHTTP(w, r)
		})
	}
}

func main() {
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

	// Use the Logrus logger as middleware
	r.Use(Logger(c.Logger))

	// CORS handler
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   c.Config.Cors.AllowedOrigins,
		AllowedMethods:   c.Config.Cors.AllowedMethods,
		AllowedHeaders:   c.Config.Cors.AllowedHeaders,
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: c.Config.Cors.AllowCredentials,
		MaxAge:           int(time.Duration.Seconds(12 * time.Hour)),
	}).Handler)

	// Serve the embedded contents of the "static" directory on the "/static" URL
	r.PathPrefix("/static/").Handler(http.StripPrefix("/", http.FileServer(http.FS(staticFS))))

	// Create a subrouter for version 1 of our API
	api := r.PathPrefix("/api").Subrouter()
	routes.AddRoutesForV1(api)

	// Serve the index.html file from the embedded static directory on the root URL ("/")
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

	// Start the server
	address := fmt.Sprintf("%s:%s", c.Config.Global.ServerAddress, c.Config.Global.ServerPort)
	c.Logger.Infof("starting server on %s", address)
	http.ListenAndServe(address, r)
}

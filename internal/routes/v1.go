/*
  HomeDash - A simple, automated dashboard for home labs.
  Copyright (C) 2023  Martijn van der Kleijn

  This file is part of HomeDash.

  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
	file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	c "github.com/mvdkleijn/homedash/internal/config"
	m "github.com/mvdkleijn/homedash/internal/models"

	"github.com/gorilla/mux"
)

var DataStore = m.DataStore{
	LastUpdated: map[string]time.Time{},
	Containers:  make(map[string][]m.ContainerInfo),
}

func AddRoutesForV1(rg *mux.Router) error {
	api := rg.PathPrefix("/v1").Subrouter()

	// Create a route to handle the POST request to /v1/applications
	api.HandleFunc("/applications", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var containerUpdate m.ContainerUpdate
		err = json.Unmarshal(body, &containerUpdate)
		if err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		if containerUpdate.Uuid == "" {
			error := "missing uuid in payload"
			c.Logger.Warn(error)
			http.Error(w, error, http.StatusUnprocessableEntity)
			return
		}

		if containerUpdate.Containers == nil {
			containerUpdate.Containers = []m.ContainerInfo{}
		}

		for i := range containerUpdate.Containers {
			value, exists := c.Index[containerUpdate.Containers[i].Icon]

			if exists {
				containerUpdate.Containers[i].IconFile = "/icons/" + value
			} else {
				containerUpdate.Containers[i].IconFile = "/static/default-icon.svg"
			}
		}

		DataStore.AddEntries(containerUpdate.Uuid, containerUpdate.Containers)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(containerUpdate)
	}).Methods("POST")

	// Create a route to handle the GET request to /v1/applications
	api.HandleFunc("/applications", func(w http.ResponseWriter, r *http.Request) {
		containerList := DataStore.GetContainerList()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(containerList)
	}).Methods("GET")

	// Create a route to handle the GET request to /v1/sidecars
	api.HandleFunc("/sidecars", func(w http.ResponseWriter, r *http.Request) {
		sidecarList := DataStore.GetSidecarList()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sidecarList)
	}).Methods("GET")

	api.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("OK")
	}).Methods("GET")

	api.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}).Methods("HEAD")

	return nil
}

func ServeIcon(w http.ResponseWriter, r *http.Request) {
	// Get the filename parameter from the URL
	vars := mux.Vars(r)
	filename := vars["filename"]

	// Construct the path to the file
	filePath := filepath.Join(c.Config.Icons.CacheDir, "icons", filename)

	fmt.Printf("Serving icon %s", filePath)

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	// Get the file's information
	fileInfo, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Get the file's content type
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if ext == "svg" {
		ext = "svg+xml"
	}
	contentType := "image/" + ext

	// Set the appropriate content type header
	w.Header().Set("Content-Type", contentType)

	// Serve the file
	http.ServeContent(w, r, filename, fileInfo.ModTime(), file)
}

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
	"io"
	"net/http"
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

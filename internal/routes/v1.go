/*
	HomeDash - A simple, automated dashboard for home labs.
	Copyright (C) 2023-2024  Martijn van der Kleijn

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
	"os"
	"path/filepath"
	"strings"
	"time"

	c "github.com/mvdkleijn/homedash/internal/config"
	m "github.com/mvdkleijn/homedash/internal/models"
	s "github.com/mvdkleijn/homedash/internal/services"
)

var DataStore = s.DataStore{
	LastUpdated: map[string]time.Time{},
	Containers:  make(map[string][]m.ContainerInfo),
}

type V1 struct{}

func (v *V1) AddRoutes(mux *http.ServeMux) error {
	mux.HandleFunc("POST /api/v1/applications", v.PostApplications)
	mux.HandleFunc("GET /api/v1/applications", v.GetApplications)
	mux.HandleFunc("GET /api/v1/sidecars", v.GetSidecars)
	mux.HandleFunc("GET /api/v1/status", v.GetStatus)
	mux.HandleFunc("HEAD /api/v1/status", v.HeadStatus)

	return nil
}

func (v *V1) GetSidecars(w http.ResponseWriter, r *http.Request) {
	sidecars := DataStore.GetSidecarList()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sidecars); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *V1) GetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode("OK")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *V1) HeadStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (v *V1) GetApplications(w http.ResponseWriter, r *http.Request) {
	containerList := DataStore.GetContainerList()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(containerList)
}

func (v *V1) PostApplications(w http.ResponseWriter, r *http.Request) {
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
		errorMsg := "missing uuid in payload"
		c.Logger.Warn().Str("uuid", errorMsg)
		http.Error(w, errorMsg, http.StatusUnprocessableEntity)
		return
	}

	if containerUpdate.Containers == nil {
		containerUpdate.Containers = []m.ContainerInfo{}
	}

	for i := range containerUpdate.Containers {
		containerUpdate.Containers[i].IconFile = c.GetIconPath(containerUpdate.Containers[i].Icon)
	}

	DataStore.AddEntries(containerUpdate.Uuid, containerUpdate.Containers)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(containerUpdate)
}

func ServeIcon(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	filename := strings.TrimPrefix(path, "/icons/")

	// If the path was just "/icons/", filename will be empty
	if filename == "" || filename == "/" {
		http.NotFound(w, r)
		return
	}

	// Validate filename to ensure it does not contain any path separators or parent directory references
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..") {
		http.Error(w, "Invalid file name", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(c.Config.Icons.CacheDir, "icons", filename)

	c.Logger.Debug().Str("filename", filePath).Msg("serving icon")

	if file, err := os.Open(filePath); err == nil {
		defer file.Close()

		if fileInfo, err := file.Stat(); err == nil {
			ext := strings.TrimPrefix(filepath.Ext(filename), ".")
			if ext == "svg" {
				ext = "svg+xml"
			}
			contentType := "image/" + ext

			w.Header().Set("Content-Type", contentType)

			http.ServeContent(w, r, filename, fileInfo.ModTime(), file)
			return
		}
	}
	http.NotFound(w, r)
}

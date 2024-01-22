/*
	HomeDash - A simple, automated dashboard for home labs.
	Copyright (C) 2023-2024  Martijn van der Kleijn

	This file is part of HomeDash.

	This Source Code Form is subject to the terms of the Mozilla Public
	License, v. 2.0. If a copy of the MPL was not distributed with this
	file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package services

import (
	"sync"
	"time"

	// c "github.com/mvdkleijn/homedash/internal/config"
	// r "github.com/mvdkleijn/homedash/internal/repositories"
	m "github.com/mvdkleijn/homedash/internal/models"
	r "github.com/mvdkleijn/homedash/internal/repositories"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
)

type ContainerUpdate struct {
	Uuid       string            `json:"uuid"`
	Containers []m.ContainerInfo `json:"containers"`
}

type DataStore struct {
	mu          sync.Mutex
	LastUpdated map[string]time.Time
	Containers  map[string][]m.ContainerInfo
}

func (ds *DataStore) CleanupOutdatedEntries(maxAgeInMinutes int) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	now := time.Now()
	uuids := maps.Keys(ds.Containers)
	log.Debugf("cleaning up outdated entries")
	for _, uuid := range uuids {
		// Remove data if no updates in X minutes or more
		if now.Sub(ds.LastUpdated[uuid]) >= time.Duration(maxAgeInMinutes)*time.Minute {
			log.Debugf("removing entries for sidecar %s", uuid)
			delete(ds.Containers, uuid)
			delete(ds.LastUpdated, uuid)
		}
	}
}

func (ds *DataStore) GetLastUpdated(uuid string) (time.Time, bool) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	time, exists := ds.LastUpdated[uuid]

	return time, exists
}

func (ds *DataStore) GetContainerList() []m.ContainerInfo {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	containerInfoList := []m.ContainerInfo{}
	// staticAppList := c.Config.Static.Apps

	for _, containerList := range ds.Containers {
		containerInfoList = append(containerInfoList, containerList...)
	}

	for _, containerList := range r.GetAppList() {
		containerInfoList = append(containerInfoList, containerList...)
	}

	// for _, app := range staticAppList {
	// 	log.Debugf("adding static app %s with icon %s", app.Name, app.Icon)
	// 	containerInfoList = append(containerInfoList, ContainerInfo{
	// 		Name:     app.Name,
	// 		Icon:     app.Icon,
	// 		IconFile: app.IconFile,
	// 		Comment:  app.Comment,
	// 	})
	// }

	return containerInfoList
}

func (ds *DataStore) GetSidecarList() []string {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	return maps.Keys(ds.Containers)
}

func (ds *DataStore) AddEntries(uuid string, containers []m.ContainerInfo) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.LastUpdated[uuid] = time.Now()
	ds.Containers[uuid] = containers
}

func (ds *DataStore) ReplaceEntries(uuid string, containers []m.ContainerInfo) {
	// Maybe check if entry already exists in future but not sure why we'd want to right now.
	ds.AddEntries(uuid, containers)
}

func (ds *DataStore) DeleteAllEntries(uuid string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	delete(ds.LastUpdated, uuid)
	delete(ds.Containers, uuid)
}

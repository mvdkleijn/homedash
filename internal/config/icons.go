/*
	HomeDash - A simple, automated dashboard for home labs.
	Copyright (C) 2023-2024  Martijn van der Kleijn

	This file is part of HomeDash.

	This Source Code Form is subject to the terms of the Mozilla Public
	License, v. 2.0. If a copy of the MPL was not distributed with this
	file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package config

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

type App struct {
	Icon        string `json:"icon"`
	IconName    string `json:"icon_name"`
	Name        string `json:"name"`
	Website     string `json:"website"`
	License     string `json:"license"`
	Description string `json:"description"`
}

type AppList struct {
	AppCount int   `json:"appcount"`
	Apps     []App `json:"apps"`
}

const (
	zipURL      = "https://github.com/linuxserver/Heimdall-Apps/archive/refs/heads/gh-pages.zip"
	zipFileName = "gh-pages.zip"
)

func downloadFile(url string, filepath string) error {
	Logger.Debug().Str("url", url).Msg("attempting to download update from url")

	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	return err
}

func unzipFile(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if strings.Contains(file.Name, "..") {
			log.Warn().Str("file", file.Name).Msg("skipping file with invalid path")
			continue
		}
		path := filepath.Join(dest, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, os.ModePerm)
			continue
		}

		fileDir := filepath.Dir(path)
		err = os.MkdirAll(fileDir, os.ModePerm)
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		_, err = io.Copy(outFile, fileReader)
		if err != nil {
			return err
		}
	}

	return nil
}

func UpdateIcons(refresh bool) {
	_, err := os.Stat(filepath.Join(Config.Icons.CacheDir, "applications_index.json"))
	if err == nil && !refresh {
		Logger.Info().Msg("already have icons and not asked to refresh")
		createIndexFromCache()
		return
	}

	os.RemoveAll(Config.Icons.CacheDir)
	os.MkdirAll(Config.Icons.TmpDir, os.ModePerm)
	os.MkdirAll(Config.Icons.CacheDir, os.ModePerm)

	err = downloadFile(zipURL, filepath.Join(Config.Icons.TmpDir, zipFileName))
	if err != nil {
		log.Err(err).Msg("failed to download the zip file")
		return
	}

	err = unzipFile(filepath.Join(Config.Icons.TmpDir, zipFileName), Config.Icons.TmpDir)
	if err != nil {
		log.Err(err).Msg("failed to unzip the file")
		return
	}

	err = os.Rename(filepath.Join(Config.Icons.TmpDir, "Heimdall-Apps-gh-pages", "icons"), filepath.Join(Config.Icons.CacheDir, "icons"))
	if err != nil {
		log.Err(err).Msg("failed to move the icons directory")
		return
	}

	err = os.Rename(filepath.Join(Config.Icons.TmpDir, "Heimdall-Apps-gh-pages", "list.json"), filepath.Join(Config.Icons.CacheDir, "list.json"))
	if err != nil {
		log.Err(err).Msg("failed to move the icons directory")
		return
	}

	createIndex()

	os.RemoveAll(Config.Icons.TmpDir)

	log.Info().Msg("Zip file downloaded, unzipped, and icons directory updated successfully.")
}

func createIndex() {
	fileData, err := os.ReadFile(filepath.Join(Config.Icons.CacheDir, "list.json"))
	if err != nil {
		log.Err(err).Msg("failed to read JSON file")
		return
	}

	var appList AppList
	err = json.Unmarshal(fileData, &appList)
	if err != nil {
		log.Err(err).Msg("failed to parse JSON file")
		return
	}

	for i := range appList.Apps {
		app := &appList.Apps[i]
		app.IconName = strings.Split(app.Icon, ".")[0]

		Index[app.IconName] = app.Icon
	}

	updatedData, err := json.MarshalIndent(appList, "", "  ")
	if err != nil {
		log.Err(err).Msg("failed to convert data to JSON")
		return
	}

	err = os.WriteFile(filepath.Join(Config.Icons.CacheDir, "applications_index.json"), updatedData, 0644)
	if err != nil {
		log.Err(err).Msg("failed to write updated JSON to file")
		return
	}

	os.RemoveAll(filepath.Join(Config.Icons.CacheDir, "list.json"))

	log.Info().Msg("JSON file successfully updated and exported.")
}

func createIndexFromCache() {
	fileData, err := os.ReadFile(filepath.Join(Config.Icons.CacheDir, "applications_index.json"))
	if err != nil {
		log.Err(err).Msg("failed to read the JSON file")
		return
	}

	var appList AppList
	err = json.Unmarshal(fileData, &appList)
	if err != nil {
		log.Err(err).Msg("failed to parse the JSON file")
		return
	}

	for i := range appList.Apps {
		app := &appList.Apps[i]

		Index[app.IconName] = app.Icon
	}

	log.Info().Msg("successfully read icon index from file.")
}

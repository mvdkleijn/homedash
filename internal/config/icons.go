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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type App struct {
	// AppID           string `json:"appid"`
	// Enhanced        bool   `json:"enhanced"`
	// TitleBackground string `json:"title_background"`
	// SHA             string `json:"sha"`
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
	// destinationDir = c.IconConfiguration.TmpDir //"data/tmp"
	// iconsDir       = "data/cache/icons"
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
	// Check if the directory exists
	_, err := os.Stat(filepath.Join(Config.Icons.CacheDir, "applications_index.json"))
	if err == nil && !refresh {
		Logger.Info().Msg("already have icons and not asked to refresh")
		createIndexFromCache()
		return
	}

	// Delete old cache directory
	os.RemoveAll(Config.Icons.CacheDir)

	// Create destination directories if they don't exist
	os.MkdirAll(Config.Icons.TmpDir, os.ModePerm)
	os.MkdirAll(Config.Icons.CacheDir, os.ModePerm)

	// Download the zip file
	err = downloadFile(zipURL, filepath.Join(Config.Icons.TmpDir, zipFileName))
	if err != nil {
		fmt.Printf("Failed to download the zip file: %v\n", err)
		return
	}

	// Unzip the file
	err = unzipFile(filepath.Join(Config.Icons.TmpDir, zipFileName), Config.Icons.TmpDir)
	if err != nil {
		fmt.Printf("Failed to unzip the file: %v\n", err)
		return
	}

	// Move the "icons" directory
	err = os.Rename(filepath.Join(Config.Icons.TmpDir, "Heimdall-Apps-gh-pages", "icons"), filepath.Join(Config.Icons.CacheDir, "icons"))
	if err != nil {
		fmt.Printf("Failed to move the icons directory: %v\n", err)
		return
	}

	// Move the "list.json" file
	err = os.Rename(filepath.Join(Config.Icons.TmpDir, "Heimdall-Apps-gh-pages", "list.json"), filepath.Join(Config.Icons.CacheDir, "list.json"))
	if err != nil {
		fmt.Printf("Failed to move the icons directory: %v\n", err)
		return
	}

	// Create an applications index based on the list.json
	createIndex()

	// Delete old tmp directory
	os.RemoveAll(Config.Icons.TmpDir)

	fmt.Println("Zip file downloaded, unzipped, and icons directory updated successfully.")
}

func createIndex() {
	// Read the JSON file
	fileData, err := os.ReadFile(filepath.Join(Config.Icons.CacheDir, "list.json"))
	if err != nil {
		fmt.Printf("Failed to read the JSON file: %v\n", err)
		return
	}

	// Parse the JSON data into the AppList struct
	var appList AppList
	err = json.Unmarshal(fileData, &appList)
	if err != nil {
		fmt.Printf("Failed to parse the JSON file: %v\n", err)
		return
	}

	// Modify each entry in the apps array
	for i := range appList.Apps {
		app := &appList.Apps[i]
		app.IconName = strings.Split(app.Icon, ".")[0]

		// Build index in memory
		Index[app.IconName] = app.Icon
	}

	// Convert the modified data back to JSON
	updatedData, err := json.MarshalIndent(appList, "", "  ")
	if err != nil {
		fmt.Printf("Failed to convert data to JSON: %v\n", err)
		return
	}

	// Write the updated JSON to a file
	err = os.WriteFile(filepath.Join(Config.Icons.CacheDir, "applications_index.json"), updatedData, 0644)
	if err != nil {
		fmt.Printf("Failed to write updated JSON to file: %v\n", err)
		return
	}

	// Delete old list.json
	os.RemoveAll(filepath.Join(Config.Icons.CacheDir, "list.json"))

	fmt.Println("JSON file successfully updated and exported.")
}

func createIndexFromCache() {
	// Read the JSON file
	fileData, err := os.ReadFile(filepath.Join(Config.Icons.CacheDir, "applications_index.json"))
	if err != nil {
		fmt.Printf("Failed to read the JSON file: %v\n", err)
		return
	}

	// Parse the JSON data into the AppList struct
	var appList AppList
	err = json.Unmarshal(fileData, &appList)
	if err != nil {
		fmt.Printf("Failed to parse the JSON file: %v\n", err)
		return
	}

	// Modify each entry in the apps array
	for i := range appList.Apps {
		app := &appList.Apps[i]

		// Build index in memory
		Index[app.IconName] = app.Icon
	}

	fmt.Println("Successfully read icon index from file.")
}

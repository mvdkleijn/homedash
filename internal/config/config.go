/*
	HomeDash - A simple, automated dashboard for home labs.
	Copyright (C) 2023-2026  Martijn van der Kleijn

	This file is part of HomeDash.

	This Source Code Form is subject to the terms of the Mozilla Public
	License, v. 2.0. If a copy of the MPL was not distributed with this
	file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package config

import (
	"fmt"
	"os"
	"strings"

	m "github.com/mvdkleijn/homedash/internal/models"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
)

type IconIndex map[string]string

type Configuration struct {
	Debug               bool `koanf:"debug"`
	MaxAgeBeforeCleanup int  `koanf:"maxage"`
	CleanCheckInterval  int  `koanf:"cleaninterval"`

	Cors   CorsConfiguration   `koanf:"cors"`
	Icons  IconConfiguration   `koanf:"icons"`
	Static StaticConfiguration `koanf:"static"`
	Server ServerConfiguration `koanf:"server"`
}

type ServerConfiguration struct {
	Address string `koanf:"address"`
	Port    string `koanf:"port"`
}

type IconConfiguration struct {
	CacheDir string `koanf:"cachedir"`
	TmpDir   string `koanf:"tmpdir"`
}

type StaticConfiguration struct {
	Apps []m.ContainerInfo `koanf:"apps"`
}

type CorsConfiguration struct {
	AllowedOrigins   []string `koanf:"allowedorigins"`
	AllowCredentials bool     `koanf:"allowcredentials"`
	AllowedHeaders   []string `koanf:"allowedheaders"`
	AllowedMethods   []string `koanf:"allowedmethods"`
	Debug            bool     `koanf:"debug"`
}

var (
	Config Configuration
	Logger *zerolog.Logger
	Index  IconIndex = IconIndex{}
	k                = koanf.New(".")
)

func initConfig() {
	allowedMethods := []string{"GET", "POST", "HEAD"}

	// Set defaults
	k.Set("debug", false)
	k.Set("maxAge", 20)
	k.Set("checkInterval", 1)
	k.Set("server.address", "")
	k.Set("server.port", "8080")
	k.Set("cors.allowedOrigins", "*")
	k.Set("cors.allowCredentials", false)
	k.Set("cors.allowedHeaders", "Content-Type")
	k.Set("cors.allowedMethods", allowedMethods)
	k.Set("cors.debug", false)
	k.Set("apps", []m.ContainerInfo{})

	if hasContainerDataDir() {
		Logger.Debug().Msg("detected default /homedash directory, using container-optimized paths")
		k.Set("icons.tmpDir", "/homedash/tmp")
		k.Set("icons.cacheDir", "/homedash/cache")
	} else {
		k.Set("icons.tmpDir", "./data/tmp")
		k.Set("icons.cacheDir", "./data/cache")
	}

	// Load Config File
	configPaths := []string{"."}
	if hasContainerDataDir() {
		configPaths = append(configPaths, "/homedash")
	}

	for _, path := range configPaths {
		if err := k.Load(file.Provider(path+"/config.yml"), yaml.Parser()); err != nil {
			Logger.Info().Err(err).Msg("tried to load configuration file but found none or error occurred")
		} else {
			Logger.Info().Str("configfile", path+"/config.yml").Msg("loaded configuration file")
			break
		}
	}

	// Load Environment Variables
	// Replaces DOT with UNDERSCORE (e.g., HOMEDASH_SERVER_PORT)
	k.Load(env.Provider("HOMEDASH_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "HOMEDASH_")), "_", ".", -1)
	}), nil)

	// Unmarshal directly into the struct
	if err := k.Unmarshal("", &Config); err != nil {
		Logger.Fatal().Err(err).Msg("failed to unmarshal configuration")
	}

	// Post-processing:	handle logic that depends on runtime state (like icon paths).
	UpdateIconPaths()

	Logger.Debug().Any("config", Config).Msg("debug config system")
}

func Setup() {
	// TODO: check an ENV var to decide whether to use human readable or default JSON.
	// Create the ConsoleWriter for human-readable output
	zerolog.TimeFieldFormat = "15:04:05.000"

	output := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05.000",
		NoColor:    true,
	}

	// Initialize the GLOBAL Logger with the writer
	log := zerolog.New(output).With().Timestamp().Logger()
	Logger = &log

	// Set the global logger level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	Logger.Info().Msg("initializing system")

	// Run config logic
	initConfig()

	if Config.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		Logger.Debug().Msg("enabled DEBUG logging level")
	}

	UpdateIcons(false)
	UpdateIconPaths()

	Logger.Info().Msg("initialization completed")
	Logger.Debug().Interface("config", Config).Msg("dumping active configuration")
}

func hasContainerDataDir() bool {
	if _, err := os.Stat("/homedash"); err != nil {
		return false
	}
	return true
}

func UpdateIconPaths() {
	for i := range Config.Static.Apps {
		Config.Static.Apps[i].IconFile = GetIconPath(Config.Static.Apps[i].Icon)
	}
}

func GetIconPath(icon string) string {
	Logger.Debug().Str("icon", icon).Msg("getting path")
	value, exists := Index[icon]

	if !exists {
		Logger.Debug().Str("icon", icon).Msg("not found in index")
		return "/static/default-icon.svg"
	}

	return fmt.Sprintf("/icons/%s", value)
}

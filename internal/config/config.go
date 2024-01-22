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
	"fmt"
	"os"
	"strings"

	m "github.com/mvdkleijn/homedash/internal/models"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type IconIndex map[string]string

type Configuration struct {
	Global GlobalConfiguration
	Cors   CorsConfiguration
	Icons  IconConfiguration
	Static StaticConfiguration
}

// GlobalConfig holds global configuration items
type GlobalConfiguration struct {
	Debug               bool
	ServerAddress       string
	ServerPort          string
	MaxAgeBeforeCleanup int
	CleanCheckInterval  int
}

type IconConfiguration struct {
	CacheDir string
	TmpDir   string
}

type StaticConfiguration struct {
	Apps []m.ContainerInfo
}

type CorsConfiguration struct {
	AllowedOrigins   []string
	AllowCredentials bool
	AllowedHeaders   []string
	AllowedMethods   []string
	Debug            bool
}

var (
	Config Configuration
	Logger *zerolog.Logger
	Index  IconIndex = IconIndex{}
)

func initViper() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	allowedMethods := []string{"GET", "POST", "HEAD"}
	viper.SetDefault("server.address", "")
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("debug", false)
	viper.SetDefault("maxAge", "20")
	if isRunningInContainer() {
		Logger.Debug().Msg("detected that we're runnning in a container, using /homedash as default data directory")
		viper.SetDefault("icons.tmpDir", "/homedash/tmp")
		viper.SetDefault("icons.cacheDir", "/homedash/cache")
	} else {
		viper.SetDefault("icons.tmpDir", "./data/tmp")
		viper.SetDefault("icons.cacheDir", "./data/cache")
	}
	viper.SetDefault("checkInterval", "1")
	viper.SetDefault("cors.allowedOrigins", "*")
	viper.SetDefault("cors.allowCredentials", false)
	viper.SetDefault("cors.allowedHeaders", "Content-Type")
	viper.SetDefault("cors.allowedMethods", allowedMethods)
	viper.SetDefault("cors.debug", false)
	viper.SetDefault("apps", []m.ContainerInfo{})

	viper.SetEnvPrefix("homedash")

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
	if isRunningInContainer() {
		viper.AddConfigPath("/homedash")
	}

	err := viper.ReadInConfig()
	if err != nil {
		Logger.Info().Msg("tried to load configuration file but found none")
	} else {
		Logger.Info().Str("configfile", viper.ConfigFileUsed()).Msg("loaded configuration file")
	}

	viper.AutomaticEnv()

	Config.Global.ServerAddress = viper.GetString("server.address")
	Config.Global.ServerPort = viper.GetString("server.port")
	Config.Global.Debug = viper.GetBool("debug")
	Config.Global.MaxAgeBeforeCleanup = viper.GetInt("maxAge")
	Config.Global.CleanCheckInterval = viper.GetInt("checkInterval")
	Config.Icons.TmpDir = viper.GetString("icons.tmpDir")
	Config.Icons.CacheDir = viper.GetString("icons.cacheDir")
	Config.Cors.AllowedOrigins = viper.GetStringSlice("cors.allowedOrigins")
	Config.Cors.AllowCredentials = viper.GetBool("cors.allowCredentials")
	Config.Cors.AllowedHeaders = viper.GetStringSlice("cors.allowedHeaders")
	Config.Cors.AllowedMethods = viper.GetStringSlice("cors.allowedMethods")
	Config.Cors.Debug = viper.GetBool("cors.debug")

	var appInfos []m.ContainerInfo
	if apps := viper.Get("static.apps"); apps != nil {
		if appSlice, ok := apps.([]interface{}); ok {
			appInfos = make([]m.ContainerInfo, len(appSlice))
			for i, v := range appSlice {
				if appMap, ok := v.(map[string]interface{}); ok {
					appInfo := m.ContainerInfo{
						Name: appMap["name"].(string),
					}
					if url, ok := appMap["url"].(string); ok {
						appInfo.Url = url
					}
					if icon, ok := appMap["icon"].(string); ok {
						appInfo.Icon = icon
					}
					if comment, ok := appMap["comment"].(string); ok {
						appInfo.Comment = comment
					}
					appInfo.IconFile = GetIconPath(appInfo.Icon)
					appInfos[i] = appInfo
				} else {
					continue
				}
			}
			Logger.Info().Str("appInfos", fmt.Sprintf("%v", appInfos)).Msg("Loaded static configuration")
		} else {
			Logger.Info().Str("apps", fmt.Sprintf("%v", apps)).Msg("Skipping invalid static configuration")
		}
	}

	Config.Static.Apps = appInfos
}

func init() {
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	Logger = &log

	log.Info().Msg("initializing system")
	initViper()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if Config.Global.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("enabled DEBUG logging level")
	}

	UpdateIcons(false)
	UpdateIconPaths()

	log.Info().Msg("initialization completed")
	Logger.Debug().Interface("config", Config).Msg("dumping active configuration")
}

func isRunningInContainer() bool {
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
	log.Debug().Str("icon", icon).Msg("getting path")
	value, exists := Index[icon]

	if !exists {
		log.Debug().Str("icon", icon).Msg("not found in index")
		return "/static/default-icon.svg"
	}

	return fmt.Sprintf("/icons/%s", value)
}

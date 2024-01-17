/*
	HomeDash - A simple, automated dashboard for home labs.
	Copyright (C) 2023  Martijn van der Kleijn

	This file is part of HomeDash.

	This Source Code Form is subject to the terms of the Mozilla Public
	License, v. 2.0. If a copy of the MPL was not distributed with this
	file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package config

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
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
	Apps []AppInfo
}

type AppInfo struct {
	Name     string `json:"name"`
	Url      string `json:"url"`
	Icon     string `json:"icon"`
	IconFile string `json:"iconFile"`
	Comment  string `json:"comment"`
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
	Logger *log.Logger
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
		Logger.Debugln("detected that we're runnning in a container, using /homedash as default data directory")
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

	// Set default values for the static configuration
	viper.SetDefault("apps", []AppInfo{})

	viper.SetEnvPrefix("homedash")

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
	if isRunningInContainer() {
		viper.AddConfigPath("/homedash")
	}

	err := viper.ReadInConfig()
	if err != nil {
		Logger.Infof("tried to load configuration file but found none")
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
	if apps := viper.Get("static.apps"); apps != nil {
		if appSlice, ok := apps.([]AppInfo); ok {
			appInfos := make([]AppInfo, len(appSlice))
			copy(appInfos, appSlice)
			Config.Static.Apps = appInfos
		}
	}

	Logger.Debugf("loaded configuration: %v", viper.AllSettings())
}

func init() {
	Logger = log.New()
	Logger.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	Logger.Info("initializing system")
	initViper()

	if Config.Global.Debug {
		Logger.SetLevel(log.DebugLevel)
		Logger.Debug("enabled DEBUG logging level")
	}

	UpdateIcons(false)

	Logger.Info("initialization completed")
	Logger.Debugf("dumping active configuration: %v", Config)
}

func isRunningInContainer() bool {
	if _, err := os.Stat("/homedash"); err != nil {
		return false
	}
	return true
}

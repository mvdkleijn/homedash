package repositories

import (
	c "github.com/mvdkleijn/homedash/internal/config"
	m "github.com/mvdkleijn/homedash/internal/models"
)

func GetAppList() map[string][]m.ContainerInfo {
	appList := make(map[string][]m.ContainerInfo)

	appList["static"] = c.Config.Static.Apps

	return appList
}

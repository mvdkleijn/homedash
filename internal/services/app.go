package services

import (
	m "github.com/mvdkleijn/homedash/internal/models"
	r "github.com/mvdkleijn/homedash/internal/repositories"
)

func GetAppList() map[string][]m.ContainerInfo {
	return r.GetAppList()
}

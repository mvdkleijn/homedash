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
	m "github.com/mvdkleijn/homedash/internal/models"
	r "github.com/mvdkleijn/homedash/internal/repositories"
)

func GetAppList() map[string][]m.ContainerInfo {
	return r.GetAppList()
}

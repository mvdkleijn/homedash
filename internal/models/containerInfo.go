/*
	HomeDash - A simple, automated dashboard for home labs.
    Copyright (C) 2023  Martijn van der Kleijn

	This file is part of HomeDash.

    This Source Code Form is subject to the terms of the Mozilla Public
  	License, v. 2.0. If a copy of the MPL was not distributed with this
  	file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package models

type ContainerInfo struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	Icon    string `json:"icon"`
	Comment string `json:"comment"`
}

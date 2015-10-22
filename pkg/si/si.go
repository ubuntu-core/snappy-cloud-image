// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2015 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

// Package si handles the communication with the system-image server
package si

import (
	"encoding/json"
	"fmt"

	"github.com/fgimenez/snappy-cloud-image/pkg/web"
)

const (
	baseURL      = "http://system-image.ubuntu.com/ubuntu-core"
	dataFileName = "index.json"
)

// Client is the default implementation of Driver
type Client struct {
	httpClient web.Getter
}

type response struct {
	Global interface{}
	Images images
}

type images []image

type image struct {
	Description   string
	Type          string
	Version       int
	VersionDetail string `json:"version_detail"`
	Files         files
}

type files []file

type file struct {
	CheckSum  string
	Order     int
	Path      string
	Signature string
	Size      int
}

// GetLatestVersion returns the highest version from the system image server for the given
// release, channel and arch, or an error in case something goes wrong
func (c *Client) GetLatestVersion(release, channel, arch string) (ver int, err error) {
	url := generateURL(release, channel, arch)
	content, err := c.httpClient.Get(url)
	if err != nil {
		return
	}
	var res response
	err = json.Unmarshal(content, &res)
	if err != nil {
		return
	}

	index := len(res.Images) - 1
	if index >= 0 {
		for {
			if res.Images[index].Type == "full" {
				break
			}
			index--
		}
		ver = res.Images[index].Version
	}
	return
}

func generateURL(release, channel, arch string) string {
	if arch == "arm" {
		arch += "hf"
	}
	return fmt.Sprintf("%s/%s/%s/generic_%s/%s",
		baseURL, release, channel, arch, dataFileName)
}

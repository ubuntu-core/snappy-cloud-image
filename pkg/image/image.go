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

// Package image knows how to crete the requested images using UDF.
// It also defines the required interfaces standarize the query and creation
// of images
package image

import "github.com/fgimenez/snappy-cloud-image/pkg/cli"

// Pollster holds the methods for querying an image backend
type Pollster interface {
	GetLatestVersion(release, channel, arch string) (ver int, err error)
}

// PollsterCreator is a Pollster that can also create new images
type PollsterCreator interface {
	Pollster
	Create(filePath, release, channel, arch string, version int) (err error)
}

// Driver defines the methods required for creating images
type Driver interface {
	Create(release, channel, arch string) (path string, err error)
}

// UDF is a concrete implementation of Driver
type UDF struct {
	cli cli.Commander
}

// Create makes the required call to UDF to
func (u *UDF) Create(release, channel, arch string) (path string, err error) {
	tmpFileName, _ := u.cli.ExecCommand("mktemp")

	var archFlag string
	if arch == "arm" {
		archFlag = "--oem beagleblack"
	}
	_, err = u.cli.ExecCommand("sudo", "ubuntu-device-flash", "core", release,
		"--channel", channel, "--developer-mode", archFlag, "-o", tmpFileName)

	return tmpFileName, err
}

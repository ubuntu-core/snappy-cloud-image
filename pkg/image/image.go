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

import (
	"path/filepath"
	"strconv"

	"github.com/ubuntu-core/snappy-cloud-image/pkg/cli"
)

const outputFileName = "udf.img"

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
	Create(release, channel, arch string, ver int) (path string, err error)
}

// UDF is a concrete implementation of Driver
type UDF struct {
	cli cli.Commander
}

// NewUDF is the UDF constructor
func NewUDF(cli cli.Commander) *UDF {
	return &UDF{cli: cli}
}

// Create makes the required call to UDF to
func (u *UDF) Create(release, channel, arch string, ver int) (path string, err error) {
	tmpDirName, _ := u.cli.ExecCommand("mktemp -d")
	tmpFileName := filepath.Join(tmpDirName, outputFileName)

	var archFlag string
	if arch == "arm" {
		archFlag = "--oem beagleblack"
	}
	_, err = u.cli.ExecCommand("sudo", "ubuntu-device-flash", "core", release,
		"--revision="+strconv.Itoa(ver), "--channel", channel, "--developer-mode",
		archFlag, "-o", tmpFileName)

	return tmpFileName, err
}

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
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/ubuntu-core/snappy-cloud-image/pkg/cli"
)

const outputFileName = "udf.img"

// Pollster holds the methods for querying an image backend
type Pollster interface {
	GetLatestVersion(release, channel, arch string) (ver int, err error)
}

// FullPollster is a Pollster that knows how to get a list of Versions too
type FullPollster interface {
	Pollster
	GetVersions(release, channel, arch string) (images []string, err error)
}

// PollsterWriter is a Pollster that can also create and delete images
type PollsterWriter interface {
	FullPollster
	Create(filePath, release, channel, arch string, version int) (err error)
	Delete(images ...string) (err error)
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
	tmpDirName, err := u.cli.ExecCommand("mktemp", "-d")
	if err != nil {
		return
	}
	tmpFileName := filepath.Join(strings.TrimSpace(tmpDirName), outputFileName)
	log.Debug("Target image filename: ", tmpFileName)

	var archFlag string
	if arch == "arm" {
		archFlag = "--oem beagleblack"
	}
	cmds := []string{"sudo", "ubuntu-device-flash", "--revision=" + strconv.Itoa(ver), "core", release,
		"--channel", channel, "--developer-mode",
		archFlag, "-o", tmpFileName}
	log.Debug("Executing command ", strings.Join(cmds, " "))
	_, err = u.cli.ExecCommand(cmds...)

	return tmpFileName, err
}

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

// Package image knows how to create the requested images using UDF.
// It also defines the required interfaces standarize the query and creation
// of images
package image

import (
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/ubuntu-core/snappy-cloud-image/pkg/cli"
	"github.com/ubuntu-core/snappy-cloud-image/pkg/flags"
)

const (
	rawOutputFileName = "udf.raw"
	outputFileName    = "udf.img"
)

// Pollster holds the methods for querying an image backend
type Pollster interface {
	GetLatestVersion(options *flags.Options) (ver int, err error)
}

// FullPollster is a Pollster that knows how to get a list of Versions too
type FullPollster interface {
	Pollster
	GetVersions(options *flags.Options) (images []string, err error)
}

// PollsterWriter is a Pollster that can also create and delete images
type PollsterWriter interface {
	FullPollster
	Create(filePath string, options *flags.Options, version int) (err error)
	Delete(images ...string) (err error)
	Purge(options *flags.Options) (err error)
}

// Driver defines the methods required for creating images
type Driver interface {
	Create(options *flags.Options, ver int) (path string, err error)
}

// UDFQcow2 is a concrete implementation of Driver
type UDFQcow2 struct {
	cli cli.Commander
}

// NewUDFQcow2 is the UDFQcow2 constructor
func NewUDFQcow2(cli cli.Commander) *UDFQcow2 {
	return &UDFQcow2{cli: cli}
}

// Create makes the required call to UDF to create the raw image, and then transforms
// it to the QCOW2 format
func (u *UDFQcow2) Create(options *flags.Options, ver int) (path string, err error) {
	tmpDirName, err := u.cli.ExecCommand("mktemp", "-d")
	if err != nil {
		return
	}
	rawTmpFileName := filepath.Join(strings.TrimSpace(tmpDirName), rawOutputFileName)
	log.Debug("Target image filename: ", rawTmpFileName)

	var archFlag string
	if options.Arch == "arm" {
		archFlag = "--oem beagleblack"
	}
	cmds := []string{"sudo", "ubuntu-device-flash",
		"--revision=" + strconv.Itoa(ver),
		"core", options.Release,
		"--channel", options.Channel,
	}
	if options.Release != "15.04" {
		cmds = append(cmds, []string{
			"--os", options.OS,
			"--kernel", options.Kernel,
			"--gadget", options.Gadget,
		}...)
	}
	cmds = append(cmds, []string{
		"--developer-mode",
		archFlag, "-o", rawTmpFileName}...)

	log.Debug("Executing command ", strings.Join(cmds, " "))
	output, err := u.cli.ExecCommand(cmds...)
	log.Debug(output)

	if err != nil {
		return
	}

	log.Debug("Converting to QCOW2 format")
	tmpFileName := filepath.Join(strings.TrimSpace(tmpDirName), outputFileName)
	cmds = []string{"/usr/bin/qemu-img",
		"convert", "-O", "qcow2",
		"-o", "compat=" + options.Qcow2compat,
		rawTmpFileName, tmpFileName}
	output, err = u.cli.ExecCommand(cmds...)
	log.Debug(output)

	return tmpFileName, err
}

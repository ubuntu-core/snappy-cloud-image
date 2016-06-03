// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2015, 2016 Canonical Ltd
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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/snapcore/snapd/progress"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/store"

	"github.com/ubuntu-core/snappy-cloud-image/pkg/cli"
	"github.com/ubuntu-core/snappy-cloud-image/pkg/flags"
)

const (
	rawOutputFileName  = "udf.raw"
	outputFileName     = "udf.img"
	errRepoDetailFmt   = "Could not get details of snap with name %s, developer %s and channel %s"
	errRepoDownloadFmt = "Could not download snap with name %s, developer %s and channel %s"
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

type storeClient interface {
	Download(*snap.Info, progress.Meter, store.Authenticator) (path string, err error)
	Snap(name, channel string, sa store.Authenticator) (r *snap.Info, err error)
}

// ErrRepoDetail is the error returned when the repo fails to retrive details of a specific snap
type ErrRepoDetail struct {
	name, developer, channel string
}

func (e *ErrRepoDetail) Error() string {
	return fmt.Sprintf(errRepoDetailFmt, e.name, e.developer, e.channel)
}

// ErrRepoDownload is the error returned when a snap could not be retrieved
type ErrRepoDownload struct {
	name, developer, channel string
}

func (e *ErrRepoDownload) Error() string {
	return fmt.Sprintf(errRepoDownloadFmt, e.name, e.developer, e.channel)
}

// UDFQcow2 is a concrete implementation of Driver
type UDFQcow2 struct {
	cli cli.Commander
	sc  storeClient
}

// NewUDFQcow2 is the UDFQcow2 constructor
func NewUDFQcow2(cli cli.Commander, sc storeClient) *UDFQcow2 {
	return &UDFQcow2{cli: cli, sc: sc}
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
	cmds := []string{"sudo", "ubuntu-device-flash"}

	if options.Release == "15.04" {
		cmds = append(cmds, "--revision="+strconv.Itoa(ver))
	}
	cmds = append(cmds, []string{
		"core", options.Release,
	}...)

	snapFlags, err := u.getSnapFlags(options)
	if err != nil {
		return
	}
	cmds = append(cmds,
		snapFlags...,
	)
	defer func() {
		for _, item := range snapFlags {
			os.RemoveAll(item)
		}
	}()

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

func (u *UDFQcow2) getSnapFile(name, channel string) (path string, err error) {
	remoteSnap, err := u.sc.Snap(name, channel, nil)
	if err != nil {
		return "", &ErrRepoDetail{name, "", channel}
	}

	log.Debugf("Downloading %s", name)
	path, err = u.sc.Download(remoteSnap, nil, nil)
	if err != nil {
		return "", &ErrRepoDownload{name, "", channel}
	}
	log.Debugf("Downloaded %s to %s", name, path)
	return
}

func (u *UDFQcow2) getSnapFlags(options *flags.Options) ([]string, error) {
	channel := GetChannel(options.OSChannel, options.KernelChannel, options.GadgetChannel)

	output := []string{
		"--channel", channel,
	}
	if options.Release != "15.04" {
		var err error
		paths := []string{}
		snaps := []string{options.OS, options.Kernel, options.Gadget}
		channels := []string{options.OSChannel, options.KernelChannel, options.GadgetChannel}
		for i := 0; i < len(snaps); i++ {
			var path string
			if channels[i] != channel {
				path, err = u.getSnapFile(snaps[i], channels[i])
				if err != nil {
					return nil, err
				}
			} else {
				path = snaps[i]
			}
			paths = append(paths, path)
		}
		output = append(output, []string{
			"--os", paths[0],
			"--kernel", paths[1],
			"--gadget", paths[2],
		}...)
	}
	return output, nil
}

// GetChannel returns the most frequent channel, if all are different it returns
// osChannel
func GetChannel(osChannel, kernelChannel, gadgetChannel string) string {
	if kernelChannel == gadgetChannel {
		return kernelChannel
	}
	return osChannel
}

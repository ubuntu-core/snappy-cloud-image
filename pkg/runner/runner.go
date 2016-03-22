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

// Package runner handles the execution entry point
package runner

import (
	"fmt"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/ubuntu-core/snappy-cloud-image/pkg/cloud"
	"github.com/ubuntu-core/snappy-cloud-image/pkg/flags"
	"github.com/ubuntu-core/snappy-cloud-image/pkg/image"
)

const imagesToKeep = 3

// Runner is the main type of the package
type Runner struct {
	imgDataOrigin image.Pollster
	imgDataTarget image.PollsterWriter
	imgDriver     image.Driver
}

// NewRunner is the Runner constructor
func NewRunner(imgDataOrigin image.Pollster, imgDataTarget image.PollsterWriter, imgDriver image.Driver) *Runner {
	return &Runner{imgDataOrigin: imgDataOrigin, imgDataTarget: imgDataTarget, imgDriver: imgDriver}
}

// ErrVersion is the type of the error returned by Exec when the version
// in SI is greater than or equal the version in cloud
type ErrVersion struct {
	siVersion, cloudVersion int
}

func (e *ErrVersion) Error() string {
	return fmt.Sprintf("error SI version %d is not greater than cloud version %d", e.siVersion, e.cloudVersion)
}

// ErrActionUnknown is the type of the error returned by Exec when the
// action given is not recognized
type ErrActionUnknown struct {
	action string
}

func (e *ErrActionUnknown) Error() string {
	return fmt.Sprintf("error unknown action %s", e.action)
}

// Exec is the main entry point, it interprets the given options and
// handles the logic of the utility
func (r *Runner) Exec(options *flags.Options) (err error) {
	if options.Action == "create" {
		return r.create(options)
	} else if options.Action == "cleanup" {
		return r.cleanup(options)
	} else if options.Action == "purge" {
		return r.purge(options)
	}
	return &ErrActionUnknown{action: options.Action}
}

func (r *Runner) create(options *flags.Options) (err error) {
	log.Infof("Checking current versions for release %s, os channel %s, kernel channel %s, gadget channel %s and arch %s",
		options.Release, options.OSChannel, options.KernelChannel, options.GadgetChannel, options.Arch)
	var siVersion, cloudVersion int

	if options.Release == "15.04" {
		siVersion, cloudVersion, err = r.getVersions(options)
		if err != nil {
			return
		}
		if siVersion <= cloudVersion {
			return &ErrVersion{siVersion, cloudVersion}
		}
	}
	var path string
	path, err = r.imgDriver.Create(options, siVersion)
	defer os.Remove(path)
	log.Infof("Creating image file in %s", path)
	if err != nil {
		return
	}

	log.Infof("Uploading %s", path)
	err = r.imgDataTarget.Create(path, options, siVersion)
	if err != nil {
		return
	}
	log.Info("Finished", path)
	return

}

func (r *Runner) getVersions(options *flags.Options) (siVersion, cloudVersion int, err error) {
	var siError, cloudError error
	versionChan := make(chan struct{}, 2)

	go func() {
		siVersion, siError = r.imgDataOrigin.GetLatestVersion(options)
		log.Info("siVersion: ", siVersion)
		versionChan <- struct{}{}
	}()

	go func() {
		cloudVersion, cloudError = r.imgDataTarget.GetLatestVersion(options)
		log.Info("cloudVersion: ", cloudVersion)
		versionChan <- struct{}{}
	}()

	for i := 0; i < 2; i++ {
		<-versionChan
	}

	if siError != nil {
		return 0, 0, siError
	}
	if cloudError != nil {
		if _, ok := cloudError.(*cloud.ErrVersionNotFound); !ok {
			return 0, 0, cloudError
		}
	}
	return
}

func (r *Runner) cleanup(options *flags.Options) (err error) {
	options.Release = strings.Replace(options.Release, ".", "", 1)
	imageList, err := r.imgDataTarget.GetVersions(options)
	if err != nil {
		log.Info("Error getting image list")
		return
	}
	if len(imageList) > imagesToKeep {
		// assumes that imageList is sorted in descending order,
		// the last items in the list will be the older ones
		log.Infof("Removing images %s", imageList[imagesToKeep:])
		err = r.imgDataTarget.Delete(imageList[imagesToKeep:]...)
	}
	return
}

func (r *Runner) purge(options *flags.Options) (err error) {
	return r.imgDataTarget.Purge(options)
}

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

// Package runner handles the execution entry point
package runner

import (
	"fmt"
	"os"
	"sync"

	"github.com/fgimenez/snappy-cloud-image/pkg/flags"
	"github.com/fgimenez/snappy-cloud-image/pkg/image"
)

// Runner is the main type of the package
type Runner struct {
	imgDataOrigin image.Pollster
	imgDataTarget image.PollsterCreator
	imgDriver     image.Driver
}

// ErrVersion is the type of the error returned by Exec when the version
// in SI is greater than or equal the version in cloud
type ErrVersion struct {
	siVersion, cloudVersion int
}

func (e *ErrVersion) Error() string {
	return fmt.Sprintf("error SI version %d is lower than cloud version %d", e.siVersion, e.cloudVersion)
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
		var siVersion, cloudVersion int
		siVersion, cloudVersion, err = r.getVersions(options.Release, options.Channel, options.Arch)
		if err != nil {
			return
		}
		if siVersion <= cloudVersion {
			return &ErrVersion{siVersion, cloudVersion}
		}
		var path string
		path, err = r.imgDriver.Create(options.Release, options.Channel, options.Arch, siVersion)
		defer os.Remove(path)
		if err != nil {
			return
		}
		err = r.imgDataTarget.Create(path, options.Release, options.Channel, options.Arch, siVersion)
		if err != nil {
			return
		}
	} else {
		return &ErrActionUnknown{action: options.Action}
	}
	return
}

func (r *Runner) getVersions(release, channel, arch string) (siVersion, cloudVersion int, err error) {
	var wg sync.WaitGroup
	var siError, cloudError error

	wg.Add(1)
	go func() {
		siVersion, siError = r.imgDataOrigin.GetLatestVersion(release, channel, arch)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		cloudVersion, cloudError = r.imgDataTarget.GetLatestVersion(release, channel, arch)
		wg.Done()
	}()

	wg.Wait()
	if siError != nil {
		return 0, 0, siError
	}
	if cloudError != nil {
		return 0, 0, cloudError
	}
	return
}

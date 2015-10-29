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

	"github.com/fgimenez/snappy-cloud-image/pkg/flags"
	"github.com/fgimenez/snappy-cloud-image/pkg/image"
)

// Runner is the main type of the package
type Runner struct {
	imgDataOrigin image.Pollster
	imgDataTarget image.PollsterCreator
	imgDriver     image.Driver
}

// NewRunner is the Runner constructor
func NewRunner(imgDataOrigin image.Pollster, imgDataTarget image.PollsterCreator, imgDriver image.Driver) *Runner {
	return &Runner{imgDataOrigin: imgDataOrigin, imgDataTarget: imgDataTarget, imgDriver: imgDriver}
}

// ErrVersion is the type of the error returned by Exec when the version
// in SI is greater than or equal the version in cloud
type ErrVersion struct {
	siVersion, cloudVersion int
}

func (e *ErrVersion) Error() string {
	return fmt.Sprintf("error SI version %d is lower than cloud version %d", e.siVersion, e.cloudVersion)
}

// Exec is the main entry point, it interprets the given options and
// handles the logic of the utility
func Exec(options *flags.Options) (err error) {
	return
}

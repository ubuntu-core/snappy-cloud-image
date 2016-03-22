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

// Package flags handles the given flags
package flags

import (
	"flag"
	"strconv"
)

// Options has fields for the existing flags
type Options struct {
	Action, Release,
	Arch, LogLevel, Qcow2compat,
	OS, Kernel, Gadget, ImageType,
	OSChannel, GadgetChannel, KernelChannel string
}

const (
	defaultAction        = "create"
	defaultRelease       = "rolling"
	defaultArch          = "amd64"
	defaultLogLevel      = "info"
	defaultQcow2compat   = "1.1"
	defaultKernel        = "canonical-pc-linux.canonical"
	defaultOS            = "ubuntu-core.canonical"
	defaultGadget        = "canonical-pc.canonical"
	defaultImageType     = "custom"
	defaultOSChannel     = "edge"
	defaultGadgetChannel = "edge"
	defaultKernelChannel = "edge"
)

// Parse analyzes the flags and returns a Options instance with the values
func Parse() *Options {
	var (
		action      = flag.String("action", defaultAction, "action to be performed")
		release     = flag.String("release", defaultRelease, "release of the image to be created")
		arch        = flag.String("arch", defaultArch, "arch of the image to be created")
		logLevel    = flag.String("loglevel", defaultLogLevel, "Level of the log putput, one of debug, info, warning, error, fatal, panic")
		qcow2compat = flag.String("qcow2compat", defaultQcow2compat, "Qcow2 compatibility level (0.10 or 1.1)")
		os          = flag.String("os", defaultOS,
			"OS snap of the image to be built, defaults to "+defaultOS)
		kernel = flag.String("kernel", defaultKernel,
			"Kernel snap of the image to be built, defaults to "+defaultKernel)
		gadget = flag.String("gadget", defaultGadget,
			"Gadget snap of the image to be built, defaults to "+defaultGadget)
		imageType = flag.String("image-type", defaultImageType,
			"Type of image to be built, this string will be put in the image name. Defaults to "+defaultImageType)
		osChannel = flag.String("os-channel", defaultOSChannel,
			"Store channel to be used for the OS snap.")
		gadgetChannel = flag.String("gadget-channel", defaultGadgetChannel,
			"Store channel to be used for the gadget snap.")
		kernelChannel = flag.String("kernel-channel", defaultKernelChannel,
			"Store channel to be used for the kernel snap.")
	)
	flag.Parse()
	dotRelease := addDot(*release)
	return &Options{
		Action:        *action,
		Release:       dotRelease,
		Arch:          *arch,
		LogLevel:      *logLevel,
		Qcow2compat:   *qcow2compat,
		OS:            *os,
		Kernel:        *kernel,
		Gadget:        *gadget,
		ImageType:     *imageType,
		OSChannel:     *osChannel,
		GadgetChannel: *gadgetChannel,
		KernelChannel: *kernelChannel,
	}
}

func addDot(release string) string {
	if len(release) == 4 {
		if _, err := strconv.Atoi(release); err == nil {
			return release[0:2] + "." + release[2:]
		}
	}
	return release
}

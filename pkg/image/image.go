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

package image

// Pollster holds the methods for querying an image backend
type Pollster interface {
	GetLatestVersion(release, channel, arch string) (ver int, err error)
}

// PollsterCreator is a Pollster that can also create new images
type PollsterCreator interface {
	Pollster
	Create(filePath, release, channel, arch string, version int) (err error)
}

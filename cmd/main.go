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

package main

import (
	"log"

	"github.com/fgimenez/snappy-cloud-image/pkg/cli"
	"github.com/fgimenez/snappy-cloud-image/pkg/cloud"
	"github.com/fgimenez/snappy-cloud-image/pkg/flags"
	"github.com/fgimenez/snappy-cloud-image/pkg/image"
	"github.com/fgimenez/snappy-cloud-image/pkg/runner"
	"github.com/fgimenez/snappy-cloud-image/pkg/si"
	"github.com/fgimenez/snappy-cloud-image/pkg/web"
)

func main() {
	parsedFlags := flags.Parse()

	cliExecutor := &cli.Executor{}
	httpClient := &web.Client{}

	imgDataOrigin := si.NewClient(httpClient)
	imgDataTarget := cloud.NewClient(cliExecutor)
	imgDriver := image.NewUDF(cliExecutor)

	runner := runner.NewRunner(imgDataOrigin, imgDataTarget, imgDriver)
	if err := runner.Exec(parsedFlags); err != nil {
		log.Panic(err.Error())
	}
}

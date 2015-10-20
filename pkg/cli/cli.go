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

// Package cli handles the interaction with the command line
package cli

import "os/exec"

var (
	execCommand = exec.Command
)

// Commander comprises the methods required by a generic command executor
type Commander interface {
	ExecCommand(...string) (output string, err error)
}

// Executor is a concrete type for CLI execution
type Executor struct{}

// ExecCommand sends the given command to the CLI and returns the output and
// the resulting error
func (e *Executor) ExecCommand(cmds ...string) (output string, err error) {
	cmd := execCommand(cmds[0], cmds[1:]...)
	outputByte, err := cmd.CombinedOutput()
	output = string(outputByte)
	return
}

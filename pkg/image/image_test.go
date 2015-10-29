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

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/check.v1"
)

const (
	testDefaultRelease = "rolling"
	testDefaultChannel = "edge"
	testDefaultArch    = "amd64"
	tmpFileName        = "tmpfilename"
)

var _ = check.Suite(&imageSuite{})

func Test(t *testing.T) { check.TestingT(t) }

type imageSuite struct {
	subject Driver
	cli     *fakeCliCommander
}

type fakeCliCommander struct {
	execCommandCalls map[string]int
	err              bool
	output           string
}

func (f *fakeCliCommander) ExecCommand(cmds ...string) (output string, err error) {
	f.execCommandCalls[strings.Join(cmds, " ")]++
	if f.err {
		err = fmt.Errorf("exec error")
	}
	return f.output, err
}

func (s *imageSuite) SetUpSuite(c *check.C) {
	s.cli = &fakeCliCommander{}
	s.subject = NewUDF(s.cli)
}

func (s *imageSuite) SetUpTest(c *check.C) {
	s.cli.execCommandCalls = make(map[string]int)
	s.cli.err = false
	s.cli.output = ""
}

func (s *imageSuite) TestCreateCallsUDF(c *check.C) {
	s.cli.output = tmpFileName
	testCases := []struct {
		release, channel, arch, expectedCall string
	}{
		{"15.04", "edge", "amd64", "sudo ubuntu-device-flash core 15.04 --channel edge --developer-mode  -o " + tmpFileName},
		{"rolling", "stable", "amd64", "sudo ubuntu-device-flash core rolling --channel stable --developer-mode  -o " + tmpFileName},
		{"15.04", "alpha", "arm", "sudo ubuntu-device-flash core 15.04 --channel alpha --developer-mode --oem beagleblack -o " + tmpFileName},
	}

	for _, item := range testCases {
		s.cli.execCommandCalls = make(map[string]int)
		_, err := s.subject.Create(item.release, item.channel, item.arch)

		c.Check(err, check.IsNil)

		c.Assert(len(s.cli.execCommandCalls) > 0, check.Equals, true)

		c.Check(s.cli.execCommandCalls[item.expectedCall], check.Equals, 1)
	}
}

func (s *imageSuite) TestCreateReturnsUDFError(c *check.C) {
	s.cli.err = true

	_, err := s.subject.Create(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(err, check.NotNil)
}

func (s *imageSuite) TestCreateReturnsCreatedFilePath(c *check.C) {
	s.cli.output = tmpFileName
	path, err := s.subject.Create(testDefaultRelease, testDefaultChannel, testDefaultArch)
	c.Assert(err, check.IsNil)

	c.Assert(path, check.Not(check.Equals), "")
	c.Assert(path, check.Equals, tmpFileName)
}

func (s *imageSuite) TestCreateUsesTmpFileName(c *check.C) {
	_, err := s.subject.Create(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(s.cli.execCommandCalls["mktemp"], check.Equals, 1)
	c.Assert(err, check.IsNil)
}

func extractKey(m map[string]int, order int) string {
	keys := []string{}
	for key := range m {
		keys = append(keys, key)
	}
	return keys[order]
}

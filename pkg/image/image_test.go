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
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/check.v1"

	"github.com/ubuntu-core/snappy-cloud-image/pkg/flags"
)

const (
	testDefaultRelease     = "rolling"
	testDefaultChannel     = "edge"
	testDefaultArch        = "amd64"
	testDefaultQcow2compat = "1.1"
	testDefaultVer         = 100
	tmpDirName             = "tmpdirname"
)

var _ = check.Suite(&imageSuite{})

func Test(t *testing.T) { check.TestingT(t) }

type imageSuite struct {
	subject        Driver
	cli            *fakeCliCommander
	defaultOptions *flags.Options
}

type fakeCliCommander struct {
	execCommandCalls         map[string]int
	err                      bool
	correctCalls, totalCalls int
	output                   string
}

func (f *fakeCliCommander) ExecCommand(cmds ...string) (output string, err error) {
	f.execCommandCalls[strings.Join(cmds, " ")]++
	f.totalCalls++
	if f.err {
		if f.totalCalls > f.correctCalls {
			err = fmt.Errorf("exec error")
		}
	}
	return f.output, err
}

func (s *imageSuite) SetUpSuite(c *check.C) {
	s.cli = &fakeCliCommander{}
	s.subject = NewUDFQcow2(s.cli)
	s.defaultOptions = &flags.Options{
		Release:     testDefaultRelease,
		Channel:     testDefaultChannel,
		Arch:        testDefaultArch,
		Qcow2compat: testDefaultQcow2compat,
	}
}

func (s *imageSuite) SetUpTest(c *check.C) {
	s.cli.execCommandCalls = make(map[string]int)
	s.cli.err = false
	s.cli.correctCalls = 0
	s.cli.totalCalls = 0
	s.cli.output = ""
}

func (s *imageSuite) TestCreateCallsUDF(c *check.C) {
	s.cli.output = tmpDirName
	filename := tmpRawFileName()
	testCases := []struct {
		release, channel, arch string
		version                int
		expectedCall           string
	}{
		{"15.04", "edge", "amd64", 100, "sudo ubuntu-device-flash --revision=100 core 15.04 --channel edge --developer-mode  -o " + filename},
		{"rolling", "stable", "amd64", 100, "sudo ubuntu-device-flash --revision=100 core rolling --channel stable --developer-mode  -o " + filename},
		{"15.04", "alpha", "arm", 56, "sudo ubuntu-device-flash --revision=56 core 15.04 --channel alpha --developer-mode --oem beagleblack -o " + filename},
	}

	for _, item := range testCases {
		s.cli.execCommandCalls = make(map[string]int)
		options := &flags.Options{
			Release:     item.release,
			Channel:     item.channel,
			Arch:        item.arch,
			Qcow2compat: testDefaultQcow2compat,
		}
		_, err := s.subject.Create(options, item.version)

		c.Check(err, check.IsNil)

		c.Assert(len(s.cli.execCommandCalls) > 0, check.Equals, true)
		c.Check(s.cli.execCommandCalls[item.expectedCall], check.Equals, 1)
	}
}

func (s *imageSuite) TestCreateDoesNotCallUDFOnMktempError(c *check.C) {
	s.cli.err = true
	s.cli.output = tmpDirName
	filename := tmpRawFileName()

	s.subject.Create(s.defaultOptions, testDefaultVer)

	expectedCall := fmt.Sprintf("sudo ubuntu-device-flash --revision=%d core %s --channel %s --developer-mode  -o %s",
		100, testDefaultRelease, testDefaultChannel, filename)

	c.Assert(s.cli.execCommandCalls[expectedCall], check.Equals, 0)
}

func (s *imageSuite) TestCreateReturnsUDFError(c *check.C) {
	s.cli.err = true
	s.cli.correctCalls = 1

	_, err := s.subject.Create(s.defaultOptions, testDefaultVer)

	c.Assert(err, check.NotNil)
}

func (s *imageSuite) TestCreateReturnsCreatedFilePath(c *check.C) {
	s.cli.output = tmpDirName
	path, err := s.subject.Create(s.defaultOptions, testDefaultVer)
	c.Assert(err, check.IsNil)

	c.Assert(path, check.Equals, tmpFileName())
}

func (s *imageSuite) TestCreateUsesTmpFileName(c *check.C) {
	_, err := s.subject.Create(s.defaultOptions, testDefaultVer)

	c.Assert(s.cli.execCommandCalls["mktemp -d"], check.Equals, 1)
	c.Assert(err, check.IsNil)
}

func (s *imageSuite) TestCreateTransformsToQCOW2(c *check.C) {
	s.cli.output = tmpDirName
	rawFilename := tmpRawFileName()
	filename := tmpFileName()

	s.subject.Create(s.defaultOptions, testDefaultVer)

	expectedCall := getExpectedCall(testDefaultQcow2compat, rawFilename, filename)
	c.Assert(s.cli.execCommandCalls[expectedCall], check.Equals, 1)
}

func (s *imageSuite) TestCreateDoesNotTransformToQCOW2OnUDFError(c *check.C) {
	s.cli.err = true
	s.cli.correctCalls = 1
	s.cli.output = tmpDirName
	rawFilename := tmpRawFileName()
	filename := tmpFileName()

	s.subject.Create(s.defaultOptions, testDefaultVer)

	expectedCall := getExpectedCall(testDefaultQcow2compat, rawFilename, filename)
	c.Assert(s.cli.execCommandCalls[expectedCall], check.Equals, 0)
}

func extractKey(m map[string]int, order int) string {
	keys := []string{}
	for key := range m {
		keys = append(keys, key)
	}
	return keys[order]
}

func tmpRawFileName() string {
	return filepath.Join(tmpDirName, rawOutputFileName)
}

func tmpFileName() string {
	return filepath.Join(tmpDirName, outputFileName)
}

func getExpectedCall(compat, inputFile, outputFile string) string {
	return fmt.Sprintf("/usr/bin/qemu-img convert -O qcow2 -o compat=%s %s %s",
		compat, inputFile, outputFile)
}

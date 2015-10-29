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

package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/fgimenez/snappy-cloud-image/pkg/flags"

	"gopkg.in/check.v1"
)

const (
	siVersionError    = "error getting si version"
	cloudVersionError = "error getting cloud version"
	cloudCreateError  = "error creating cloud image"
	udfCreateError    = "error creating image"
)

var _ = check.Suite(&runnerSuite{})

func Test(t *testing.T) { check.TestingT(t) }

type runnerSuite struct {
	subject     *Runner
	options     *flags.Options
	siClient    *fakeSiClient
	cloudClient *fakeCloudClient
	udfDriver   *fakeImgDriver
}

type fakeSiClient struct {
	getVersionCalls map[string]int
	doErr           bool
	version         int
}

func (s *fakeSiClient) GetLatestVersion(release, channel, arch string) (ver int, err error) {
	key := getFakeKey(release, channel, arch)
	s.getVersionCalls[key]++
	if s.doErr {
		err = fmt.Errorf(siVersionError)
	}
	return s.version, err
}

type fakeCloudClient struct {
	getVersionCalls map[string]int
	createCalls     map[string]int
	doVerErr        bool
	doCreateErr     bool
	version         int
}

func (s *fakeCloudClient) GetLatestVersion(release, channel, arch string) (ver int, err error) {
	key := getFakeKey(release, channel, arch)
	s.getVersionCalls[key]++
	if s.doVerErr {
		err = fmt.Errorf(cloudVersionError)
	}
	return s.version, err
}

func (s *fakeCloudClient) Create(filePath, release, channel, arch string, version int) (err error) {
	key := getFullCreateKey(filePath, release, channel, arch, version)
	s.createCalls[key]++
	if s.doCreateErr {
		err = fmt.Errorf(cloudCreateError)
	}
	return
}

type fakeImgDriver struct {
	createCalls map[string]int
	path        string
	doErr       bool
}

func (s *fakeImgDriver) Create(release, channel, arch string, version int) (path string, err error) {
	key := getCreateKey(release, channel, arch, version)
	s.createCalls[key]++
	if s.doErr {
		err = fmt.Errorf(udfCreateError)
	}
	return s.path, err
}

func (s *runnerSuite) SetUpSuite(c *check.C) {
	s.siClient = &fakeSiClient{}
	s.cloudClient = &fakeCloudClient{}
	s.udfDriver = &fakeImgDriver{}
	s.subject = NewRunner(s.siClient, s.cloudClient, s.udfDriver)
	s.options = &flags.Options{
		Action: "create", Release: "15.04", Channel: "edge", Arch: "amd64"}
}

func (s *runnerSuite) SetUpTest(c *check.C) {
	s.siClient.getVersionCalls = make(map[string]int)
	s.siClient.doErr = false
	s.siClient.version = 2
	s.cloudClient.getVersionCalls = make(map[string]int)
	s.cloudClient.createCalls = make(map[string]int)
	s.cloudClient.doVerErr = false
	s.cloudClient.doCreateErr = false
	s.cloudClient.version = 1
	s.udfDriver.createCalls = make(map[string]int)
	s.udfDriver.doErr = false
	s.udfDriver.path = "path"
	s.options.Action = "create"
}

func (s *runnerSuite) TestExecGetsSIVersion(c *check.C) {
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFakeKey(s.options.Release, s.options.Channel, s.options.Arch)
	c.Assert(s.siClient.getVersionCalls[key], check.Equals, 1)
}

func (s *runnerSuite) TestExecDoesNotGetSIVersionOnNonCreateAction(c *check.C) {
	s.options.Action = "non-create"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)

	c.Assert(len(s.siClient.getVersionCalls), check.Equals, 0)
}

func (s *runnerSuite) TestExecReturnsGetSIVersionError(c *check.C) {
	s.siClient.doErr = true
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, siVersionError)
}

func (s *runnerSuite) TestExecGetsCloudVersion(c *check.C) {
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFakeKey(s.options.Release, s.options.Channel, s.options.Arch)
	c.Assert(s.cloudClient.getVersionCalls[key], check.Equals, 1)
}

func (s *runnerSuite) TestExecDoesNotGetCloudVersionOnNonCreateAction(c *check.C) {
	s.options.Action = "non-create"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)

	c.Assert(len(s.cloudClient.getVersionCalls), check.Equals, 0)
}

func (s *runnerSuite) TestExecReturnsGetCloudVersionError(c *check.C) {
	s.cloudClient.doVerErr = true
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, cloudVersionError)
}

func (s *runnerSuite) TestExecReturnsErrVersionIfCloudVersionNotLessThanSIVersion(c *check.C) {
	testCases := []struct {
		siVersion, cloudVersion int
		expectedError           error
	}{
		{99, 100, &ErrVersion{siVersion: 99, cloudVersion: 100}},
		{99, 99, &ErrVersion{siVersion: 99, cloudVersion: 99}},
	}
	for _, item := range testCases {
		s.siClient.version = item.siVersion
		s.cloudClient.version = item.cloudVersion

		err := s.subject.Exec(s.options)

		c.Assert(err, check.FitsTypeOf, item.expectedError)
		c.Assert(err.Error(), check.Equals, item.expectedError.Error())
	}
}

func (s *runnerSuite) TestExecCallsDriverCreate(c *check.C) {
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getCreateKey(s.options.Release, s.options.Channel, s.options.Arch, s.siClient.version)
	c.Assert(s.udfDriver.createCalls[key], check.Equals, 1)
}

func (s *runnerSuite) TestExecDoesNotCallDriverCreateOnNonCreateAction(c *check.C) {
	s.options.Action = "non-create"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)

	c.Assert(len(s.udfDriver.createCalls), check.Equals, 0)
}

func (s *runnerSuite) TestExecReturnsDriverCreateError(c *check.C) {
	s.udfDriver.doErr = true
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, udfCreateError)
}

func (s *runnerSuite) TestExecCallsCloudCreate(c *check.C) {
	s.udfDriver.path = "mypath"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFullCreateKey("mypath", s.options.Release, s.options.Channel, s.options.Arch, s.siClient.version)
	c.Assert(s.cloudClient.createCalls[key], check.Equals, 1)
}

func (s *runnerSuite) TestExecDoesNotCallCloudCreateOnNonCreateAction(c *check.C) {
	s.options.Action = "non-create"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)

	c.Assert(len(s.udfDriver.createCalls), check.Equals, 0)
}

func (s *runnerSuite) TestExecReturnsCloudCreateError(c *check.C) {
	s.cloudClient.doCreateErr = true
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, cloudCreateError)
}

func (s *runnerSuite) TestExecRemovesImageFile(c *check.C) {
	tmpFile, err := ioutil.TempFile("", "")
	defer os.Remove(tmpFile.Name())
	c.Assert(err, check.IsNil)

	s.udfDriver.path = tmpFile.Name()

	s.subject.Exec(s.options)

	_, err = os.Stat(tmpFile.Name())

	c.Assert(os.IsNotExist(err), check.Equals, true)
}

func (s *runnerSuite) TestExecReturnsErrorOnInvalidAction(c *check.C) {
	s.options.Action = "non-create"
	err := s.subject.Exec(s.options)
	expectedError := &ErrActionUnknown{action: s.options.Action}

	c.Assert(err, check.FitsTypeOf, expectedError)
	c.Assert(err.Error(), check.Equals, expectedError.Error())
}

func getFakeKey(release, channel, arch string) string {
	return fmt.Sprintf("%s - %s - %s", release, channel, arch)
}

func getCreateKey(release, channel, arch string, ver int) string {
	return fmt.Sprintf("%s - %d", getFakeKey(release, channel, arch), ver)
}

func getFullCreateKey(path, release, channel, arch string, ver int) string {
	return fmt.Sprintf("%s - %s", path, getCreateKey(release, channel, arch, ver))
}

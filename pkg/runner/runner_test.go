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
	"strconv"
	"strings"
	"testing"

	"github.com/ubuntu-core/snappy-cloud-image/pkg/cloud"
	"github.com/ubuntu-core/snappy-cloud-image/pkg/flags"

	"gopkg.in/check.v1"
)

const (
	siVersionError          = "error getting si version"
	cloudLatestVersionError = "error getting latest cloud version"
	cloudVersionsError      = "error getting cloud versions"
	cloudCreateError        = "error creating cloud image"
	cloudDeleteError        = "error deleting cloud images"
	udfCreateError          = "error creating image"
)

var _ = check.Suite(&runnerCreateSuite{})
var _ = check.Suite(&runnerCleanupSuite{})

func Test(t *testing.T) { check.TestingT(t) }

type runnerCreateSuite struct {
	subject     *Runner
	options     *flags.Options
	siClient    *fakeSiClient
	cloudClient *fakeCloudClient
	udfDriver   *fakeImgDriver
}

type runnerCleanupSuite struct {
	subject     *Runner
	options     *flags.Options
	cloudClient *fakeCloudClient
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
	getLatestVersionCalls map[string]int
	getVersionsCalls      map[string]int
	createCalls           map[string]int
	deleteCalls           map[string]int
	doVerErr              bool
	doVerNotFoundErr      bool
	doCreateErr           bool
	doDeleteErr           bool
	version               int
	versions              []string
}

func (s *fakeCloudClient) GetLatestVersion(release, channel, arch string) (ver int, err error) {
	key := getFakeKey(release, channel, arch)
	s.getLatestVersionCalls[key]++
	if s.doVerErr {
		err = fmt.Errorf(cloudLatestVersionError)
	}
	if s.doVerNotFoundErr {
		err = cloud.NewErrVersionNotFound(release, channel, arch)
	}
	return s.version, err
}

func (s *fakeCloudClient) GetVersions(release, channel, arch string) (images []string, err error) {
	key := getFakeKey(release, channel, arch)
	s.getVersionsCalls[key]++
	if s.doVerErr {
		err = fmt.Errorf(cloudVersionsError)
	}
	return s.versions, err
}

func (s *fakeCloudClient) Create(filePath, release, channel, arch string, version int) (err error) {
	key := getFullCreateKey(filePath, release, channel, arch, version)
	s.createCalls[key]++
	if s.doCreateErr {
		err = fmt.Errorf(cloudCreateError)
	}
	return
}

func (s *fakeCloudClient) Delete(versions ...string) (err error) {
	key := getDeleteKey(versions)
	s.deleteCalls[key]++
	if s.doDeleteErr {
		err = fmt.Errorf(cloudDeleteError)
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

func (s *runnerCreateSuite) SetUpSuite(c *check.C) {
	s.siClient = &fakeSiClient{}
	s.cloudClient = &fakeCloudClient{}
	s.udfDriver = &fakeImgDriver{}
	s.subject = NewRunner(s.siClient, s.cloudClient, s.udfDriver)
	s.options = &flags.Options{
		Action: "create", Release: "15.04", Channel: "edge", Arch: "amd64"}
}

func (s *runnerCreateSuite) SetUpTest(c *check.C) {
	s.siClient.getVersionCalls = make(map[string]int)
	s.siClient.doErr = false
	s.siClient.version = 2
	s.cloudClient.getLatestVersionCalls = make(map[string]int)
	s.cloudClient.createCalls = make(map[string]int)
	s.cloudClient.doVerErr = false
	s.cloudClient.doVerNotFoundErr = false
	s.cloudClient.doCreateErr = false
	s.cloudClient.version = 1
	s.udfDriver.createCalls = make(map[string]int)
	s.udfDriver.doErr = false
	s.udfDriver.path = "path"
	s.options.Action = "create"
	s.options.Release = "15.04"
}

func (s *runnerCleanupSuite) SetUpSuite(c *check.C) {
	s.cloudClient = &fakeCloudClient{}
	s.subject = NewRunner(&fakeSiClient{}, s.cloudClient, &fakeImgDriver{})
	s.options = &flags.Options{
		Action: "cleanup", Release: "15.04", Channel: "edge", Arch: "amd64"}
}

func (s *runnerCleanupSuite) SetUpTest(c *check.C) {
	s.cloudClient.getVersionsCalls = make(map[string]int)
	s.cloudClient.deleteCalls = make(map[string]int)
	s.cloudClient.doVerErr = false
	s.cloudClient.doDeleteErr = false
	s.cloudClient.versions = []string{}
	s.options.Action = "cleanup"
}

func (s *runnerCreateSuite) TestExecCreateGetsSIVersion(c *check.C) {
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFakeKey(s.options.Release, s.options.Channel, s.options.Arch)
	c.Assert(s.siClient.getVersionCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecDoesNotGetSIVersionOnNonCreateAction(c *check.C) {
	s.options.Action = "non-create"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)

	c.Assert(len(s.siClient.getVersionCalls), check.Equals, 0)
}

func (s *runnerCreateSuite) TestExecReturnsGetSIVersionError(c *check.C) {
	s.siClient.doErr = true
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, siVersionError)
}

func (s *runnerCreateSuite) TestExecGetSIVersionReceivesReleaseWithDot(c *check.C) {
	s.options.Release = "1504"

	s.subject.Exec(s.options)

	key := getFakeKey("15.04", s.options.Channel, s.options.Arch)
	c.Assert(s.siClient.getVersionCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecGetsCloudLatestVersion(c *check.C) {
	s.options.Release = "1504"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFakeKey(s.options.Release, s.options.Channel, s.options.Arch)
	c.Assert(s.cloudClient.getLatestVersionCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecGetCloudLatestVersionReceivesReleaseWithoutDot(c *check.C) {
	s.options.Release = "15.04"
	s.subject.Exec(s.options)

	key := getFakeKey("1504", s.options.Channel, s.options.Arch)
	c.Assert(s.cloudClient.getLatestVersionCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecDoesNotGetCloudVersionOnNonCreateAction(c *check.C) {
	s.options.Action = "non-create"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)

	c.Assert(len(s.cloudClient.getLatestVersionCalls), check.Equals, 0)
}

func (s *runnerCreateSuite) TestExecReturnsGetCloudVersionError(c *check.C) {
	s.cloudClient.doVerErr = true
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, cloudLatestVersionError)
}

func (s *runnerCreateSuite) TestExecDoesNotReturnGetCloudVersionNotFoundError(c *check.C) {
	s.cloudClient.doVerNotFoundErr = true
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)
}

func (s *runnerCreateSuite) TestExecReturnsErrVersionIfCloudVersionNotLessThanSIVersion(c *check.C) {
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

func (s *runnerCreateSuite) TestExecCallsDriverCreate(c *check.C) {
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getCreateKey(s.options.Release, s.options.Channel, s.options.Arch, s.siClient.version)
	c.Assert(s.udfDriver.createCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecDoesNotCallDriverCreateOnNonCreateAction(c *check.C) {
	s.options.Action = "non-create"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)

	c.Assert(len(s.udfDriver.createCalls), check.Equals, 0)
}

func (s *runnerCreateSuite) TestExecDriverCreateReceivesReleaseWithDot(c *check.C) {
	s.options.Release = "1504"

	s.subject.Exec(s.options)

	key := getCreateKey("15.04", s.options.Channel, s.options.Arch, s.siClient.version)
	c.Assert(s.udfDriver.createCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecReturnsDriverCreateError(c *check.C) {
	s.udfDriver.doErr = true
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, udfCreateError)
}

func (s *runnerCreateSuite) TestExecCallsCloudCreate(c *check.C) {
	s.udfDriver.path = "mypath"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFullCreateKey("mypath", s.options.Release, s.options.Channel, s.options.Arch, s.siClient.version)
	c.Assert(s.cloudClient.createCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecDoesNotCallCloudCreateOnNonCreateAction(c *check.C) {
	s.options.Action = "non-create"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)

	c.Assert(len(s.udfDriver.createCalls), check.Equals, 0)
}

func (s *runnerCreateSuite) TestExecReturnsCloudCreateError(c *check.C) {
	s.cloudClient.doCreateErr = true
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, cloudCreateError)
}

func (s *runnerCreateSuite) TestExecRemovesImageFile(c *check.C) {
	tmpFile, err := ioutil.TempFile("", "")
	defer os.Remove(tmpFile.Name())
	c.Assert(err, check.IsNil)

	s.udfDriver.path = tmpFile.Name()

	s.subject.Exec(s.options)

	_, err = os.Stat(tmpFile.Name())

	c.Assert(os.IsNotExist(err), check.Equals, true)
}

func (s *runnerCreateSuite) TestExecReturnsErrorOnInvalidAction(c *check.C) {
	s.options.Action = "invalid-action"
	err := s.subject.Exec(s.options)
	expectedError := &ErrActionUnknown{action: s.options.Action}

	c.Assert(err, check.FitsTypeOf, expectedError)
	c.Assert(err.Error(), check.Equals, expectedError.Error())
}

func (s *runnerCleanupSuite) TestExecGetsCloudVersions(c *check.C) {
	s.options.Release = "1504"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFakeKey(s.options.Release, s.options.Channel, s.options.Arch)
	c.Assert(s.cloudClient.getVersionsCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecDoesNotGetCloudVersionsOnNonCleanupAction(c *check.C) {
	s.options.Action = "non-cleanup"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)

	c.Assert(len(s.cloudClient.getVersionsCalls), check.Equals, 0)
}

func (s *runnerCleanupSuite) TestExecGetVersionsReceivesReleaseWithoutDots(c *check.C) {
	s.options.Release = "15.04"
	s.subject.Exec(s.options)

	key := getFakeKey("1504", s.options.Channel, s.options.Arch)
	fmt.Println(s.cloudClient.getVersionsCalls)
	c.Assert(s.cloudClient.getVersionsCalls[key], check.Equals, 1)
}

func (s *runnerCleanupSuite) TestExecReturnsGetVersionsError(c *check.C) {
	s.cloudClient.doVerErr = true
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, cloudVersionsError)
}

func (s *runnerCleanupSuite) TestExecCallsDeleteForExcedentImages(c *check.C) {
	excedent := 2
	base := 10
	for i := imagesToKeep + excedent; i >= 0; i-- {
		s.cloudClient.versions = append(
			s.cloudClient.versions,
			cloud.GetImageID(s.options.Release, s.options.Channel, s.options.Arch, i+base))
	}

	s.subject.Exec(s.options)

	expectedCall := getDeleteKey(s.cloudClient.versions[imagesToKeep:])

	c.Assert(s.cloudClient.deleteCalls[expectedCall], check.Equals, 1)
}

func (s *runnerCleanupSuite) TestExecDoesNotCallDeleteWithouExcedentImages(c *check.C) {
	base := 10
	for i := imagesToKeep - 1; i >= 0; i-- {
		s.cloudClient.versions = append(
			s.cloudClient.versions,
			cloud.GetImageID(s.options.Release, s.options.Channel, s.options.Arch, i+base))
	}

	s.subject.Exec(s.options)

	c.Assert(len(s.cloudClient.deleteCalls), check.Equals, 0)
}

func (s *runnerCleanupSuite) TestExecDoesNotCallDeleteOnGetVersionsError(c *check.C) {
	s.cloudClient.doVerErr = true
	for i := imagesToKeep + 1; i >= 0; i-- {
		s.cloudClient.versions = append(s.cloudClient.versions, "version"+strconv.Itoa(i))
	}

	s.subject.Exec(s.options)

	c.Assert(len(s.cloudClient.deleteCalls), check.Equals, 0)
}

func (s *runnerCleanupSuite) TestExecReturnsDeleteError(c *check.C) {
	s.cloudClient.doDeleteErr = true
	for i := imagesToKeep + 1; i >= 0; i-- {
		s.cloudClient.versions = append(s.cloudClient.versions, "version"+strconv.Itoa(i))
	}

	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, cloudDeleteError)
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

func getDeleteKey(versions []string) string {
	return strings.Join(versions, " ")
}

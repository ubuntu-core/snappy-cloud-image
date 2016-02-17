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
	cloudPurgeError         = "error purging cloud images"
	udfCreateError          = "error creating image"
)

var _ = check.Suite(&runnerCreateSuite{})
var _ = check.Suite(&runnerCleanupSuite{})
var _ = check.Suite(&runnerPurgeSuite{})

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

type runnerPurgeSuite struct {
	subject     *Runner
	options     *flags.Options
	cloudClient *fakeCloudClient
}

type fakeSiClient struct {
	getVersionCalls map[string]int
	doErr           bool
	version         int
}

func (s *fakeSiClient) GetLatestVersion(options *flags.Options) (ver int, err error) {
	key := getFakeKey(options)
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
	purgeCalls            int
	doVerErr              bool
	doVerNotFoundErr      bool
	doCreateErr           bool
	doDeleteErr           bool
	doPurgeErr            bool
	version               int
	versions              []string
}

func (s *fakeCloudClient) GetLatestVersion(options *flags.Options) (ver int, err error) {
	key := getFakeKey(options)
	s.getLatestVersionCalls[key]++
	if s.doVerErr {
		err = fmt.Errorf(cloudLatestVersionError)
	}
	if s.doVerNotFoundErr {
		err = cloud.NewErrVersionNotFound(options)
	}
	return s.version, err
}

func (s *fakeCloudClient) GetVersions(options *flags.Options) (images []string, err error) {
	key := getFakeKey(options)
	s.getVersionsCalls[key]++
	if s.doVerErr {
		err = fmt.Errorf(cloudVersionsError)
	}
	return s.versions, err
}

func (s *fakeCloudClient) Create(filePath string, options *flags.Options, version int) (err error) {
	key := getFullCreateKey(filePath, options, version)
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

func (s *fakeCloudClient) Purge(options *flags.Options) (err error) {
	s.purgeCalls++
	if s.doPurgeErr {
		err = fmt.Errorf(cloudPurgeError)
	}
	return
}

type fakeImgDriver struct {
	createCalls map[string]int
	path        string
	doErr       bool
}

func (s *fakeImgDriver) Create(options *flags.Options, version int) (path string, err error) {
	key := getCreateKey(options, version)
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

func (s *runnerPurgeSuite) SetUpSuite(c *check.C) {
	s.cloudClient = &fakeCloudClient{}
	s.subject = NewRunner(&fakeSiClient{}, s.cloudClient, &fakeImgDriver{})
	s.options = &flags.Options{
		Action: "purge", Release: "15.04", Channel: "edge", Arch: "amd64"}
}

func (s *runnerPurgeSuite) SetUpTest(c *check.C) {
	s.cloudClient.purgeCalls = 0
	s.cloudClient.doPurgeErr = false
	s.options.Action = "purge"
}

func (s *runnerCreateSuite) TestExecCreateGetsSIVersionFor1504(c *check.C) {
	s.options.Release = "15.04"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFakeKey(s.options)
	c.Assert(s.siClient.getVersionCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecCreateDoesNotGetSIVersionForNon1504(c *check.C) {
	s.options.Release = "non-15.04"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFakeKey(s.options)
	c.Assert(s.siClient.getVersionCalls[key], check.Equals, 0)
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

func (s *runnerCreateSuite) TestExecGetsCloudLatestVersionFor1504(c *check.C) {
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFakeKey(s.options)
	c.Assert(s.cloudClient.getLatestVersionCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecDoesNotGetCloudLatestVersionForNon1504(c *check.C) {
	s.options.Release = "non15.04"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFakeKey(s.options)
	c.Assert(s.cloudClient.getLatestVersionCalls[key], check.Equals, 0)
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

func (s *runnerCreateSuite) TestExecCallsDriverCreateWithSIVersionFor1504(c *check.C) {
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getCreateKey(s.options, s.siClient.version)
	c.Assert(s.udfDriver.createCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecCallsDriverCreateWithZeroForNon1504(c *check.C) {
	s.options.Release = "non15.04"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getCreateKey(s.options, 0)
	c.Assert(s.udfDriver.createCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecDoesNotCallDriverCreateOnNonCreateAction(c *check.C) {
	s.options.Action = "non-create"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)

	c.Assert(len(s.udfDriver.createCalls), check.Equals, 0)
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

	key := getFullCreateKey("mypath", s.options, s.siClient.version)
	c.Assert(s.cloudClient.createCalls[key], check.Equals, 1)
}

func (s *runnerCreateSuite) TestExecCallsCloudCreateWithZeroVersionForNon1504(c *check.C) {
	s.options.Release = "non15.04"
	s.udfDriver.path = "mypath"
	err := s.subject.Exec(s.options)

	c.Assert(err, check.IsNil)

	key := getFullCreateKey("mypath", s.options, 0)
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

	key := getFakeKey(s.options)
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

	key := getFakeKey(s.options)
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
			cloud.GetImageID(s.options, i+base))
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
			cloud.GetImageID(s.options, i+base))
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

func (s *runnerPurgeSuite) TestExecCallsPurge(c *check.C) {
	s.subject.Exec(s.options)

	c.Assert(s.cloudClient.purgeCalls, check.Equals, 1)
}

func (s *runnerPurgeSuite) TestExecDoesNotCallPurgeOnNonPurgeAction(c *check.C) {
	s.options.Action = "non-purge"

	s.subject.Exec(s.options)

	c.Assert(s.cloudClient.purgeCalls, check.Equals, 0)
}

func (s *runnerPurgeSuite) TestExecReturnsPurgeError(c *check.C) {
	s.cloudClient.doPurgeErr = true

	err := s.subject.Exec(s.options)

	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, cloudPurgeError)
}

func getFakeKey(options *flags.Options) string {
	return fmt.Sprintf("%s - %s - %s", options.Release, options.Channel, options.Arch)
}

func getCreateKey(options *flags.Options, ver int) string {
	return fmt.Sprintf("%s - %d", getFakeKey(options), ver)
}

func getFullCreateKey(path string, options *flags.Options, ver int) string {
	return fmt.Sprintf("%s - %s", path, getCreateKey(options, ver))
}

func getDeleteKey(versions []string) string {
	return strings.Join(versions, " ")
}

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

package cloud

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/check.v1"

	"github.com/ubuntu-core/snappy-cloud-image/pkg/flags"
)

const (
	testDefaultRelease   = "rolling"
	testDefaultChannel   = "edge"
	testDefaultArch      = "amd64"
	testDefaultImageType = "custom"
	testImageVersion     = 198
	baseCompleteResponse = `| 06c12690-08ef-4a9b-aaa6-6e8249bcfef8 | ubuntu-released/ubuntu-oneiric-11.10-amd64-server-20130509-disk1.img                                 |
| 8fa0213b-e598-473f-bb33-901281063395 | smoser-cloud-images/ubuntu-hardy-8.04-amd64-server-20121003                                          |
| 56e4a037-887f-4e8c-8e9f-edad2060232b | smoser-cloud-images/ubuntu-hardy-8.04-amd64-server-20121003-ramdisk                                  |
| cc3fff76-6e86-4bab-93a4-74c45cf3d078 | smoser-cloud-images/ubuntu-hardy-8.04-amd64-server-20121003-kernel                                   |
%s
| f5eca345-3d7c-480d-a5de-3057ef1c5e82 | smoser-cloud-images/ubuntu-hardy-8.04-i386-server-20121003                                           |
| 45a240bc-f3c7-4e2f-99b5-7761dabd67c2 | smoser-cloud-images/ubuntu-hardy-8.04-i386-server-20121003-ramdisk                                   |
| 1e2111f6-7f02-4d07-bec2-229c8dd30559 | smoser-cloud-images/ubuntu-hardy-8.04-i386-server-20121003-kernel                                    |
| 47537aad-dcdb-422e-9302-2f874f88f216 | quantal-desktop-amd64                                                                                |
%s
%s
| f3618134-0151-48a2-8964-42574322fd52 | precise-desktop-amd64                                                                                |
| 762d5ce2-fbc2-4685-8d6c-71249d19df9e | ubuntu-core/devel/ubuntu-1504-snappy-core-amd64-edge-20151020-disk1.img                              |
| 08763be0-3b3d-41e3-b5b0-08b9006fc1d7 | smoser-lucid-loader/lucid-amd64-linux-image-2.6.32-34-virtual-v-2.6.32-34.77~smloader0-build0-loader |
| 842949c6-225b-4ad0-81b7-98de2b818eed | smoser-lucid-loader/lucid-amd64-linux-image-2.6.32-34-virtual-v-2.6.32-34.77~smloader0-kernel        |
| bf412075-2c8d-4753-8d19-4e502cf57d8d | None                                                                                                 |
%s
`
	baseResponse = "| 762d5ce2-fbc2-4685-8d6c-71249d19df9e | %s                        |"
)

type cloudSuite struct {
	subject        *Client
	cli            *fakeCliCommander
	defaultOptions *flags.Options
}

type fakeCliCommander struct {
	execCommandCalls map[string]int
	output           string
	err              bool
}

func (f *fakeCliCommander) ExecCommand(cmds ...string) (output string, err error) {
	f.execCommandCalls[strings.Join(cmds, " ")]++
	if f.err {
		err = fmt.Errorf("exec error")
	}
	return f.output, err
}

var _ = check.Suite(&cloudSuite{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *cloudSuite) SetUpSuite(c *check.C) {
	s.cli = &fakeCliCommander{}
	s.subject = NewClient(s.cli)
}

func (s *cloudSuite) SetUpTest(c *check.C) {
	s.defaultOptions = &flags.Options{
		Release:       testDefaultRelease,
		OSChannel:     testDefaultChannel,
		KernelChannel: testDefaultChannel,
		GadgetChannel: testDefaultChannel,
		Arch:          testDefaultArch,
		ImageType:     testDefaultImageType,
	}
	s.cli.execCommandCalls = make(map[string]int)
	s.cli.output = fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, testImageVersion))
	s.cli.err = false
}

func (s *cloudSuite) TestGetLatestVersionQueriesGlance(c *check.C) {
	s.subject.GetLatestVersion(s.defaultOptions)

	c.Assert(s.cli.execCommandCalls["openstack image list --private --property status=active"], check.Equals, 1)
}

func (s *cloudSuite) TestGetLatestVersionReturnsTheLatestVersion(c *check.C) {
	version := 100
	versionLine := fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, version))
	versionPlusOneLine := fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, version+1))
	versionPlusTwoLine := fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, version+2))

	testCases := []struct {
		glanceOutput    string
		expectedVersion int
	}{
		{fmt.Sprintf(baseCompleteResponse, "", "", "", ""),
			0},
		{fmt.Sprintf(baseCompleteResponse, versionLine, "", "", ""),
			version},
		{fmt.Sprintf(baseCompleteResponse, versionLine, versionPlusOneLine, "", ""),
			version + 1},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionLine, "", ""),
			version + 1},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionLine, "", versionPlusTwoLine),
			version + 2},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionPlusTwoLine, versionLine, versionPlusOneLine),
			version + 2},
	}
	for _, item := range testCases {
		s.cli.output = item.glanceOutput
		ver, _ := s.subject.GetLatestVersion(s.defaultOptions)

		c.Check(ver, check.Equals, item.expectedVersion)
	}
}

func (s *cloudSuite) TestGetLatestVersionReturnsGlanceError(c *check.C) {
	s.cli.err = true

	_, err := s.subject.GetLatestVersion(s.defaultOptions)

	c.Assert(err, check.NotNil)
}

func (s *cloudSuite) TestGetLatestVersionReturnsVersionNumberError(c *check.C) {
	s.cli.output = "| 762d5ce2-fbc2-4685-8d6c-71249d19df9e | ubuntu-core/custom/ubuntu-rolling-snappy-core-amd64-edge-10f-disk1.img |"

	_, err := s.subject.GetLatestVersion(s.defaultOptions)

	c.Assert(err, check.NotNil)
	c.Assert(err, check.FitsTypeOf, &strconv.NumError{})
}

func (s *cloudSuite) TestGetLatestVersionReturnsVersionNotFoundError(c *check.C) {
	s.cli.output = fmt.Sprintf(baseCompleteResponse, "", "", "", "")

	_, err := s.subject.GetLatestVersion(s.defaultOptions)

	c.Assert(err, check.FitsTypeOf, &ErrVersionNotFound{})
	c.Assert(err.Error(), check.Equals,
		fmt.Sprintf(errVerNotFoundPattern, testDefaultRelease, testDefaultChannel, testDefaultArch))
}

func (s *cloudSuite) TestGetLatestVersionRemovesDotFromRelease(c *check.C) {
	expectedVersion := 100
	s.defaultOptions.Release = "1604"
	versionLine := fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, expectedVersion))
	s.cli.output = fmt.Sprintf(baseCompleteResponse, versionLine, "", "", "")
	version, _ := s.subject.GetLatestVersion(s.defaultOptions)

	c.Assert(version, check.Equals, expectedVersion)
}

func (s *cloudSuite) TestCreateCallsGlance(c *check.C) {
	path := "mypath"
	version := 100
	err := s.subject.Create(path, s.defaultOptions, version)

	c.Assert(err, check.IsNil)

	imageName := getImageID(s.defaultOptions, version)
	expectedCall := fmt.Sprintf("openstack image create --disk-format qcow2 --file %s %s", path, imageName)

	c.Assert(s.cli.execCommandCalls[expectedCall], check.Equals, 1)
}

func (s *cloudSuite) TestCreateWithOneProperty(c *check.C) {
	testProperty := "testproperty='testvalue'"
	s.defaultOptions.Properties = testProperty
	path := "dummy"
	err := s.subject.Create(path, s.defaultOptions, testImageVersion)

	c.Assert(err, check.IsNil)

	imageName := getImageID(s.defaultOptions, testImageVersion)
	expectedCall := fmt.Sprintf("openstack image create --disk-format qcow2 --file %s --property %s %s",
		path, testProperty, imageName)
	c.Assert(s.cli.execCommandCalls[expectedCall], check.Equals, 1)
}

func (s *cloudSuite) TestCreateWithMultipleProperties(c *check.C) {
	testProperty := "testproperty1='testvalue1',testproperty2='testvalue2',testproperty3='testvalue3'"
	s.defaultOptions.Properties = testProperty
	path := "dummy"
	err := s.subject.Create(path, s.defaultOptions, testImageVersion)

	c.Assert(err, check.IsNil)

	expectedProperties := "--property testproperty1='testvalue1' --property testproperty2='testvalue2' --property testproperty3='testvalue3'"
	imageName := getImageID(s.defaultOptions, testImageVersion)
	expectedCall := fmt.Sprintf("openstack image create --disk-format qcow2 --file %s %s %s",	path, expectedProperties, imageName)
	c.Assert(s.cli.execCommandCalls[expectedCall], check.Equals, 1)
}


func (s *cloudSuite) TestCreateReturnsError(c *check.C) {
	s.cli.err = true

	path := "mypath"
	version := 100
	err := s.subject.Create(path, s.defaultOptions, version)

	c.Assert(err, check.NotNil)
}

func (s *cloudSuite) TestGetImageID(c *check.C) {
	testCases := []struct {
		imageType, release, channel, arch string
		version                           int
		expectedID                        string
	}{
		{"custom", "rolling", "edge", "amd64", 100, "ubuntu-core/custom/ubuntu-rolling-snappy-core-amd64-edge-100-disk1.img"},
		{"testing", "rolling", "stable", "amd64", 10, "ubuntu-core/testing/ubuntu-rolling-snappy-core-amd64-stable-10-disk1.img"},
		{"mycustom", "rolling", "alpha", "amd64", 210, "ubuntu-core/mycustom/ubuntu-rolling-snappy-core-amd64-alpha-210-disk1.img"},
		{"testing2", "15.04", "edge", "amd64", 54, "ubuntu-core/testing2/ubuntu-1504-snappy-core-amd64-edge-54-disk1.img"},
		{"custom", "1504", "stable", "amd64", 23, "ubuntu-core/custom/ubuntu-1504-snappy-core-amd64-stable-23-disk1.img"},
		{"custom", "15.04", "alpha", "amd64", 2105, "ubuntu-core/custom/ubuntu-1504-snappy-core-amd64-alpha-2105-disk1.img"},
	}
	for _, item := range testCases {
		options := &flags.Options{Release: item.release, OSChannel: item.channel, KernelChannel: item.channel, GadgetChannel: item.channel, Arch: item.arch, ImageType: item.imageType}
		c.Check(GetImageID(options, item.version), check.Equals, item.expectedID)
	}
}

func (s *cloudSuite) TestGetImageIDAssignsVersionOnZeroVersionGiven(c *check.C) {
	output := GetImageID(s.defaultOptions, 0)

	ver := getVersionFromImageID(output)

	c.Assert(ver != "0", check.Equals, true)
}

func (s *cloudSuite) TestGetImageIDAssignsIncreasingVersionNumbersOnZeroVersionGiven(c *check.C) {
	var currVer, prevVer string
	for i := 0; i < 100; i++ {
		output := GetImageID(s.defaultOptions, 0)

		currVer = getVersionFromImageID(output)

		c.Check(currVer > prevVer, check.Equals, true)
		prevVer = currVer
		fmt.Println(currVer)
	}
}

func (s *cloudSuite) TestDeleteCallsCli(c *check.C) {
	testCases := []struct {
		images       []string
		expectedCall string
	}{
		{[]string{"version1", "version2"}, "openstack image delete version1 version2"},
		{[]string{"version2", "version1"}, "openstack image delete version2 version1"},
		{[]string{"version2", "version1", "version3", "version4"}, "openstack image delete version2 version1 version3 version4"},
	}
	for _, item := range testCases {
		s.subject.Delete(item.images...)
		c.Assert(s.cli.execCommandCalls[item.expectedCall], check.Equals, 1)
	}
}

func (s *cloudSuite) TestDeleteReturnsCliError(c *check.C) {
	s.cli.err = true

	err := s.subject.Delete("image1", "image2")

	c.Assert(err, check.NotNil)
}

func (s *cloudSuite) TestGetVersionsReturnsImageNames(c *check.C) {
	version := 100
	versionLine := fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, version))
	versionID := getIDFromGlanceResponse(versionLine)
	versionPlusOneLine := fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, version+1))
	versionPlusOneID := getIDFromGlanceResponse(versionPlusOneLine)
	versionPlusTwoLine := fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, version+2))
	versionPlusTwoID := getIDFromGlanceResponse(versionPlusTwoLine)

	testCases := []struct {
		glanceOutput       string
		expectedImageNames []string
	}{
		{fmt.Sprintf(baseCompleteResponse, "", "", "", ""),
			[]string{}},
		{fmt.Sprintf(baseCompleteResponse, versionLine, "", "", ""),
			[]string{versionID}},
		{fmt.Sprintf(baseCompleteResponse, versionLine, versionPlusOneLine, "", ""),
			[]string{versionPlusOneID, versionID}},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionLine, "", ""),
			[]string{versionPlusOneID, versionID}},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionLine, "", versionPlusTwoLine),
			[]string{versionPlusTwoID, versionPlusOneID, versionID}},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionPlusTwoLine, versionLine, versionPlusOneLine),
			[]string{versionPlusTwoID, versionPlusOneID, versionPlusOneID, versionID}},
	}
	for _, item := range testCases {
		s.cli.output = item.glanceOutput
		imageList, _ := s.subject.GetVersions(s.defaultOptions)

		c.Check(testEq(imageList, item.expectedImageNames), check.Equals, true)
	}
}

func (s *cloudSuite) TestGetVersionsQueriesGlance(c *check.C) {
	_, err := s.subject.GetVersions(s.defaultOptions)

	c.Assert(err, check.IsNil)
	c.Assert(s.cli.execCommandCalls[imageListCmd], check.Equals, 1)
}

func (s *cloudSuite) TestGetVersionsReturnsGlanceError(c *check.C) {
	s.cli.err = true

	_, err := s.subject.GetVersions(s.defaultOptions)

	c.Assert(err, check.NotNil)
}

func (s *cloudSuite) TestGetLatestVersionsRemovesDotFromRelease(c *check.C) {
	version := 100
	s.defaultOptions.Release = "1604"
	versionLine := fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, version))
	s.cli.output = fmt.Sprintf(baseCompleteResponse, versionLine, "", "", "")
	list, _ := s.subject.GetVersions(s.defaultOptions)

	expected := getIDFromGlanceResponse(versionLine)

	c.Assert(testEq(list, []string{expected}), check.Equals, true)
}

func (s *cloudSuite) TestPurgeCallsCliForListing(c *check.C) {
	s.subject.Purge(s.defaultOptions)

	c.Assert(s.cli.execCommandCalls[imageListCmd], check.Equals, 1)
}

func (s *cloudSuite) TestPurgeReturnsListingError(c *check.C) {
	s.cli.err = true
	err := s.subject.Purge(s.defaultOptions)

	c.Assert(err, check.NotNil)
}

func (s *cloudSuite) TestPurgeCallsCliForDeleting(c *check.C) {
	version := 100
	versionLine := fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, version))
	versionID := getIDFromGlanceResponse(versionLine)
	s.defaultOptions.Release = testDefaultRelease + "-plusOneRelease"
	s.defaultOptions.OSChannel = testDefaultChannel + "-plusOneChannel"
	s.defaultOptions.KernelChannel = testDefaultChannel + "-plusOneChannel"
	s.defaultOptions.GadgetChannel = testDefaultChannel + "-plusOneChannel"
	versionPlusOneLine := fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, version+1))
	versionPlusOneID := getIDFromGlanceResponse(versionPlusOneLine)
	s.defaultOptions.Release = testDefaultRelease + "-plusTwoRelease"
	s.defaultOptions.OSChannel = testDefaultChannel + "-plusTwoChannel"
	s.defaultOptions.KernelChannel = testDefaultChannel + "-plusTwoChannel"
	s.defaultOptions.GadgetChannel = testDefaultChannel + "-plusTwoChannel"
	versionPlusTwoLine := fmt.Sprintf(baseResponse, getImageID(s.defaultOptions, version+2))
	versionPlusTwoID := getIDFromGlanceResponse(versionPlusTwoLine)

	testCases := []struct {
		glanceOutput       string
		expectedImageNames []string
	}{
		{fmt.Sprintf(baseCompleteResponse, "", "", "", ""),
			[]string{}},
		{fmt.Sprintf(baseCompleteResponse, versionLine, "", "", ""),
			[]string{versionID}},
		{fmt.Sprintf(baseCompleteResponse, versionLine, versionPlusOneLine, "", ""),
			[]string{versionID, versionPlusOneID}},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionLine, "", ""),
			[]string{versionPlusOneID, versionID}},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionLine, "", versionPlusTwoLine),
			[]string{versionPlusOneID, versionID, versionPlusTwoID}},
		{fmt.Sprintf(baseCompleteResponse, versionPlusOneLine, versionPlusTwoLine, versionLine, versionPlusOneLine),
			[]string{versionPlusOneID, versionPlusTwoID, versionID, versionPlusOneID}},
	}
	for _, item := range testCases {
		s.cli.output = item.glanceOutput

		s.subject.Purge(s.defaultOptions)

		expectedCall := strings.Join(append([]string{"openstack image delete"}, item.expectedImageNames...), " ")
		c.Check(s.cli.execCommandCalls[expectedCall], check.Equals, 1)
	}
}

func (s *cloudSuite) TestPurgeReturnsCliError(c *check.C) {
	s.cli.err = true

	imgID := getImageID(s.defaultOptions, testImageVersion)
	unexpectedCall := "openstack image delete " + imgID
	err := s.subject.Purge(s.defaultOptions)

	c.Assert(err, check.NotNil)
	c.Assert(s.cli.execCommandCalls[unexpectedCall], check.Equals, 0)
}

func (s *cloudSuite) TestExtractVersionsFromListDoNotModifyRelease(c *check.C) {
	expectedRelease := "15.04"
	s.defaultOptions.Release = expectedRelease

	s.subject.extractVersionsFromList(*s.defaultOptions)

	c.Assert(s.defaultOptions.Release, check.Equals, expectedRelease)
}

func getIDFromGlanceResponse(response string) string {
	// response is of the form:
	// | 762d5ce2-fbc2-4685-8d6c-71249d19df9e | ubuntu-core/custom/ubuntu-%s-snappy-core-%s-%s-%d-disk1.img                        |
	items := strings.Fields(response)
	return items[3]
}

func testEq(a, b []string) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func getImageID(options *flags.Options, ver int) string {
	return fmt.Sprintf("ubuntu-core/%s/ubuntu-%s-snappy-core-%s-%s-%d-disk1.img",
		options.ImageType, options.Release, options.Arch, options.OSChannel, ver)
}

func getVersionFromImageID(imageID string) string {
	parts := strings.Split(imageID, "-")
	return parts[7]
}

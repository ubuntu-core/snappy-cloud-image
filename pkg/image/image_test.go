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

package image

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ubuntu-core/snappy/progress"
	"github.com/ubuntu-core/snappy/snap/remote"
	"github.com/ubuntu-core/snappy/snappy"
	"gopkg.in/check.v1"

	"github.com/ubuntu-core/snappy-cloud-image/pkg/flags"
)

const (
	testDefaultRelease       = "rolling"
	testDefaultArch          = "amd64"
	testDefaultQcow2compat   = "1.1"
	testDefaultOS            = "myos"
	testDefaultKernel        = "mykernel"
	testDefaultGadget        = "mygadget"
	testDefaultVer           = 100
	testDefaultOSChannel     = "myoschannel"
	testDefaultKernelChannel = "mykernelchannel"
	testDefaultGadgetChannel = "mygadgetchannel"
	tmpDirName               = "tmpdirname"
)

var _ = check.Suite(&imageSuite{})

var (
	testSnaps    = []string{testDefaultOS, testDefaultKernel, testDefaultGadget}
	testChannels = []string{testDefaultOSChannel, testDefaultKernelChannel, testDefaultGadgetChannel}
)

func Test(t *testing.T) { check.TestingT(t) }

type imageSuite struct {
	subject        Driver
	cli            *fakeCliCommander
	storeClient    *fakeStoreClient
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

type fakeStoreClient struct {
	snapCalls                                map[string]int
	downloadCalls                            map[string]int
	totalSnapCalls, correctSnapCalls         int
	snapErr                                  bool
	totalDownloadCalls, correctDownloadCalls int
	downloadErr                              bool
}

func (f *fakeStoreClient) Download(remoteSnap *snappy.RemoteSnap, pb progress.Meter) (path string, err error) {
	f.downloadCalls[getDownloadCall(remoteSnap.Name(), remoteSnap.Channel())]++
	f.totalDownloadCalls++

	if f.downloadErr {
		if f.totalDownloadCalls > f.correctDownloadCalls {
			return "", errors.New("")
		}
	}
	return getSnapFilename(remoteSnap.Name(), remoteSnap.Channel()), nil
}

func (f *fakeStoreClient) Snap(name, channel string) (remoteSnap *snappy.RemoteSnap, err error) {
	f.snapCalls[getSnapCall(name, channel)]++

	f.totalSnapCalls++

	if f.snapErr {
		if f.totalSnapCalls > f.correctSnapCalls {
			return nil, errors.New("")
		}
	}

	return snappy.NewRemoteSnap(remote.Snap{Name: name, Channel: channel}), nil
}

func (s *imageSuite) SetUpSuite(c *check.C) {
	s.cli = &fakeCliCommander{}
	s.storeClient = &fakeStoreClient{}
	s.subject = NewUDFQcow2(s.cli, s.storeClient)
}

func (s *imageSuite) SetUpTest(c *check.C) {
	s.defaultOptions = &flags.Options{
		Release:       testDefaultRelease,
		Arch:          testDefaultArch,
		Qcow2compat:   testDefaultQcow2compat,
		OS:            testDefaultOS,
		Kernel:        testDefaultKernel,
		Gadget:        testDefaultGadget,
		OSChannel:     testDefaultOSChannel,
		KernelChannel: testDefaultKernelChannel,
		GadgetChannel: testDefaultGadgetChannel,
	}

	s.cli.execCommandCalls = make(map[string]int)
	s.cli.err = false
	s.cli.correctCalls = 0
	s.cli.totalCalls = 0
	s.cli.output = ""
	s.storeClient.snapCalls = make(map[string]int)
	s.storeClient.downloadCalls = make(map[string]int)
	s.storeClient.snapErr = false
	s.storeClient.correctSnapCalls = 0
	s.storeClient.totalSnapCalls = 0
	s.storeClient.downloadErr = false
	s.storeClient.correctDownloadCalls = 0
	s.storeClient.totalDownloadCalls = 0
}

func (s *imageSuite) TestCreateCallsUDF(c *check.C) {
	s.cli.output = tmpDirName
	filename := tmpRawFileName()
	testCases := []struct {
		release, arch, os, kernel, gadget, osChannel, gadgetChannel, kernelChannel string
		version                                                                    int
		expectedCall                                                               string
	}{
		{"16.04", "amd64", "os1", "kernel1", "gadget1", "oschan1", "gadgetchan1", "kernchan1", 100, "sudo ubuntu-device-flash core 16.04 --channel oschan1 --os os1 --kernel kernel1_kernchan1.snap --gadget gadget1_gadgetchan1.snap --developer-mode  -o " + filename},
		{"rolling", "amd64", "os2", "kernel2", "gadget2", "oschan2", "gadchan2", "kchan2", 100, "sudo ubuntu-device-flash core rolling --channel oschan2 --os os2 --kernel kernel2_kchan2.snap --gadget gadget2_gadchan2.snap --developer-mode  -o " + filename},
		{"17.10", "arm", "os3", "kernel3", "gadget3", "chanos3", "gadchan3", "kernchan3", 56, "sudo ubuntu-device-flash core 17.10 --channel chanos3 --os os3 --kernel kernel3_kernchan3.snap --gadget gadget3_gadchan3.snap --developer-mode --oem beagleblack -o " + filename},
	}

	for _, item := range testCases {
		s.cli.execCommandCalls = make(map[string]int)
		options := &flags.Options{
			Release:       item.release,
			Arch:          item.arch,
			Qcow2compat:   testDefaultQcow2compat,
			OS:            item.os,
			Kernel:        item.kernel,
			Gadget:        item.gadget,
			OSChannel:     item.osChannel,
			GadgetChannel: item.gadgetChannel,
			KernelChannel: item.kernelChannel,
		}
		_, err := s.subject.Create(options, item.version)

		c.Check(err, check.IsNil)

		c.Assert(len(s.cli.execCommandCalls) > 0, check.Equals, true)
		c.Check(s.cli.execCommandCalls[item.expectedCall], check.Equals, 1)
	}
}

func (s *imageSuite) TestCreateCallsUDFWithoutRevisionForNon1504(c *check.C) {
	s.cli.output = tmpDirName
	filename := tmpRawFileName()

	expectedCall := fmt.Sprintf("sudo ubuntu-device-flash core %s --channel %s --os %s --kernel %s_%s.snap --gadget %s_%s.snap --developer-mode  -o "+filename,
		s.defaultOptions.Release, s.defaultOptions.OSChannel, s.defaultOptions.OS, s.defaultOptions.Kernel, s.defaultOptions.KernelChannel, s.defaultOptions.Gadget, s.defaultOptions.GadgetChannel)

	_, err := s.subject.Create(s.defaultOptions, testDefaultVer)

	c.Check(err, check.IsNil)
	c.Assert(len(s.cli.execCommandCalls) > 0, check.Equals, true)
	c.Check(s.cli.execCommandCalls[expectedCall], check.Equals, 1)
}

func (s *imageSuite) TestCreateCallsUDFWithoutAllSnapsParamsFor1504(c *check.C) {
	s.cli.output = tmpDirName
	filename := tmpRawFileName()

	version := 56
	release := "15.04"

	expectedCall := fmt.Sprintf("sudo ubuntu-device-flash --revision=%d core %s --channel %s --developer-mode  -o %s",
		version, release, testDefaultOSChannel, filename)

	s.defaultOptions.Release = release

	_, err := s.subject.Create(s.defaultOptions, version)

	c.Check(err, check.IsNil)
	c.Assert(len(s.cli.execCommandCalls) > 0, check.Equals, true)

	c.Check(s.cli.execCommandCalls[expectedCall], check.Equals, 1)
}

func (s *imageSuite) TestCreateDoesNotCallUDFOnMktempError(c *check.C) {
	s.cli.err = true
	s.cli.output = tmpDirName
	filename := tmpRawFileName()

	s.subject.Create(s.defaultOptions, testDefaultVer)

	expectedCall := fmt.Sprintf("sudo ubuntu-device-flash core %s --channel %s --os %s --kernel %s --gadget %s --developer-mode  -o %s",
		testDefaultRelease, testDefaultOSChannel, testDefaultOS, testDefaultKernel, testDefaultGadget, filename)

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

func (s *imageSuite) TestCreateCallsStoreSnapForEachSnap(c *check.C) {
	s.subject.Create(s.defaultOptions, testDefaultVer)

	for i := 1; i < len(testSnaps); i++ {
		c.Check(s.storeClient.snapCalls[getSnapCall(testSnaps[i], testChannels[i])],
			check.Equals, 1)
	}
}

func (s *imageSuite) TestCreateCallsStoreDownloadForEachSnap(c *check.C) {
	s.subject.Create(s.defaultOptions, testDefaultVer)

	for i := 1; i < len(testSnaps); i++ {
		c.Check(s.storeClient.downloadCalls[getDownloadCall(testSnaps[i], testChannels[i])],
			check.Equals, 1)
	}
}

func (s *imageSuite) TestCreateReturnsStoreSnapErrorForEachSnap(c *check.C) {
	s.storeClient.snapErr = true

	for i := 1; i < len(testSnaps); i++ {
		s.storeClient.totalSnapCalls = 0
		s.storeClient.correctSnapCalls = i - 1

		_, err := s.subject.Create(s.defaultOptions, testDefaultVer)

		c.Assert(err, check.NotNil)
		c.Check(err, check.FitsTypeOf, &ErrRepoDetail{})
		c.Check(err.Error(), check.Equals, fmt.Sprintf(errRepoDetailFmt, testSnaps[i], "", testChannels[i]))
	}
}

func (s *imageSuite) TestCreateReturnsStoreDownloadErrorForEachSnap(c *check.C) {
	s.storeClient.downloadErr = true

	for i := 1; i < len(testSnaps); i++ {
		s.storeClient.totalDownloadCalls = 0
		s.storeClient.correctDownloadCalls = i - 1

		_, err := s.subject.Create(s.defaultOptions, testDefaultVer)

		c.Assert(err, check.NotNil)
		c.Check(err, check.FitsTypeOf, &ErrRepoDownload{})
		c.Check(err.Error(), check.Equals, fmt.Sprintf(errRepoDownloadFmt, testSnaps[i], "", testChannels[i]))
	}
}

func (s *imageSuite) TestCreateSetsChannelWhenAllSnapChannelsAreEqual(c *check.C) {
	commonChannel := "commonChannel"

	s.defaultOptions.OSChannel = commonChannel
	s.defaultOptions.KernelChannel = commonChannel
	s.defaultOptions.GadgetChannel = commonChannel

	s.cli.output = tmpDirName
	filename := tmpRawFileName()

	expectedCall := fmt.Sprintf("sudo ubuntu-device-flash core %s --channel %s --os %s --kernel %s --gadget %s --developer-mode  -o "+filename,
		s.defaultOptions.Release, commonChannel, s.defaultOptions.OS, s.defaultOptions.Kernel, s.defaultOptions.Gadget)

	_, err := s.subject.Create(s.defaultOptions, testDefaultVer)

	c.Check(err, check.IsNil)
	c.Assert(len(s.cli.execCommandCalls) > 0, check.Equals, true)
	c.Check(s.cli.execCommandCalls[expectedCall], check.Equals, 1)
}

func (s *imageSuite) TestCreateSetsChannelWhenTwoSnapChannelsAreEqual(c *check.C) {
	commonChannel := "commonChannel"
	anotherChannel := "anotherchannel"
	s.defaultOptions.OSChannel = anotherChannel
	s.defaultOptions.KernelChannel = commonChannel
	s.defaultOptions.GadgetChannel = commonChannel

	s.cli.output = tmpDirName
	filename := tmpRawFileName()

	expectedCall := fmt.Sprintf("sudo ubuntu-device-flash core %s --channel %s --os %s_%s.snap --kernel %s --gadget %s --developer-mode  -o "+filename,
		s.defaultOptions.Release, commonChannel, s.defaultOptions.OS, anotherChannel, s.defaultOptions.Kernel, s.defaultOptions.Gadget)

	_, err := s.subject.Create(s.defaultOptions, testDefaultVer)

	c.Check(err, check.IsNil)
	c.Assert(len(s.cli.execCommandCalls) > 0, check.Equals, true)
	c.Check(s.cli.execCommandCalls[expectedCall], check.Equals, 1)
}

func (s *imageSuite) TestCreateSetsChannelWhenAllSnapChannelsAreDifferent(c *check.C) {
	s.cli.output = tmpDirName
	filename := tmpRawFileName()

	expectedCall := fmt.Sprintf("sudo ubuntu-device-flash core %s --channel %s --os %s --kernel %s_%s.snap --gadget %s_%s.snap --developer-mode  -o "+filename,
		s.defaultOptions.Release, s.defaultOptions.OSChannel, s.defaultOptions.OS, s.defaultOptions.Kernel, s.defaultOptions.KernelChannel, s.defaultOptions.Gadget, s.defaultOptions.GadgetChannel)

	_, err := s.subject.Create(s.defaultOptions, testDefaultVer)

	c.Check(err, check.IsNil)
	c.Assert(len(s.cli.execCommandCalls) > 0, check.Equals, true)
	c.Check(s.cli.execCommandCalls[expectedCall], check.Equals, 1)
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

func getSnapCall(name, channel string) string {
	return fmt.Sprintf("%s - %s", name, channel)
}

func getSnapFilename(name, channel string) string {
	return fmt.Sprintf("%s_%s.snap", name, channel)
}

func getDownloadCall(name, channel string) string {
	return fmt.Sprintf("%s - %s", name, channel)
}

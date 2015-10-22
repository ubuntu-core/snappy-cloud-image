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

package si

import (
	"fmt"
	"testing"

	"gopkg.in/check.v1"

	"github.com/fgimenez/snappy-cloud-image/pkg/web"
)

var (
	_                 = check.Suite(&siSuite{})
	validJSONImage    = fmt.Sprintf(imageBase, "full", testImageVersion)
	validJSONResponse = fmt.Sprintf(responseBase, validJSONImage)
)

const (
	testDefaultRelease = "rolling"
	testDefaultChannel = "edge"
	testDefaultArch    = "amd64"
	testImageVersion   = 198
	responseBase       = `{
    "global": {
        "generated_at": "Wed Oct 21 06:11:44 UTC 2015"
    },
    "images": [
        %s
    ]
}`
	imageBase = `{
            "description": "ubuntu=20150831,raw-device=20150831,version=157",
            "files": [
                {
                    "checksum": "600338ea231bfca085bc2e9cfad116e83fd1cbc31b31095475ce8010da2d67d9",
                    "order": 0,
                    "path": "/pool/ubuntu-3a12e5dbfa64a87fc47d7cef47e4ecee487686ef1cef280602faee4fb805d187.tar.xz",
                    "signature": "/pool/ubuntu-3a12e5dbfa64a87fc47d7cef47e4ecee487686ef1cef280602faee4fb805d187.tar.xz.asc",
                    "size": 50434700
                },
                {
                    "checksum": "970b64173315c0723b8ca64d5ee3f5afc7ee758c94936ce6f5524780609c6727",
                    "order": 1,
                    "path": "/pool/device-2ca4079697ab7806e8154d9aa4991cfd7e6825db59ef43a6f535a720fc72eddd.tar.xz",
                    "signature": "/pool/device-2ca4079697ab7806e8154d9aa4991cfd7e6825db59ef43a6f535a720fc72eddd.tar.xz.asc",
                    "size": 88374184
                },
                {
                    "checksum": "0fd13110a11d439da22080fce91467b2aac80074073938dcba59d862dda66326",
                    "order": 2,
                    "path": "/ubuntu-core/rolling/edge/generic_amd64/version-157.tar.xz",
                    "signature": "/ubuntu-core/rolling/edge/generic_amd64/version-157.tar.xz.asc",
                    "size": 432
                }
            ],
            "type": "%s",
            "version": %d,
            "version_detail": "ubuntu=20150831,raw-device=20150831,version=157"
        }`
)

type siSuite struct {
	subject   *Client
	webGetter *fakeWebGetter
}

func Test(t *testing.T) { check.TestingT(t) }

type fakeWebGetter struct {
	calls  map[string]int
	error  bool
	output []byte
}

func (w *fakeWebGetter) Get(url string) (output []byte, err error) {
	w.calls[url]++
	if w.error {
		err = &web.ErrHTTPGet{}
	}
	return w.output, err
}

func (s *siSuite) SetUpSuite(c *check.C) {
	s.webGetter = &fakeWebGetter{}
	s.subject = &Client{httpClient: s.webGetter}
}

func (s *siSuite) SetUpTest(c *check.C) {
	s.webGetter.calls = make(map[string]int)
	s.webGetter.error = false
	s.webGetter.output = []byte(validJSONResponse)
}

func (s *siSuite) TestGetLatestVersionQueriesTheRightUrl(c *check.C) {
	testCases := []struct {
		release, channel, arch, expected string
	}{
		{"15.04", "alpha", "amd64", baseURL + "/15.04/alpha/generic_amd64/" + dataFileName},
		{"15.04", "alpha", "arm", baseURL + "/15.04/alpha/generic_armhf/" + dataFileName},
		{"15.04", "edge", "amd64", baseURL + "/15.04/edge/generic_amd64/" + dataFileName},
		{"15.04", "edge", "arm", baseURL + "/15.04/edge/generic_armhf/" + dataFileName},
		{"15.04", "stable", "amd64", baseURL + "/15.04/stable/generic_amd64/" + dataFileName},
		{"15.04", "stable", "arm", baseURL + "/15.04/stable/generic_armhf/" + dataFileName},
		{"rolling", "alpha", "amd64", baseURL + "/rolling/alpha/generic_amd64/" + dataFileName},
		{"rolling", "alpha", "arm", baseURL + "/rolling/alpha/generic_armhf/" + dataFileName},
		{"rolling", "edge", "amd64", baseURL + "/rolling/edge/generic_amd64/" + dataFileName},
		{"rolling", "edge", "arm", baseURL + "/rolling/edge/generic_armhf/" + dataFileName},
		{"rolling", "stable", "amd64", baseURL + "/rolling/stable/generic_amd64/" + dataFileName},
		{"rolling", "stable", "arm", baseURL + "/rolling/stable/generic_armhf/" + dataFileName},
	}
	for _, item := range testCases {
		_, err := s.subject.GetLatestVersion(item.release, item.channel, item.arch)

		c.Check(err, check.IsNil)
		c.Check(s.webGetter.calls[item.expected], check.Equals, 1)
	}
}

func (s *siSuite) TestGetLatestVersionReturnshttpGetterError(c *check.C) {
	s.webGetter.error = true

	_, err := s.subject.GetLatestVersion(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(err, check.NotNil)
	c.Assert(err, check.FitsTypeOf, &web.ErrHTTPGet{})
}

func (s *siSuite) TestGetLatestVersionParsesJsonResponse(c *check.C) {
	output, err := s.subject.GetLatestVersion(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(err, check.IsNil)
	c.Assert(output, check.Equals, testImageVersion)
}

func (s *siSuite) TestGetLatestVersionGetsTheLatestVersion(c *check.C) {
	image1 := fmt.Sprintf(imageBase, "full", testImageVersion-1)
	image2 := fmt.Sprintf(imageBase, "full", testImageVersion)
	images := image1 + "," + image2

	response := fmt.Sprintf(responseBase, images)

	s.webGetter.output = []byte(response)

	output, err := s.subject.GetLatestVersion(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(err, check.IsNil)
	c.Assert(output, check.Equals, testImageVersion)
}

func (s *siSuite) TestGetLatestVersionHonoursDeltas(c *check.C) {
	image1 := fmt.Sprintf(imageBase, "full", testImageVersion)
	image2 := fmt.Sprintf(imageBase, "delta", testImageVersion+1)
	images := image1 + "," + image2

	response := fmt.Sprintf(responseBase, images)

	s.webGetter.output = []byte(response)

	output, err := s.subject.GetLatestVersion(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(err, check.IsNil)
	c.Assert(output, check.Equals, testImageVersion)
}

func (s *siSuite) TestGetLatestVersionReturnsUnmarshalError(c *check.C) {
	s.webGetter.output = []byte("{{Not a valid JSON 'string']")

	_, err := s.subject.GetLatestVersion(testDefaultRelease, testDefaultChannel, testDefaultArch)

	c.Assert(err, check.NotNil)
}

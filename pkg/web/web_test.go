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

package web

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"gopkg.in/check.v1"
)

const (
	testURL        = "http://example.com"
	httpErrMsg     = "http.Get failed"
	responseString = "This is the response string"
	bodyReadErrMsg = "read of resp.Body failed"
)

var _ = check.Suite(&webSuite{})

type fakeBody struct {
	bytes.Buffer
	readError  bool
	closeCalls int
}

type webSuite struct {
	subject      *Client
	backHTTPGet  func(string) (resp *http.Response, err error)
	httpGetCalls map[string]int
	httpGetError bool

	body *fakeBody
}

func (b *fakeBody) Read(buf []byte) (n int, err error) {
	if b.readError {
		return 0, fmt.Errorf(bodyReadErrMsg)
	}
	return b.Buffer.Read(buf)
}

func (b *fakeBody) Close() (err error) {
	b.closeCalls++
	return
}

func Test(t *testing.T) { check.TestingT(t) }

func (s *webSuite) SetUpSuite(c *check.C) {
	s.subject = &Client{}
	s.body = &fakeBody{}
	s.backHTTPGet = httpGet
	httpGet = s.fakeHTTPGet
}

func (s *webSuite) TearDownSuite(c *check.C) {
	httpGet = s.backHTTPGet
}

func (s *webSuite) SetUpTest(c *check.C) {
	s.httpGetCalls = make(map[string]int)
	s.httpGetError = false
	s.body.Reset()
	s.body.readError = false
	s.body.closeCalls = 0
}

func (s *webSuite) fakeHTTPGet(url string) (resp *http.Response, err error) {
	s.httpGetCalls[url]++
	resp = &http.Response{Body: s.body}
	if s.httpGetError {
		return resp, fmt.Errorf(httpErrMsg)
	}
	return resp, nil
}

func (s *webSuite) TestGetCallsHttpGet(c *check.C) {
	s.subject.Get(testURL)

	c.Assert(s.httpGetCalls[testURL], check.Equals, 1)
}

func (s *webSuite) TestGetReturnsHttpGetErrors(c *check.C) {
	s.httpGetError = true
	_, err := s.subject.Get(testURL)

	c.Assert(err, check.FitsTypeOf, &ErrHTTPGet{})
	c.Assert(err.Error(), check.Equals, httpErrMsg)
}

func (s *webSuite) TestGetReturnsHttpGetBody(c *check.C) {
	s.body.Write([]byte(responseString))

	content, err := s.subject.Get(testURL)

	c.Assert(err, check.IsNil)
	c.Assert(content, check.Equals, responseString)
}

func (s *webSuite) TestGetReturnsBodyReadErrors(c *check.C) {
	s.body.readError = true
	_, err := s.subject.Get(testURL)

	c.Assert(err, check.FitsTypeOf, &ErrBodyRead{})
	c.Assert(err.Error(), check.Equals, bodyReadErrMsg)
}

func (s *webSuite) TestGetCallsBodyClose(c *check.C) {
	s.subject.Get(testURL)

	c.Assert(s.body.closeCalls, check.Equals, 1)
}

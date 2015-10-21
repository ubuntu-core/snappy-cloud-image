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

// Package web handles the needed web connections
package web

import (
	"io/ioutil"
	"net/http"
)

var (
	httpGet = http.Get
)

// Getter has the generic Get method for retrieving the contents of an url
type Getter interface {
	Get(string) (content string, err error)
}

// Client is the default web client
type Client struct{}

// ErrHTTPGet is the type of the errors returned when http.Get fails
type ErrHTTPGet struct {
	msg string
}

// ErrBodyRead is the type of the errors returned when reading resp.Body fails
type ErrBodyRead struct {
	msg string
}

// Get retrieves the contents of the given url and return them as a string, with
// the eventual errors in the process
func (c *Client) Get(url string) (content string, err error) {
	resp, err := httpGet(url)
	defer resp.Body.Close()
	if err != nil {
		return "", &ErrHTTPGet{msg: err.Error()}
	}

	bcontent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", &ErrBodyRead{msg: err.Error()}
	}

	return string(bcontent), err
}

func (e *ErrHTTPGet) Error() string {
	return e.msg
}

func (e *ErrBodyRead) Error() string {
	return e.msg
}

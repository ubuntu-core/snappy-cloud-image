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

// Package cloud manages the interaction with the cloud provider, currently only
// OpenStack is supported. It knows how to query the highest published version
// of the snappy image for a given release and channel and to upload new images
package cloud

import (
	"bufio"
	"fmt"
	"sort"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/ubuntu-core/snappy-cloud-image/pkg/cli"
)

const (
	baseImageName          = "ubuntu-core/custom/ubuntu-"
	imageNamePrefixPattern = baseImageName + "%s-snappy-core-%s-%s"
	imageNameSufix         = "disk1.img"
	errVerNotFoundPattern  = "Version not found for release %s, channel %s and arch %s"
	imageListCmd           = "openstack image list --property status=active"
)

// Client is the implementation of Clouder that interacts with the provider
type Client struct {
	cli cli.Commander
}

// NewClient is the Client constructor
func NewClient(cli cli.Commander) *Client {
	return &Client{cli}
}

// ErrVersionNotFound is the type error returned when there are no images for a given
// release, channel and arch
type ErrVersionNotFound struct{ release, channel, arch string }

// NewErrVersionNotFound is th ErrVersionNotFound constructor
func NewErrVersionNotFound(release, channel, arch string) *ErrVersionNotFound {
	return &ErrVersionNotFound{release: release, channel: channel, arch: arch}
}

func (e *ErrVersionNotFound) Error() string {
	return fmt.Sprintf(errVerNotFoundPattern, e.release, e.channel, e.arch)
}

// GetLatestVersion returns the highest version of the custom images for the given
// release, channel and arch, -1 if none is found, and the eventual error
func (c *Client) GetLatestVersion(release, channel, arch string) (ver int, err error) {
	imageIDs, err := c.extractVersionsFromList(release, channel, arch)
	if err != nil {
		return 0, err
	}
	version, err := extractVersion(imageIDs[0])
	if err != nil {
		return 0, err
	}
	return version, nil
}

// Create makes the call to create the new image given a file path with the local image
// and the required bits for making up the image name
func (c *Client) Create(path, release, channel, arch string, version int) (err error) {
	imageID := GetImageID(release, channel, arch, version)

	log.Debugf("Creating image %s from file %s", imageID, path)

	_, err = c.cli.ExecCommand("openstack", "image", "create", "--disk-format", "qcow2", "--file", path, imageID)
	return
}

// extractVersionsFromList returns a list of image names that match the given
// release, channel and arch sorted in descendant version number order
func (c *Client) extractVersionsFromList(release, channel, arch string) ([]string, error) {
	release = removeDot(release)
	var imageIDs sort.StringSlice
	imageIDs, err := c.getImageList(imgTemplate(release, channel, arch))
	if err != nil {
		return imageIDs, err
	}
	if len(imageIDs) > 0 {
		sort.Sort(sort.Reverse(imageIDs[:]))
		return imageIDs, nil
	}
	return []string{}, NewErrVersionNotFound(release, channel, arch)
}

// getImageList returns a list of image IDs that match a given pattern
func (c *Client) getImageList(pattern string) (imagelist []string, err error) {
	/* list is of the form:
	| 08763be0-3b3d-41e3-b5b0-08b9006fc1d7 | smoser-lucid-loader/lucid-amd64-linux-image-2.6.32-34-virtual-v-2.6.32-34.77~smloader0-build0-loader |
	| 842949c6-225b-4ad0-81b7-98de2b818eed | smoser-lucid-loader/lucid-amd64-linux-image-2.6.32-34-virtual-v-2.6.32-34.77~smloader0-kernel        |
	| 762d5ce2-fbc2-4685-8d6c-71249d19df9e | ubuntu-core/custom/ubuntu-1504-snappy-core-amd64-edge-202-disk1.img                                  |
	*/
	list, err := c.cli.ExecCommand(strings.Fields(imageListCmd)...)
	if err != nil {
		return []string{}, err
	}

	reader := strings.NewReader(list)
	scanner := bufio.NewScanner(reader)

	var imageIDs []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, pattern) {
			fields := strings.Fields(line)
			imageIDs = append(imageIDs, fields[3])
		}
	}
	return imageIDs, nil
}

func imgTemplate(release, channel, arch string) (pattern string) {
	return fmt.Sprintf(imageNamePrefixPattern, release, arch, channel)
}

// Returns the version contained in imageID, which is of the form:
// ubuntu-core/custom/ubuntu-rolling-snappy-core-amd64-edge-100-disk1.img,
// in this case it should return 100
func extractVersion(imageID string) (ver int, err error) {
	parts := strings.Split(imageID, "-")
	return strconv.Atoi(parts[7])
}

// GetImageID returns the image name for the given parameters
func GetImageID(release, channel, arch string, version int) (name string) {
	release = removeDot(release)
	imageNamePrefix := fmt.Sprintf(imageNamePrefixPattern, release, arch, channel)
	return fmt.Sprintf("%s-%d-%s", imageNamePrefix, version, imageNameSufix)
}

// Delete calls the cli command to remove the given images
func (c *Client) Delete(images ...string) (err error) {
	_, err = c.cli.ExecCommand(append([]string{"openstack", "image", "delete"}, images...)...)
	return
}

// GetVersions returns a descending ordered list (newer first) of image names for the given parameters
func (c *Client) GetVersions(release, channel, arch string) (imageNames []string, err error) {
	return c.extractVersionsFromList(release, channel, arch)
}

// Purge asks the glance endpoint to remove all the custom images present.
// Use with care! For example it can be useful when deploying a new jenkins,
// the instances from images created with the previous one won't be accessible
// any more
func (c *Client) Purge() error {
	images, err := c.getImageList(baseImageName)
	if err != nil {
		return err
	}
	return c.Delete(images...)
}

func removeDot(in string) string {
	return strings.Replace(in, ".", "", 1)
}

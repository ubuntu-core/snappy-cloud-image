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
package flags

import (
	"flag"
	"os"
	"testing"

	"gopkg.in/check.v1"
)

var _ = check.Suite(&flagsSuite{})

type flagsSuite struct {
	backOsArgs []string
}

func Test(t *testing.T) { check.TestingT(t) }

func (s *flagsSuite) SetUpTest(c *check.C) {
	s.backOsArgs = os.Args
	resetFlag(func() {})
}

func (s *flagsSuite) TearDownTest(c *check.C) {
	os.Args = s.backOsArgs
}

func (s *flagsSuite) TestParseReturnsParsedFlags(c *check.C) {
	parsedFlags := Parse()

	c.Assert(parsedFlags, check.FitsTypeOf, &Options{})
}

func (s *flagsSuite) TestParseDefaultAction(c *check.C) {
	parsedFlags := Parse()

	c.Assert(parsedFlags.Action, check.Equals, defaultAction)
}

func (s *flagsSuite) TestParseDefaultRelease(c *check.C) {
	parsedFlags := Parse()

	c.Assert(parsedFlags.Release, check.Equals, defaultRelease)
}

func (s *flagsSuite) TestParseDefaultChannel(c *check.C) {
	parsedFlags := Parse()

	c.Assert(parsedFlags.Channel, check.Equals, defaultChannel)
}

func (s *flagsSuite) TestParseDefaultArch(c *check.C) {
	parsedFlags := Parse()

	c.Assert(parsedFlags.Arch, check.Equals, defaultArch)
}

func (s *flagsSuite) TestParseSetsActionToFlagValue(c *check.C) {
	os.Args = []string{"", "-action", "myaction"}
	parsedFlags := Parse()

	c.Assert(parsedFlags.Action, check.Equals, "myaction")
}

func (s *flagsSuite) TestParseSetsReleaseToFlagValue(c *check.C) {
	os.Args = []string{"", "-release", "myrelease"}
	parsedFlags := Parse()

	c.Assert(parsedFlags.Release, check.Equals, "myrelease")
}

func (s *flagsSuite) TestParseSetsChannelToFlagValue(c *check.C) {
	os.Args = []string{"", "-channel", "mychannel"}
	parsedFlags := Parse()

	c.Assert(parsedFlags.Channel, check.Equals, "mychannel")
}

func (s *flagsSuite) TestParseSetsArchToFlagValue(c *check.C) {
	os.Args = []string{"", "-arch", "myarch"}
	parsedFlags := Parse()

	c.Assert(parsedFlags.Arch, check.Equals, "myarch")
}

// from flag.ResetForTesting
func resetFlag(usage func()) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.Usage = usage
}

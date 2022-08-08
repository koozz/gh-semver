// Copyright 2022 Jan van den Berg
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type SemVer struct {
	Prefix   string
	LeadingV string
	Major    uint64
	Minor    uint64
	Patch    uint64
	Ext      *SemVerExtended
}

type SemVerExtended struct {
	Branch         string
	CommitDistance uint64
	CommitHash     string
}

var branchStripCharacters = regexp.MustCompile(`[^0-9A-Za-z]`)

func NewSemVer(major, minor, patch uint64) *SemVer {
	return &SemVer{
		Prefix:   "",
		LeadingV: "",
		Major:    major,
		Minor:    minor,
		Patch:    patch,
		Ext:      nil,
	}
}

func ParseSemVer(input string) (*SemVer, error) {
	re, err := regexp.Compile(`(?P<prefix>.+-)?(?P<v>v)?(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?P<extended>-(?P<branch>\w+)\.(?P<commit_distance>\d+)\.(?P<commit_hash>\w+))?`)
	if err != nil {
		return nil, err
	}

	semver := NewSemVer(0, 0, 0)
	matches := re.FindStringSubmatch(input)

	if matches[re.SubexpIndex("v")] == "v" {
		semver.LeadingV = "v"
	}

	major, err := strconv.ParseUint(matches[re.SubexpIndex("major")], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("error parsing major; %v", err)
	}
	semver.Major = major

	minor, err := strconv.ParseUint(matches[re.SubexpIndex("minor")], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("error parsing minor; %v", err)
	}
	semver.Minor = minor

	patch, err := strconv.ParseUint(matches[re.SubexpIndex("patch")], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("error parsing patch; %v", err)
	}
	semver.Patch = patch

	if matches[re.SubexpIndex("extended")] != "" {
		branch := matches[re.SubexpIndex("branch")]
		commitDistance, err := strconv.ParseUint(matches[re.SubexpIndex("commit_distance")], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("error parsing commit distance; %v", err)
		}
		commitHash := matches[re.SubexpIndex("commit_hash")]

		semver.Ext = &SemVerExtended{branch, commitDistance, commitHash}
	}
	return semver, nil
}

func (s *SemVer) GreaterThan(other *SemVer) bool {
	return s.Major > other.Major ||
		(s.Major == other.Major && s.Minor > other.Minor) ||
		(s.Major == other.Major && s.Minor == other.Minor && s.Patch > other.Patch)
}

func (s *SemVer) SameBranch(other *SemVer) bool {
	return s.Ext != nil && other.Ext != nil && s.Ext.Branch == other.Ext.Branch
}

func (s *SemVer) IncMajor() SemVer {
	return SemVer{
		Prefix:   s.Prefix,
		LeadingV: s.LeadingV,
		Major:    s.Major + 1,
		Minor:    0,
		Patch:    0,
		Ext:      s.Ext,
	}
}

func (s *SemVer) IncMinor() SemVer {
	return SemVer{
		Prefix:   s.Prefix,
		LeadingV: s.LeadingV,
		Major:    s.Major,
		Minor:    s.Minor + 1,
		Patch:    0,
		Ext:      s.Ext,
	}
}

func (s *SemVer) IncPatch() SemVer {
	return SemVer{
		Prefix:   s.Prefix,
		LeadingV: s.LeadingV,
		Major:    s.Major,
		Minor:    s.Minor,
		Patch:    s.Patch + 1,
		Ext:      s.Ext,
	}
}

func (s *SemVer) SetBranch(branch string) SemVer {
	if s.Ext == nil {
		s.Ext = &SemVerExtended{"", 0, ""}
	}
	s.Ext.Branch = branch

	return *s
}

func (s *SemVer) SetCommitDistance(commitDistance uint64) SemVer {
	if s.Ext == nil {
		s.Ext = &SemVerExtended{"", 0, ""}
	}
	s.Ext.CommitDistance = commitDistance

	return *s
}

func (s *SemVer) SetCommitHash(commitHash string) SemVer {
	if s.Ext == nil {
		s.Ext = &SemVerExtended{"", 0, ""}
	}
	if len(commitHash) >= 7 {
		s.Ext.CommitHash = commitHash[0:7]
	} else {
		s.Ext.CommitHash = commitHash
	}

	return *s
}

func (s *SemVer) Print(release bool) string {
	var version string
	if release || s.Ext == nil {
		version = fmt.Sprintf("%s%d.%d.%d", s.LeadingV, s.Major, s.Minor, s.Patch)
	} else {
		branch := branchStripCharacters.ReplaceAllString(s.Ext.Branch, "")
		version = fmt.Sprintf("%s%d.%d.%d-%s.%d.%s", s.LeadingV, s.Major, s.Minor, s.Patch, branch, s.Ext.CommitDistance, s.Ext.CommitHash)
	}
	if s.Prefix != "" {
		return strings.Join([]string{s.Prefix, version}, "-")
	} else {
		return version
	}
}

// Copyright 2021 Scott Leggett (https://github.com/smlx/ccv)
// Copyright 2022 Jan van den Berg
//
//	modifications
//	- added VersionBump struct
//	- changed to own SemVer struct
//	- added extended information (if not on main branch)
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
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type ConventionalCommits struct {
	majorRegex *regexp.Regexp
	minorRegex *regexp.Regexp
	patchRegex *regexp.Regexp
	// filterPath string
	// prefix     string
}

type VersionBump struct {
	major bool
	minor bool
	patch bool
}

func NewConventionalCommits( /*filterPath, prefix string*/ ) *ConventionalCommits {
	return &ConventionalCommits{
		majorRegex: regexp.MustCompile(`^(fix|feat)(\(.+\))?!: |BREAKING CHANGE: `),
		minorRegex: regexp.MustCompile(`^feat(\(.+\))?: `),
		patchRegex: regexp.MustCompile(`^fix(\(.+\))?: `),
		// filterPath: filterPath,
		// prefix:     prefix,
	}
}

// SemVer returns the calculated next semantic version
func (cc *ConventionalCommits) SemVer() (*SemVer, error) {
	// open current repository
	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("couldn't open git repository: %w", err)
	}

	tags, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("couldn't get tags: %w", err)
	}

	// map tags to commit hashes
	tagRefs := map[string]string{}
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		tagRefs[ref.Hash().String()] = ref.Name().Short()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't iterate tags: %w", err)
	}

	// no existing tags
	if len(tagRefs) == 0 {
		return NewSemVer(0, 1, 0), nil
	}

	// traverse main branch to find latest version
	latestMain, mainVersionBump, err := cc.traverse(repo, tagRefs, git.LogOrderDFS)
	if err != nil {
		return nil, fmt.Errorf("couldn't walk commits on main: %w", err)
	}
	mainBranch, err := cc.getMainBranch(repo)
	if err != nil {
		return nil, fmt.Errorf("couldn't figure out main branch: %w", err)
	}
	latestMain.SetBranch(mainBranch)

	// traverse current branch to find latest version
	latestBranch, branchVersionBump, err := cc.traverse(repo, tagRefs, git.LogOrderDFSPost)
	if err != nil {
		return nil, fmt.Errorf("couldn't walk commits on branch: %w", err)
	}
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("couldn't get head: %w", err)
	}
	latestBranch.SetBranch(head.Name().Short())

	// might be in detached head state
	if latestMain == nil && latestBranch == nil {
		return nil, fmt.Errorf("tags exist in the repository, but not in ancestors of HEAD")
	}

	// figure out the latest version in either parent
	var latestVersion *SemVer
	if latestMain == nil {
		latestVersion = latestBranch
	} else if latestBranch == nil {
		latestVersion = latestMain
	} else if latestMain.GreaterThan(latestBranch) {
		latestVersion = latestMain
	} else {
		latestVersion = latestBranch
	}

	// figure out the highest increment in either parent
	var newVersion SemVer
	switch {
	case mainVersionBump.major || branchVersionBump.major:
		newVersion = latestVersion.IncMajor()
	case mainVersionBump.minor || branchVersionBump.minor:
		newVersion = latestVersion.IncMinor()
	case mainVersionBump.patch || branchVersionBump.patch:
		newVersion = latestVersion.IncPatch()
	default:
		newVersion = *latestVersion
	}

	// drop extended information for main branch
	if latestBranch.SameBranch(latestMain) {
		newVersion.Ext = nil
	}
	return &newVersion, nil
}

func (cc *ConventionalCommits) traverse(repo *git.Repository, tagRefs map[string]string, order git.LogOrder) (*SemVer, *VersionBump, error) {
	versionBump := &VersionBump{}

	var stopIter error = fmt.Errorf("stop commit iteration")
	var latestTag string

	var commitDistance uint64 = 0
	var commitHash string = ""

	// walk commit hashes back from HEAD via main
	commits, err := repo.Log(&git.LogOptions{Order: order})
	if err != nil {
		return nil, versionBump, fmt.Errorf("couldn't get commits: %w", err)
	}

	err = commits.ForEach(func(commit *object.Commit) error {
		if commitHash == "" {
			commitHash = commit.Hash.String()
		}

		if latestTag = tagRefs[commit.Hash.String()]; latestTag != "" {
			return stopIter
		}
		commitDistance += 1

		// analyze commit message
		if cc.patchRegex.MatchString(commit.Message) {
			versionBump.patch = true
		}
		if cc.minorRegex.MatchString(commit.Message) {
			versionBump.minor = true
		}
		if cc.majorRegex.MatchString(commit.Message) {
			versionBump.major = true
		}
		return nil
	})
	if err != nil && err != stopIter {
		return nil, versionBump, fmt.Errorf("couldn't determine latest tag: %w", err)
	}

	// not tagged yet. this can happen if we are on a branch with no tags.
	if latestTag == "" {
		return nil, versionBump, nil
	}

	// parse
	latestVersion, err := ParseSemVer(latestTag)
	if err != nil {
		return nil, versionBump, fmt.Errorf("couldn't parse tag '%v': %w", latestTag, err)
	}

	// set extended information
	latestVersion.SetBranch("")
	latestVersion.SetCommitDistance(commitDistance)
	latestVersion.SetCommitHash(commitHash)
	return latestVersion, versionBump, nil
}

func (cc *ConventionalCommits) getMainBranch(repo *git.Repository) (string, error) {
	var mainBranch = ""

	refsIter, err := repo.References()
	if err != nil {
		return mainBranch, err
	}

	// find remote HEAD and strip off remote's name from branch
	refsIter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.SymbolicReference {
			if ref.Name().Short() != plumbing.HEAD.Short() {
				remote := strings.Replace(ref.Name().Short(), plumbing.HEAD.Short(), "", -1)
				mainBranch = strings.Replace(ref.Target().Short(), remote, "", -1)
			}
		}
		return nil
	})

	return mainBranch, nil
}

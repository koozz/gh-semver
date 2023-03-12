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

	"github.com/cli/go-gh"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type ConventionalCommits struct {
	gitRepo    *git.Repository
	majorRegex *regexp.Regexp
	minorRegex *regexp.Regexp
	patchRegex *regexp.Regexp
	filterPath string
	prefix     string
}

type VersionBump struct {
	major bool
	minor bool
	patch bool
}

func NewConventionalCommits(repo *git.Repository, filterPath, prefix string) *ConventionalCommits {
	return &ConventionalCommits{
		gitRepo:    repo,
		majorRegex: regexp.MustCompile(`^(fix|feat)(\(.+\))?!: |BREAKING CHANGE: `),
		minorRegex: regexp.MustCompile(`^feat(\(.+\))?: `),
		patchRegex: regexp.MustCompile(`^fix(\(.+\))?: `),
		filterPath: filterPath,
		prefix:     prefix,
	}
}

// SemVer returns the calculated next semantic version
func (cc *ConventionalCommits) SemVer() (*SemVer, error) {
	tags, err := cc.gitRepo.Tags()
	if err != nil {
		return nil, fmt.Errorf("couldn't get tags: %w", err)
	}

	// map relevant tags to commit hashes
	tagRefs := map[string]string{}
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		if cc.prefix == "" || strings.HasPrefix(ref.Name().Short(), cc.prefix) {
			var sha plumbing.Hash
			annotatedTag, _ := cc.gitRepo.TagObject(ref.Hash())
			if annotatedTag != nil {
				sha = annotatedTag.Target
			} else {
				sha = ref.Hash()
			}
			tagRefs[sha.String()] = ref.Name().Short()
		}
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
	latestMain, mainVersionBump, err := cc.traverse(tagRefs, git.LogOrderDFS)
	if err != nil {
		return nil, fmt.Errorf("couldn't walk commits on main: %w", err)
	}
	mainBranch, err := cc.getMainBranch()
	if err != nil {
		return nil, fmt.Errorf("couldn't figure out main branch: %w", err)
	}
	latestMain.SetBranch(mainBranch)

	// traverse current branch to find latest version
	latestBranch, branchVersionBump, err := cc.traverse(tagRefs, git.LogOrderDFSPost)
	if err != nil {
		return nil, fmt.Errorf("couldn't walk commits on branch: %w", err)
	}
	head, err := cc.gitRepo.Head()
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

func (cc *ConventionalCommits) traverse(tagRefs map[string]string, order git.LogOrder) (*SemVer, *VersionBump, error) {
	versionBump := &VersionBump{}

	var stopIter error = fmt.Errorf("stop commit iteration")
	var latestTag string

	var commitDistance uint64 = 0
	var commitHash string = ""

	// walk commit hashes back from HEAD via main
	commits, err := cc.gitRepo.Log(&git.LogOptions{Order: order})
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

		if relevant := cc.isRelevantCommit(commit); relevant {
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
		}
		return err
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

func (cc *ConventionalCommits) isRelevantCommit(commit *object.Commit) bool {
	// With no filtering, each commit is relevant
	if cc.prefix == "" {
		return true
	}

	// Filter on path
	fileIter, err := commit.Files()
	if err != nil {
		return true
	}

	var relevant = false
	fileIter.ForEach(func(file *object.File) error {
		if !relevant && strings.HasPrefix(file.Name, cc.filterPath) {
			relevant = true
		}
		return nil
	})
	return relevant
}

func (cc *ConventionalCommits) getMainBranch() (string, error) {
	args := []string{"repo", "view", "--json", "defaultBranchRef", "--jq", ".defaultBranchRef.name"}
	stdOut, _, err := gh.Exec(args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdOut.String()), nil
}

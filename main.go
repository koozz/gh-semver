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
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/koozz/gh-semver/internal/semver"
)

func main() {
	var (
		action bool
		/* future feature */
		// filterPath    string
		// prefix        string
		release bool
	)
	flag.BoolVar(&action, "action", false, "GitHub Action output format named 'version'")
	/* future feature */
	// flag.StringVar(&filterPath, "filter-path", "", "The path to filter commits (in case of a mono-repo)")
	// flag.StringVar(&prefix, "prefix", "", "The prefix of the tag (in case of a mono-repo)")
	flag.BoolVar(&release, "release", false, "Force release tag")
	flag.Parse()

	conventionalCommits := semver.NewConventionalCommits( /* filterPath, prefix */ )
	nextVersion, err := conventionalCommits.SemVer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}

	format := "%s\n"
	if action {
		format = "::set-output name=version::%s\n"
	}
	fmt.Printf(format, nextVersion.PrintTag(release))
}

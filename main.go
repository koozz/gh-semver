package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/cli/safeexec"
)

type SemVer struct {
	Major          int
	Minor          int
	Patch          int
	Branch         string
	CommitDistance int
	CommitHash     string
}

func parse(input string) SemVer {
	return SemVer{
		Major:          0,
		Minor:          0,
		Patch:          0,
		Branch:         "",
		CommitDistance: 0,
		CommitHash:     "",
	}
}

func main() {
	var (
		base bool
		curr bool
		next bool
	)

	flag.BoolVar(&base, "base", false, "Parses the base version.")
	flag.BoolVar(&curr, "current", false, "Parses the current version.")
	flag.BoolVar(&next, "next", false, "Parses the next version.")
	flag.Parse()

	branch := getBranch()
	if base {
		tag, err := getTag(branch)
		if err != nil {
			fmt.Println("0.0.0")
		} else {
			fmt.Printf("%s\n", tag)
		}
	}
	if curr {
		tag, err := getTag(branch)
		if err != nil {
			fmt.Println("0.0.0")
		} else {
			fmt.Printf("%s\n", tag)
		}
		distance, err := getCommitDistance(tag)
		if err != nil {
			fmt.Println("no commit distance")
		}
		hash, err := getCommitHash()
		if err != nil {
			fmt.Println("no commit hash")
		}
		semver := parse(tag)
		semver.Branch = branch
		semver.CommitDistance = distance
		semver.CommitHash = hash
	}
	if next {

	}

	// fmt.Println("hi world, this is the gh-semver extension!")
	// client, err := gh.RESTClient(nil)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// response := struct{ Login string }{}
	// err = client.Get("user", &response)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// fmt.Printf("running as %s\n", response.Login)
}

func getBranch() string {
	branch, _, err := git("symbolic-ref", "--short", "HEAD")
	if err != nil {
		branch, _, err = git("describe", "--tags")
		if err != nil {
			fmt.Println("unable to determine branch")
			os.Exit(1)
		}
	}
	fmt.Printf("branch: %s\n", branch.String())
	return strings.Split(branch.String(), "\n")[0]
}

func getTag(branch string) (string, error) {
	stdOut, _, err := git("tag", "--merged", branch)
	if err != nil {
		return "", err
	}
	var latest string
	for _, tag := range strings.Split(stdOut.String(), "\n") {
		fmt.Printf("tag: %v\n", tag)
		latest = tag
	}

	fmt.Printf("tag: %s\n", latest)
	return latest, nil
}

func getCommitDistance(tag string) (int, error) {
	// https://stackoverflow.com/questions/11657295/count-the-number-of-commits-on-a-git-branch
	//git rev-list --count HEAD ^#{tag}
	stdOut, _, err := git("rev-list", "--count", "HEAD", fmt.Sprintf("^%s", tag))
	if err != nil {
		return -1, err
	}

	result := strings.Split(stdOut.String(), "\n")[0]
	distance, err := strconv.Atoi(result)
	if err != nil {
		return -1, err
	}

	return distance, nil
}

func getCommitHash() (string, error) {
	stdOut, _, err := git("rev-parse", "--verify", "HEAD", "--short")
	if err != nil {
		return "", err
	}

	return strings.Split(stdOut.String(), "\n")[0], nil
}

func getVersionBump() {
	// walk all commits, check commit messages
}

// For more examples of using go-gh, see:
// https://github.com/cli/go-gh/blob/trunk/example_gh_test.go

// Exec gh command with provided arguments.
func git(args ...string) (stdOut, stdErr bytes.Buffer, err error) {
	path, err := path()
	if err != nil {
		err = fmt.Errorf("could not find gh executable in PATH. error: %w", err)
		return
	}
	return run(path, nil, args...)
}

func path() (string, error) {
	return safeexec.LookPath("git")
}

func run(path string, env []string, args ...string) (stdOut, stdErr bytes.Buffer, err error) {
	cmd := exec.Command(path, args...)
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	if env != nil {
		cmd.Env = env
	}
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("failed to run gh: %s. error: %w", stdErr.String(), err)
		return
	}
	return
}

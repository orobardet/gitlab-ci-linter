// Copyright © 2017-2020 Olivier Robardet
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"path/filepath"
)

// 'install' command of the program
func commandInstall(c *cli.Context) error {

	if c.Args().Present() && c.Args().Get(0) != "" {
		processPathArgument(c.Args().Get(0))
	}

	// Find git repository. First, start from gitlab-ci file location
	gitRepoPath, err := findGitRepo(filepath.Dir(gitlabCiFilePath))
	if err == nil {
		// if not found, search from directoryRoot
		gitRepoPath, _ = findGitRepo(directoryRoot)
	}

	if gitRepoPath == "" {
		return cli.NewExitError(fmt.Sprintf("No GIT repository found, can't install a hook"), 5)
	}
	if verboseMode {
		fmt.Printf("Git repository found: %s\n", gitRepoPath)
	}

	// Extract origin remote url from repository config
	remoteUrl, err := getGitOriginRemoteUrl(gitRepoPath)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to find origin remote url in repository: %s", err), 5)
	}

	// Check if we can use the origin remote url
	if remoteUrl != "" {
		// Guess gitlab url based on remote url
		_, err = guessGitlabAPIFromGitRemoteUrl(remoteUrl)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("No valid and responding Gitlab API URL found from repository's origin remote, can't install a hook"), 5)
		}
	} else if verboseMode {
		// Warn user that we're defaulting because no origin remote was found
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Fprintf(color.Output, yellow("No origin remote found in repository, will be using default Gitlab API '%s'\n"), gitlabRootUrl)
	}

	status, err := createGitHookLink(gitRepoPath, "pre-commit")
	if err != nil {
		return cli.NewExitError(err, 5)
	}
	switch status {
	case HookAlreadyExists:
		yellow := color.New(color.FgYellow).SprintFunc()
		msg := fmt.Sprintf(yellow("A pre-commit hook already exists\nPlease install manually by adding a call to me in your pre-commit script."))
		return cli.NewExitError(msg, 4)
	case HookAlreadyCreated:
		cyan := color.New(color.FgCyan).SprintFunc()
		fmt.Fprintf(color.Output, cyan("Already installed.\n"))
	case HookCreated:
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Fprintf(color.Output, green("Git pre-commit hook installed in %s\n"), filepath.Dir(gitRepoPath))
	default:
		return cli.NewExitError("Unkown error", 5)
	}

	return nil
}

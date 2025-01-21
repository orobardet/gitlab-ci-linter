// Copyright © 2018-2020 Olivier Robardet
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
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func getGitlabLintURL(gitRepoPath string) (string, error) {
	// If a gitlab URL was given as parameter, just use it
	if gitlabRootURL != "" {
		return gitlabRootURL, nil
	}

	// Else, let's try to guess it, it there is a git repository
	if gitRepoPath == "" {
		// Warn user that we're defaulting because no git repo was found
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Fprintf(color.Output, yellow("No GIT repository found, using default Gitlab API '%s'\n"), defaultGitlabRootURL)

		return defaultGitlabRootURL, nil
	}

	// Extract origin remote url from repository config
	remoteURL, err := getGitOriginRemoteURL(gitRepoPath)
	if err != nil {
		return defaultGitlabRootURL, fmt.Errorf("Failed to find origin remote url in repository: %s", err)
	}

	// Check if we can use the origin remote url
	if remoteURL != "" {
		// Guess gitlab url based on remote url
		localGitlabRootURL, err := guessGitlabAPIFromGitRemoteURL(remoteURL)
		if err != nil {
			return defaultGitlabRootURL, fmt.Errorf("No valid and responding Gitlab API URL found from repository's origin remote: %s", err)
		}
		return localGitlabRootURL, nil
	}

	// Warn user that we're defaulting because no origin remote was found
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Fprintf(color.Output, yellow("No origin remote found in repository, using default Gitlab API '%s'\n"), gitlabRootURL)

	return defaultGitlabRootURL, nil
}

// 'check' command of the program, which is the main action
// It aims to validate the syntax of a .gitlab-ci.yml file, using the CI Lint API of a Gitlab instance
// First it search for a gitlab-ci file if no one is given
// Then it search for a .git repository directory
// If a .git repository is found, its origin remote is analysed to extract and guess a the Gitlab root url to use for
// the API. If no valid origin remote or API is found, the defaultGitlabRootURL is used
// Finally, it call the API with the gitlab-ci file content. If the content if syntax valid, it silently stop. Else it
// display the error messages returned by the API and exit with an error
func commandCheck(c *cli.Context) error {

	if c.Args().Present() && c.Args().Get(0) != "" {
		processPathArgument(c.Args().Get(0))
	}

	if verboseMode {
		fmt.Printf("Settings:\n  directoryRoot: %s\n  gitlabCiFilePath: %s\n", directoryRoot, gitlabCiFilePath)
	}

	// Find gitlab-ci file, if not given
	if gitlabCiFilePath == "" {
		file, err := findGitlabCiFile(directoryRoot)
		if err != nil {
			fmt.Println("No gitlab-ci file found")
			return nil
		}
		gitlabCiFilePath = file
	}

	cwd, _ := os.Getwd()
	relativeGitlabCiFilePath, _ := filepath.Rel(cwd, gitlabCiFilePath)

	// Find git repository. First, start from gitlab-ci file location
	gitRepoPath, err := findGitRepo(filepath.Dir(gitlabCiFilePath))
	if err == nil {
		// if not found, search from directoryRoot
		gitRepoPath, _ = findGitRepo(directoryRoot)
	}

	localGitlabLintURL, err := getGitlabLintURL(gitRepoPath)
	if err != nil {
		return cli.Exit(err, 5)
	}

	fmt.Printf("Validating %s... ", relativeGitlabCiFilePath)

	if verboseMode {
		fmt.Printf("\n")
	}

	// Call the API to validate the gitlab-ci file
	ciFileContent, err := os.ReadFile(gitlabCiFilePath)
	if err != nil {
		return cli.Exit(fmt.Sprintf("error while reading '%s' file content: %s", relativeGitlabCiFilePath, err), 5)
	}

	status, errorMessages, err := lintGitlabCIUsingAPI(localGitlabLintURL, string(ciFileContent))
	if err != nil {
		return cli.Exit(fmt.Errorf("error linting using Gitlab API %s: %w", localGitlabLintURL, err), 5)
	}

	if !status {
		if verboseMode {
			fmt.Printf("%s ", relativeGitlabCiFilePath)
		}

		red := color.New(color.FgRed).SprintFunc()
		fmt.Fprintf(color.Output, "%s\n", red("KO"))

		messages := red(strings.Join(errorMessages, "\n"))
		fmt.Fprintf(os.Stderr, "%s\n", messages)

		return cli.Exit("", 10)
	}

	if verboseMode {
		fmt.Printf("%s ", relativeGitlabCiFilePath)
	}
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Fprintf(color.Output, "%s\n", green("OK"))

	return nil
}

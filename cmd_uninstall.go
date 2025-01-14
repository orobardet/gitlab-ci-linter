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
	"path/filepath"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

// 'uninstall' command of the program
func commandUninstall(c *cli.Context) error {

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
		return cli.Exit("No GIT repository found, can't install a hook", 5)
	}
	if verboseMode {
		fmt.Printf("Git repository found: %s\n", gitRepoPath)
	}

	status, err := deleteGitHookLink(gitRepoPath, "pre-commit")
	if err != nil {
		return cli.Exit(err, 5)
	}
	switch status {
	case HookNotMatching:
		red := color.New(color.FgRed).SprintFunc()
		return cli.Exit(red("Unknown pre-commit hook\nPlease uninstall manually."), 4)
	case HookNotExisting:
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Fprintf(color.Output, "%s\n", yellow("No pre-commit hook found."))
	case HookDeleted:
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Fprintf(color.Output, "%s\n", green("Git pre-commit hook uninstalled."))
	default:
		return cli.Exit("Unknown error", 5)
	}

	return nil
}

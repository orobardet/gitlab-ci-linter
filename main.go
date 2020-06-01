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
	"math"
	"net/url"
	"os"
	"path/filepath"
)

// APPNAME contains the application name
var APPNAME = "gitlab-ci-linter"

// VERSION contains the version of the program
var VERSION = "0.0.0-dev"

// REVISION contains the revision of the program
var REVISION = "HEAD"

// BUILDTIME contains the build date and time of the program
var BUILDTIME = ""

// The Gitlab instance root URL to use.
var gitlabRootURL string

// The full path of the gitlab-ci file to check, if given at calls.
// If no path is given at call, the variable will be an empty string, and the program will search for the file
// using gitlabCiFileName.
// Search start on the directoryRoot, and goes up in the directory hierarchy until a file is found or the root is reach
var gitlabCiFilePath string

// Directory to start searching for gitlab-ci file and git repository
var directoryRoot string

// Personal access token for accessing the repository when you have two factor authentication (2FA) enabled.
var personalAccessToken string

// Timeout in seconds for HTTP request to the Gitlab API
// Request will fail if lasting more than the timeout
var httpRequestTimeout uint = 5

// Tells if output should be colorized or not
var colorMode = true

// Tells if verbose mode is on or off
var verboseMode = false

// Analyse a PATH argument, that can be a directory or file, to use it as a gitlab-ci file a a directory
// where to start searching
func processPathArgument(path string) {
	fileInfo, err := os.Stat(path)
	if !os.IsNotExist(err) {
		if fileInfo.IsDir() {
			directoryRoot, _ = filepath.Abs(path)
			if verboseMode {
				fmt.Printf("%s directory used as repository root.\n", path)
			}
		} else {
			gitlabCiFilePath, _ = filepath.Abs(path)
			if verboseMode {
				fmt.Printf("%s used as gitlab-ci.yml file.\n", path)
			}
		}
	}
}

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("version=%s revision=%s built on=%s\n", VERSION, REVISION, BUILDTIME)
	}

	cli.AppHelpTemplate = `{{.Name}} - {{.Usage}}
version {{if .Version}}{{.Version}}{{end}}
{{if len .Authors}}{{range .Authors}}{{ . }}{{end}}{{end}} - https://gitlab.com/orobardet/gitlab-ci-linter

Usage:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} [command [command options]]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}

{{if .VisibleFlags}}Global options:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
{{if .Description}}Arguments:
   {{.Description}}{{end}}

{{if .Commands}}Commands:
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}
   If no command is given, 'check 'is used by default
{{end}}`

	app := cli.NewApp()
	app.Name = APPNAME
	app.Version = fmt.Sprintf("%s (%s)", VERSION, REVISION[:int(math.Min(float64(len(REVISION)), 7))])
	app.Authors = []*cli.Author{
		{Name: "Olivier ROBARDET"},
	}
	app.Usage = "lint your .gitlab-ci.yml using the Gitlab lint API"
	app.EnableBashCompletion = true
	app.UseShortOptionHandling = true

	pathArgumentDescription := `If PATH if given, it will depending of its type on filesystem:
    - if a file, it will be used as the gitlab-ci file to check (similar to global --ci-file option)
    - if a directory, it will be used as the folder from where to search for a ci file and a git repository (similar to global --directory option)
   PATH have precedence over --ci-file and --directory options.`

	app.ArgsUsage = "[PATH]"
	app.Description = pathArgumentDescription
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "gitlab-url",
			Aliases:     []string{"u"},
			Value:       "",
			Usage:       fmt.Sprintf("root `URL` of the Gitlab instance to use API (default: auto-detect from remote origin, else \"%s\")", defaultGitlabRootURL),
			EnvVars:     []string{"GCL_GITLAB_URL"},
			Destination: &gitlabRootURL,
		},
		&cli.StringFlag{
			Name:        "ci-file",
			Aliases:     []string{"f"},
			Usage:       "`FILE` is the relative or absolute path to the gitlab-ci file",
			EnvVars:     []string{"GCL_GITLAB_CI_FILE"},
			Destination: &gitlabCiFilePath,
		},
		&cli.StringFlag{
			Name:        "directory",
			Aliases:     []string{"d"},
			Value:       ".",
			Usage:       "`DIR` is the directory from where to search for gitlab-ci file and git repository",
			EnvVars:     []string{"GCL_DIRECTORY"},
			Destination: &directoryRoot,
		},
		&cli.StringFlag{
			Name:        "personal-access-token",
			Aliases:     []string{"p"},
			Value:       "",
			Usage:       "personal access token `TOK` for accessing repositories when you have 2FA enabled",
			EnvVars:     []string{"GCL_PERSONAL_ACCESS_TOKEN"},
			Destination: &personalAccessToken,
		},
		&cli.UintFlag{
			Name:        "timeout",
			Aliases:     []string{"t"},
			Value:       httpRequestTimeout,
			Usage:       "timeout in second after which http request to Gitlab API will timeout (and the program will fails)",
			EnvVars:     []string{"GCL_TIMEOUT"},
			Destination: &httpRequestTimeout,
		},
		&cli.BoolFlag{
			Name:    "no-color",
			Aliases: []string{"n"},
			Usage:   "don't color output. By defaults the output is colorized if a compatible terminal is detected.",
			EnvVars: []string{"GCL_NOCOLOR"},
		},
		&cli.BoolFlag{
			Name:        "verbose",
			Aliases:     []string{"v"},
			Usage:       "verbose mode",
			EnvVars:     []string{"GCL_VERBOSE"},
			Destination: &verboseMode,
		},
	}
	cli.VersionFlag = &cli.BoolFlag{
		Name:  "version, V",
		Usage: "print the version information",
	}

	app.Commands = []*cli.Command{
		{
			Name:        "check",
			Aliases:     []string{"c"},
			Usage:       "Check the .gitlab-ci.yml (default command if none is given)",
			Action:      commandCheck,
			ArgsUsage:   "[PATH]",
			Description: pathArgumentDescription,
		},
		{
			Name:        "install",
			Aliases:     []string{"i"},
			Usage:       "install as git pre-commit hook",
			Action:      commandInstall,
			ArgsUsage:   "[PATH]",
			Description: pathArgumentDescription,
		},
		{
			Name:        "uninstall",
			Aliases:     []string{"u"},
			Usage:       "uninstall the git pre-commit hook",
			Action:      commandUninstall,
			ArgsUsage:   "[PATH]",
			Description: pathArgumentDescription,
		},
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Print the version information",
			Action: func(c *cli.Context) error {
				cli.ShowVersion(c)
				return nil
			},
		},
	}

	app.Before = func(c *cli.Context) error {
		if c.Bool("no-color") {
			colorMode = false
			color.NoColor = true
		}

		// Check if the given directory path exists
		if directoryRoot != "" {
			directoryRoot, _ = filepath.Abs(directoryRoot)
			fileInfo, err := os.Stat(directoryRoot)
			if os.IsNotExist(err) {
				return cli.NewExitError(fmt.Sprintf("'%s' does not exists", directoryRoot), 1)
			}
			if !fileInfo.IsDir() {
				return cli.NewExitError(fmt.Sprintf("'%s' is not a directory", directoryRoot), 1)
			}
		}

		// Check if the given gitlab-ci file path exists
		if gitlabCiFilePath != "" {
			gitlabCiFilePath, _ = filepath.Abs(gitlabCiFilePath)
			fileInfo, err := os.Stat(gitlabCiFilePath)
			if os.IsNotExist(err) {
				return cli.NewExitError(fmt.Sprintf("'%s' does not exists", gitlabCiFilePath), 1)
			}
			if fileInfo.IsDir() {
				return cli.NewExitError(fmt.Sprintf("'%s' is a directory, not a file", gitlabCiFilePath), 1)
			}
		}

		if gitlabRootURL != "" {
			u, err := url.Parse(gitlabRootURL)
			if err != nil {
				cli.NewExitError(fmt.Sprintf("Unable to parse gitlab root URL '%s': %s", gitlabRootURL, err), 1)
			}
			if u.Scheme == "" {
				u.Scheme = "https"
			}
			gitlabRootURL = u.String()
		}

		return nil
	}

	app.Action = func(c *cli.Context) error {
		return commandCheck(c)
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}

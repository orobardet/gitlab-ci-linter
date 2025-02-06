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
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"gitlab.com/orobardet/gitlab-ci-linter/config"
)

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

// Try to get personal access token as 'account' from .netrc file
var useNetrc bool

// Path of .netrc file to use
var netrcFile string

// The project path (namespace + name) of the GitLab project that is used in the API endpoint to validate the CI configuration.
var projectPath string

// The identifier of the GitLab project that is used in the API endpoint to validate the CI configuration.
var projectID string

// Timeout in seconds for HTTP request to the Gitlab API
// Request will fail if lasting more than the timeout
var httpRequestTimeout int64 = 15

// Tells if output should be colorized or not
var colorMode = true //nolint:unused

// Tells if verbose mode is on or off
var verboseMode = false

// Tells if the response should include the merged yaml from the Gitlab API
var includeMergedYaml = false

// Tells if run pipeline creation simulation
var dryRun = false

// When dry_run is true, sets the branch or tag context to use to validate the CI/CD YAML configuration. Defaults to the project’s default branch when not set.
var dryRunRef string

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
	cli.VersionPrinter = func(_ *cli.Context) {
		fmt.Printf("version=%s revision=%s built on=%s\n", config.VERSION, config.REVISION, config.BUILDTIME)
	}

	cli.AppHelpTemplate = `{{.Name}} - {{.Usage}}
version {{if .Version}}{{.Version}}{{end}}
{{if len .Authors}}{{range .Authors}}{{ . }}{{end}}{{end}} - https://gitlab.com/orobardet/gitlab-ci-linter

Usage:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} [command [command options]]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}

   The used Gitlab API is tied to a Gitlab project. Thus, the tools needs to know which Gitlab project (on which Gitlab instance) it has to target.
   By default, it will try to autodetect from the 'origin' remote configured in the git repository (if any), by extracting the FQDN as the root URL, 
   and the project path. Works for 'http'' or 'ssh' remotes. e.g.: a remote "https://gitlab.com/orobardet/gitlab-ci-linter.git" or
   "git@gitlab.com:orobardet/gitlab-ci-linter.git" will target the API of the project "orobardet/gitlab-ci-linter" on "https://gitlab.com".

   In case the auto-detection does not work, or you don't have a compatible remote, or you want to target another project, you can specify the Gitlab
   root URL using '-gitlab-url|-u' flag, and the project using '--project-path|-P' or '--project-id|-I' flags. '--project-id' has precedence over '--project-path'.

   If your gitlab instance or project needs an authentification (which is the case on gitlab.com), you have to specify a personal access token with '--personal-access-token|-p'.
   You can also use the flag '--netrc|-n' to try getting the token from the .netrc file (by default ~/.netrc on *nix, $HOME/_netrc on Windows), but not the token must be set
   on the 'account' field, not 'password' (to prevent conflict with basic auth). Also, the 'default' entry of .netrc is ignored.
   e.g.: for gitlab.com, the .netrc entry should be:
      machine gitlab.com
        # possible login and password definition
        account MY_PERSONAL_ACCESS_TOKEN

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
	app.Name = config.APPNAME
	app.Version = fmt.Sprintf("%s (%s)", config.VERSION, config.REVISION[:int(math.Min(float64(len(config.REVISION)), 7))])
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
			Usage:       "personal access token `TOK` for accessing repositories when you have 2FA enabled. Has precedence over .netrc usage",
			EnvVars:     []string{"GCL_PERSONAL_ACCESS_TOKEN"},
			Destination: &personalAccessToken,
		},
		&cli.BoolFlag{
			Name:        "netrc",
			Aliases:     []string{"n"},
			Usage:       "Try to get personal access token as 'account' from .netrc file",
			EnvVars:     []string{"GCL_NETRC"},
			Destination: &useNetrc,
		},
		&cli.StringFlag{
			Name:        "netrc-file",
			Value:       "",
			Usage:       "Path of .netrc file to use. By default, try to detect it.",
			EnvVars:     []string{"GCL_NETRC_FILE"},
			Destination: &netrcFile,
		},
		&cli.StringFlag{
			Name:        "project-path",
			Aliases:     []string{"P"},
			Value:       "",
			Usage:       "`PATH` of the GitLab project that is used in the API for Gitlab >=13.6. Has precedence over path guessing from remote",
			EnvVars:     []string{"CI_PROJECT_PATH", "GCL_PROJECT_PATH"},
			Destination: &projectPath,
		},
		&cli.StringFlag{
			Name:        "project-id",
			Aliases:     []string{"I"},
			Value:       "",
			Usage:       "`ID` of the GitLab project that is used in the API for Gitlab >=13.6. Has precedence over --project-path",
			EnvVars:     []string{"CI_PROJECT_ID", "GCL_PROJECT_ID"},
			Destination: &projectID,
		},
		&cli.Int64Flag{
			Name:        "timeout",
			Aliases:     []string{"t"},
			Value:       httpRequestTimeout,
			Usage:       "timeout in second after which http request to Gitlab API will timeout (and the program will fails)",
			EnvVars:     []string{"GCL_TIMEOUT"},
			Destination: &httpRequestTimeout,
		},
		&cli.BoolFlag{
			Name:    "no-color",
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
		&cli.BoolFlag{
			Name:        "merged-yaml",
			Aliases:     []string{"m"},
			Usage:       "include merged yaml in response",
			EnvVars:     []string{"GCL_INCLUDE_MERGED_YAML"},
			Destination: &includeMergedYaml,
		},
		&cli.BoolFlag{
			Name:        "dry-run",
			Aliases:     []string{"s"},
			Usage:       "run pipeline creation simulation",
			EnvVars:     []string{"GCL_DRY_RUN"},
			Destination: &dryRun,
		},
		&cli.StringFlag{
			Name:        "dry-run-ref",
			Usage:       "When dry_run is true, sets the branch or tag to validate ci yml",
			EnvVars:     []string{"GCL_DRY_RUN_REF"},
			Destination: &dryRunRef,
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
				return cli.Exit(fmt.Sprintf("'%s' does not exists", directoryRoot), 1)
			}
			if !fileInfo.IsDir() {
				return cli.Exit(fmt.Sprintf("'%s' is not a directory", directoryRoot), 1)
			}
		}

		// Check if the given gitlab-ci file path exists
		if gitlabCiFilePath != "" {
			gitlabCiFilePath, _ = filepath.Abs(gitlabCiFilePath)
			fileInfo, err := os.Stat(gitlabCiFilePath)
			if os.IsNotExist(err) {
				return cli.Exit(fmt.Sprintf("'%s' does not exists", gitlabCiFilePath), 1)
			}
			if fileInfo.IsDir() {
				return cli.Exit(fmt.Sprintf("'%s' is a directory, not a file", gitlabCiFilePath), 1)
			}
		}

		gitlabRootURL = strings.TrimSpace(gitlabRootURL)
		if gitlabRootURL != "" {
			u, err := url.Parse(gitlabRootURL)
			if err != nil {
				return cli.Exit(fmt.Sprintf("Unable to parse gitlab root URL '%s': %s", gitlabRootURL, err), 1)
			}
			if u.Scheme == "" {
				u.Scheme = "https"
			}
			gitlabRootURL = u.String()
		}

		projectPath = strings.TrimSpace(projectPath)
		projectID = strings.TrimSpace(projectID)

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

package main

import (
	"errors"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/urfave/cli"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Version of the program
var VERSION = "0.0.0-dev"

// Revision of the program
var REVISION = "HEAD"

// Build date and time of the program
var BUILD_TIME = ""

// Name of the git repo directory
const gitRepoDirectory = ".git"

// Name of the git repo config file in a git repo directory
const gitRepoConfigFilename = "config"

// Filename of a gitlab-ci file. Used to find the gitlab-ci file if no path are given at calls
const gitlabCiFileName = ".gitlab-ci.yml"

// Default Gitlab instance URL to use
const defaultGitlabRootUrl = "https://gitlab.com"

// Path of the Gitlab CI lint API, to be used on the root url
const gitlabApiCiLintPath = "/api/v4/ci/lint"

// The Gitlab instance root URL to use.
var gitlabRootUrl string

// The full path of the gitlab-ci file to check, if given at calls.
// If no path is given at call, the variable will be an empty string, and the program will search for the file
// using gitlabCiFileName.
// Search start on the directoryRoot, and goes up in the directory hierarchy until a file is found or the root is reach
var gitlabCiFilePath string

// Directory to start searching for gitlab-ci file and git repository
var directoryRoot string

// Search in the given directory a git repository directory
// It goes up in the filesystem hierarchy until a repository is found, or the root is reach
// A git repository directory is a '.git' folder (gitRepoDirectory constant) containing a 'config' file
// (gitRepoConfigFilename constant)
func findGitRepo(directory string) (string, error) {
	candidate := directory + string(filepath.Separator) + gitRepoDirectory

	fileInfo, err := os.Stat(candidate)
	if !os.IsNotExist(err) && fileInfo.IsDir() {
		// Found a git directory, check of it has a config file
		fileInfo, err = os.Stat(candidate + string(filepath.Separator) + gitRepoConfigFilename)

		if !os.IsNotExist(err) && !fileInfo.IsDir() {
			return candidate, nil
		}
	}

	// If we are at the root of the filesystem, it means we did not find any gitlab-ci file
	if directory[len(directory)-1] == filepath.Separator {
		return "", errors.New("not found")
	} else { // else check the parent directory
		return findGitRepo(filepath.Dir(directory))
	}
}

// Search in the given directory a git repository directory
// It goes up in the filesystem hierarchy until a repository is found, or the root is reach
func findGitlabCiFile(directory string) (string, error) {
	candidate := directory + string(filepath.Separator) + gitlabCiFileName

	fileInfo, err := os.Stat(candidate)
	if !os.IsNotExist(err) && !fileInfo.IsDir() {
		return candidate, nil
	}

	// If we are at the root of the filesystem, it means we did not find any gitlab-ci file
	if directory[len(directory)-1] == filepath.Separator {
		return "", errors.New("not found")
	} else { // else check the parent directory
		return findGitlabCiFile(filepath.Dir(directory))
	}
}

// Extract the orign remote remote url from a git repo directory
func getGitOriginRemoteUrl(gitDirectory string) (string, error) {
	cfg, err := ini.Load(gitDirectory + string(filepath.Separator) + gitRepoConfigFilename)

	if err != nil {
		fmt.Println(err)
		return "", err
	}

	remote, err := cfg.GetSection("remote \"origin\"")
	if err == nil && remote.HasKey("url") {
		return remote.Key("url").String(), nil
	}

	return "", nil
}

// Transform a git remote url, that can be a full http ou ssh url, to a simple http FQDN host
func httpiseRemoteUrl(remoteUrl string) string {
	re := regexp.MustCompile("^(https?://[^/]*).*$")
	if re.MatchString(remoteUrl) { // http remote
		matches := re.FindStringSubmatch(remoteUrl)
		if len(matches) >= 2 {
			return matches[1]
		}
	} else { // ssh remote
		re = regexp.MustCompile("^([^@]*@)?([^:]+)")
		matches := re.FindStringSubmatch(remoteUrl)
		if len(matches) >= 3 {
			return "http://" + matches[2]
		}
	}

	return ""
}

func checkGitlabAPIUrl(rootUrl string) bool {
	resp, err := http.Post(rootUrl+gitlabApiCiLintPath, "application/json", strings.NewReader(`{"content": "{ \"image\": \"ruby:2.1\", \"services\": [\"postgres\"], \"before_script\": [\"gem install bundler\", \"bundle install\", \"bundle exec rake db:create\"], \"variables\": {\"DB_NAME\": \"postgres\"}, \"types\": [\"test\", \"deploy\", \"notify\"], \"rspec\": { \"script\": \"rake spec\", \"tags\": [\"ruby\", \"postgres\"], \"only\": [\"branches\"]}}"}`))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	fmt.Printf("%s\n", body)

	return false
}

// 'check' command of the program, which is the main action
// It aims to validate the syntax of a .gitlab-ci.yml file, using the CI Lint API of a Gitlab instance
// First it search for a gitlab-ci file if no one is given
// Then it search for a .git repository directory
// If a .git repository is found, its origin remote is analysed to extract and guess a the Gitlab root url to use for
// the API. If no valid origin remote or API is found, the defaultGitlabRootUrl is used
// Finally, it call the API with the gitlab-ci file content. If the content if syntax valid, it silently stop. Else it
// display the error messages returned by the API and exit with an error
func commandCheck(c *cli.Context) error {
	directoryRoot, _ = filepath.Abs(directoryRoot)

	// Check if the given gitlab-ci file path exists
	if gitlabCiFilePath != "" {
		fileInfo, err := os.Stat(gitlabCiFilePath)
		if os.IsNotExist(err) {
			cli.NewExitError(fmt.Sprintf("'%s' does not exists", gitlabCiFilePath), 1)
		}
		if fileInfo.IsDir() {
			cli.NewExitError(fmt.Sprintf("'%s' is a directory, not a file", gitlabCiFilePath), 1)
		}
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

	// Find git repository
	// First, start from gitlab-ci file location
	gitRepoPath, err := findGitRepo(filepath.Dir(gitlabCiFilePath))
	if err == nil {
		// if not found, search from directoryRoot
		gitRepoPath, _ = findGitRepo(directoryRoot)
	}

	// Extract origin remote from repository en guess gitlab url
	if gitRepoPath != "" {
		remoteUrl, err := getGitOriginRemoteUrl(gitRepoPath)
		if err == nil {
			httpRemoteUrl := httpiseRemoteUrl(remoteUrl)
			if httpRemoteUrl != "" && checkGitlabAPIUrl(httpRemoteUrl) {
				fmt.Printf("API url found: %s", httpRemoteUrl)
				gitlabRootUrl = httpRemoteUrl
			}
		}
	}

	// Call the API to validate the gitlab-ci file

	return nil
}

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("version=%s revision=%s built on=%s\n", VERSION, REVISION, BUILD_TIME)
	}

	cli.AppHelpTemplate = `{{.Name}} - {{.Usage}}
version {{if .Version}}{{.Version}}{{end}}
{{if len .Authors}}{{range .Authors}}{{ . }}{{end}}{{end}} - https://gitlab.com/orobardet/gitlab-ci-linter

Usage:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}

{{if .VisibleFlags}}Global options:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
{{if .Commands}}Commands:
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}`

	app := cli.NewApp()
	app.Name = "gitlab-ci-linter"
	app.Version = fmt.Sprintf("%s (%s)", VERSION, REVISION[:int(math.Min(float64(len(REVISION)), 7))])
	app.Authors = []cli.Author{
		{Name: "Olivier ROBARDET"},
	}
	app.Usage = "lint your .gitlab-ci.yml using the Gitlab lint API"
	app.EnableBashCompletion = true

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "gitlab-url,u",
			Value:       defaultGitlabRootUrl,
			Usage:       "Root `URL` of the Gitlab instance to use API",
			EnvVar:      "GCL_GITLAB_URL",
			Destination: &gitlabRootUrl,
		},
		cli.StringFlag{
			Name:        "ci-file,f",
			Usage:       "`FILE` is the relative or absolute path to the gitlab-ci file",
			EnvVar:      "GCL_GITLAB_CI_FILE",
			Destination: &gitlabCiFilePath,
		},
		cli.StringFlag{
			Name:        "directory,d",
			Value:       ".",
			Usage:       "`DIR` is the directory from where to search for gitlab-ci file and git repository",
			EnvVar:      "GCL_DIRECTORY",
			Destination: &directoryRoot,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Show version information",
			Action: func(c *cli.Context) {
				cli.ShowVersion(c)
			},
		},
		{
			Name:    "check",
			Aliases: []string{"c"},
			Usage:   "Check the .gitlab-ci.yml (default commend if none is given)",
			Action:  commandCheck,
		},
	}

	app.Action = commandCheck

	app.Run(os.Args)
}

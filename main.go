package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/go-ini/ini"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Application name
var APPNAME = "gitlab-ci-linter"

// Version of the program
var VERSION = "0.0.0-dev"

// Revision of the program
var REVISION = "HEAD"

// Build date and time of the program
var BUILDTIME = ""

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

// Personal access token for accessing the repository when you have two factor authentication (2FA) enabled.
var personalAccessToken string

// Timeout in seconds for HTTP request to the Gitlab API
// Request will fail if lasting more than the timeout
var httpRequestTimeout uint = 5

// Tells if output should be colorized or not
var colorMode = true

// Tells if verbose mode is on or off
var verboseMode = false

type GitlabAPILintRequest struct {
	Content string `json:"content"`
}

type GitlabAPILintResponse struct {
	Status string   `json:"status,omitempty"`
	Error  string   `json:"error,omitempty"`
	Errors []string `json:"errors,omitempty"`
}

const (
	HookError = iota
	HookCreated
	HookAlreadyCreated
	HookAlreadyExists
	HookDeleted
	HookNotExisting
	HookNotMatching
)

// Search in the given directory a git repository directory
// It goes up in the filesystem hierarchy until a repository is found, or the root is reach
// A git repository directory is a '.git' folder (gitRepoDirectory constant) containing a 'config' file
// (gitRepoConfigFilename constant)
func findGitRepo(directory string) (string, error) {
	candidate := path.Join(directory, gitRepoDirectory)

	fileInfo, err := os.Stat(candidate)
	if !os.IsNotExist(err) && fileInfo.IsDir() {
		// Found a git directory, check of it has a config file
		fileInfo, err = os.Stat(path.Join(candidate, gitRepoConfigFilename))

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
	candidate := path.Join(directory, gitlabCiFileName)

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

// Load git config file from git repository directory
func loadGitCfg(gitDirectory string) (*ini.File, error) {
	cfg, err := ini.Load(path.Join(gitDirectory, gitRepoConfigFilename))
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// Extract the origin remote remote url from a git config file
func getGitOriginRemoteUrl(gitDirectory string) (string, error) {
	cfg, err := loadGitCfg(gitDirectory)
	if err != nil {
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

func initGitlabHttpClientRequest(method string, url string, content string) (*http.Client, *http.Request, error) {
	var httpClient *http.Client
	var req *http.Request

	httpClient = &http.Client{
		Timeout: time.Second * time.Duration(httpRequestTimeout),
	}

	req, err := http.NewRequest(method, url, strings.NewReader(content))
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", APPNAME, VERSION))
	if personalAccessToken != "" {
		req.Header.Add("PRIVATE-TOKEN", personalAccessToken)
	}

	return httpClient, req, err
}

// Check if we can get a response with the rootUrl on the API CI Lint endpoint, and if a redirection occurs
// If a redirection is detected, return the redirected root URL.
// This is needed as redirection response only occurs when the API is call using en HTTP GET, but in the en the API
// has to be called in POST
func checkGitlabAPIUrl(rootUrl string) (string, error) {

	newRootUrl := rootUrl

	lintURL := rootUrl + gitlabApiCiLintPath

	if verboseMode {
		fmt.Printf("Checking '%s' (using '%s')...\n", rootUrl, lintURL)
	}

	httpClient, req, err := initGitlabHttpClientRequest("GET", lintURL, "")
	if err != nil {
		return newRootUrl, err
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		return newRootUrl, err
	}
	defer resp.Body.Close()

	// Getting the full URL used for the last query, after following potential redirection
	lastUrl := resp.Request.URL.String()

	// Let's try to get the redirected root URL by removing the gitlab API path from the last use URL
	lastRootUrl := strings.TrimSuffix(lastUrl, gitlabApiCiLintPath)
	// If the result is not empty or unchanged, it means
	if lastRootUrl != "" && lastRootUrl != lastUrl {
		newRootUrl = lastRootUrl
	}

	if verboseMode {
		fmt.Printf("Url '%s' validated\n", newRootUrl)
	}

	return newRootUrl, nil
}

// Send the content of a gitlab-ci file to a Gitlab instance lint API to check its validity
// In case of invalid, lint error messages are returned in `msgs`
func lintGitlabCIUsingAPI(rootUrl string, ciFileContent string) (status bool, msgs []string, err error) {

	msgs = []string{}
	status = false

	// Prepare the JSON content of the POST request:
	// {
	//   "content": "<ESCAPED CONTENT OF THE GITLAB-CI FILE>"
	// }
	var reqParams = GitlabAPILintRequest{Content: ciFileContent}
	reqBody, _ := json.Marshal(reqParams)

	// Prepare requesting the API
	lintURL := rootUrl + gitlabApiCiLintPath
	if verboseMode {
		fmt.Printf("Querying %s...\n", lintURL)
	}
	httpClient, req, err := initGitlabHttpClientRequest("POST", lintURL, string(reqBody))

	// Make the request to the API
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Get the results
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var result GitlabAPILintResponse
	err = json.Unmarshal([]byte(body), &result)
	if err != nil {
		return
	}

	// Analyse the results
	if result.Status == "valid" {
		status = true
		return
	}

	if result.Status == "invalid" {
		msgs = result.Errors
	}

	if result.Error != "" {
		err = errors.New(result.Error)
	}

	return
}

// Analyse a PATH argument, that can be a directory or file, to use it as a gitlab-ci file a a directory
// where to start searching
func processPathArgument(path string) {
	fileInfo, err := os.Stat(path)
	if !os.IsNotExist(err) {
		if fileInfo.IsDir() {
			directoryRoot, _ = filepath.Abs(path)
		} else {
			gitlabCiFilePath, _ = filepath.Abs(path)
		}
	}
}

func guessGitlabAPIFromGitRemoteUrl(remoteUrl string) (apiRootUrl string, err error) {
	httpRemoteUrl, err := checkGitlabAPIUrl(httpiseRemoteUrl(remoteUrl))
	if err != nil {
		return "", err
	}
	if httpRemoteUrl != "" {
		apiRootUrl = httpRemoteUrl
		if verboseMode {
			fmt.Printf("API url found: %s\n", httpRemoteUrl)
		}
	} else {
		return "", errors.New("Unknown error occurs")
	}

	return
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

	if gitRepoPath == "" {
		// Warn user that we're defaulting because no git repo was found
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Fprintf(color.Output, yellow("No GIT repository found, using default Gitlab API '%s'\n"), gitlabRootUrl)
	} else {
		// Extract origin remote url from repository config
		remoteUrl, err := getGitOriginRemoteUrl(gitRepoPath)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("Failed to find origin remote url in repository: %s", err), 5)
		}

		// Check if we can use the origin remote url
		if remoteUrl != "" {
			// Guess gitlab url based on remote url
			gitlabRootUrl, err = guessGitlabAPIFromGitRemoteUrl(remoteUrl)
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("No valid and responding Gitlab API URL found from repository's origin remote: %s", err), 5)
			}
		} else {
			// Warn user that we're defaulting because no origin remote was found
			yellow := color.New(color.FgYellow).SprintFunc()
			fmt.Fprintf(color.Output, yellow("No origin remote found in repository, using default Gitlab API '%s'\n"), gitlabRootUrl)
		}
	}

	fmt.Printf("Validating %s... ", relativeGitlabCiFilePath)

	if verboseMode {
		fmt.Printf("\n")
	}

	// Call the API to validate the gitlab-ci file
	ciFileContent, err := ioutil.ReadFile(gitlabCiFilePath)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Error while reading '%s' file content: %s", relativeGitlabCiFilePath, err), 5)
	}

	status, errorMessages, err := lintGitlabCIUsingAPI(gitlabRootUrl, string(ciFileContent))
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Error querying Gitlab API '%s' for CI lint: %s", gitlabRootUrl, err), 5)
	}

	if !status {
		if verboseMode {
			fmt.Printf("%s ", relativeGitlabCiFilePath)
		}

		red := color.New(color.FgRed).SprintFunc()
		fmt.Fprintf(color.Output, "%s\n", red("KO"))

		messages := red(strings.Join(errorMessages, "\n"))
		os.Stderr.WriteString(fmt.Sprintf("%s\n", messages))

		return cli.NewExitError("", 10)
	}

	if verboseMode {
		fmt.Printf("%s ", relativeGitlabCiFilePath)
	}
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Fprintf(color.Output, "%s\n", green("OK"))

	return nil
}

func createGitHookLink(gitRepoPath string, hookName string) (int, error) {
	currentExe, err := os.Executable()
	if err != nil {
		return HookError, err
	}

	err = os.MkdirAll(path.Join(gitRepoPath, "hooks"), 0755)
	if err != nil {
		return HookError, err
	}

	hookPath := path.Join(gitRepoPath, "hooks", hookName)

	// There is no hook already
	fi, err := os.Lstat(hookPath)
	if os.IsNotExist(err) {
		err = os.Symlink(currentExe, hookPath)
		if err != nil {
			return HookError, err
		}
	} else {
		// If there is a hook, maybe it's already ourself?
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			return HookAlreadyExists, nil
		} else {
			linkDest, err := os.Readlink(hookPath)
			if err != nil {
				return HookError, err
			}

			linkDest, err = filepath.Abs(linkDest)
			if err != nil {
				return HookError, err
			}

			if linkDest == currentExe {
				return HookAlreadyCreated, nil
			} else {
				return HookAlreadyExists, nil
			}
		}
	}

	return HookCreated, nil
}

func deleteGitHookLink(gitRepoPath string, hookName string) (int, error) {
	hookPath := path.Join(gitRepoPath, "hooks", hookName)

	fi, err := os.Lstat(hookPath)
	if os.IsNotExist(err) {
		return HookNotExisting, nil
	} else {
		currentExe, err := os.Executable()
		if err != nil {
			return HookError, err
		}
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			return HookNotMatching, nil
		} else {
			linkDest, err := os.Readlink(hookPath)
			if err != nil {
				return HookError, err
			}

			linkDest, err = filepath.Abs(linkDest)
			if err != nil {
				return HookError, err
			}

			if verboseMode {
				fmt.Println(linkDest)
			}

			if linkDest == currentExe {
				err = os.Remove(hookPath)
				if err != nil {
					return HookError, err
				}
			} else {
				return HookNotMatching, nil
			}
		}

	}

	return HookDeleted, nil
}

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
		return cli.NewExitError(fmt.Sprintf("No GIT repository found, can't install a hook"), 5)
	}
	if verboseMode {
		fmt.Printf("Git repository found: %s\n", gitRepoPath)
	}

	status, err := deleteGitHookLink(gitRepoPath, "pre-commit")
	if err != nil {
		return cli.NewExitError(err, 5)
	}
	switch status {
	case HookNotMatching:
		red := color.New(color.FgRed).SprintFunc()
		msg := fmt.Sprintf(red("Unknown pre-commit hook\nPlease uninstall manually."))
		return cli.NewExitError(msg, 4)
	case HookNotExisting:
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Fprintf(color.Output, yellow("No pre-commit hook found.\n"))
	case HookDeleted:
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Fprintf(color.Output, green("Git pre-commit hook uinstalled.\n"))
	default:
		return cli.NewExitError("Unkown error", 5)
	}

	return nil
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

	pathArgumentDescription := `If PATH if given, it will depending of its type on filesystem:
    - if a file, it will be used as the gitlab-ci file to check (similar to global --ci-file option)
    - if a directory, it will be used as the folder from where to search for a ci file and a git repository (similar to global --directory option)
   PATH have precedence over --ci-file and --directory options.`

	app.ArgsUsage = "[PATH]"
	app.Description = pathArgumentDescription
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "gitlab-url,u",
			Value:       defaultGitlabRootUrl,
			Usage:       "root `URL` of the Gitlab instance to use API",
			EnvVars:     []string{"GCL_GITLAB_URL"},
			Destination: &gitlabRootUrl,
		},
		&cli.StringFlag{
			Name:        "ci-file,f",
			Usage:       "`FILE` is the relative or absolute path to the gitlab-ci file",
			EnvVars:     []string{"GCL_GITLAB_CI_FILE"},
			Destination: &gitlabCiFilePath,
		},
		&cli.StringFlag{
			Name:        "directory,d",
			Value:       ".",
			Usage:       "`DIR` is the directory from where to search for gitlab-ci file and git repository",
			EnvVars:     []string{"GCL_DIRECTORY"},
			Destination: &directoryRoot,
		},
		&cli.StringFlag{
			Name:        "personal-access-token,p",
			Value:       "",
			Usage:       "personal access token `TOK` for accessing repositories when you have 2FA enabled",
			EnvVars:     []string{"GCL_PERSONAL_ACCESS_TOKEN"},
			Destination: &personalAccessToken,
		},
		&cli.UintFlag{
			Name:        "timeout,t",
			Value:       httpRequestTimeout,
			Usage:       "timeout in second after which http request to Gitlab API will timeout (and the program will fails)",
			EnvVars:     []string{"GCL_TIMEOUT"},
			Destination: &httpRequestTimeout,
		},
		&cli.BoolFlag{
			Name:    "no-color,n",
			Usage:   "don't color output. By defaults the output is colorized if a compatible terminal is detected.",
			EnvVars: []string{"GCL_NOCOLOR"},
		},
		&cli.BoolFlag{
			Name:        "verbose,v",
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
		}

		if !colorMode {
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

		return nil
	}

	app.Action = func(c *cli.Context) error {
		return commandCheck(c)
	}

	app.Run(os.Args)
}

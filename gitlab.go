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
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.com/orobardet/gitlab-ci-linter/config"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Filename of a gitlab-ci file. Used to find the gitlab-ci file if no path are given at calls
const gitlabCiFileName = ".gitlab-ci.yml"

// Default Gitlab instance URL to use
const defaultGitlabRootURL = "https://gitlab.com"

// Path of the Gitlab CI lint API, to be used on the root url
const gitlabAPICiLintPath = "/api/v4/ci/lint"

// GitlabAPILintRequest struct represents the JSON body of a request sent to the Gitlab API /ci/lint
type GitlabAPILintRequest struct {
	Content string `json:"content"`
}

// GitlabAPILintResponse struct represents the JSON body of a response from the Gitlab API /ci/lint
type GitlabAPILintResponse struct {
	Status     string   `json:"status,omitempty"`
	Error      string   `json:"error,omitempty"`
	Errors     []string `json:"errors,omitempty"`
	MergedYaml string   `json:"merged_yaml,omitempty"`
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
	}

	return findGitlabCiFile(filepath.Dir(directory))
}

func initGitlabHTTPClientRequest(method string, url string, content string) (*http.Client, *http.Request, error) {
	var httpClient *http.Client
	var req *http.Request

	httpClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     false,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: time.Second * time.Duration(httpRequestTimeout),
	}

	req, err := http.NewRequest(method, url, strings.NewReader(content))
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", config.APPNAME, config.VERSION))
	if personalAccessToken != "" {
		req.Header.Add("PRIVATE-TOKEN", personalAccessToken)
	}

	return httpClient, req, err
}

// Check if we can get a response with the rootUrl on the API CI Lint endpoint, and if a redirection occurs
// If a redirection is detected, return the redirected root URL.
// This is needed as redirection response only occurs when the API is call using en HTTP GET, but in the en the API
// has to be called in POST
func checkGitlabAPIUrl(rootURL string) (string, error) {

	newRootURL := rootURL

	lintURL := rootURL + gitlabAPICiLintPath

	if verboseMode {
		fmt.Printf("Checking '%s' (using '%s')...\n", rootURL, lintURL)
	}

	httpClient, req, err := initGitlabHTTPClientRequest("GET", lintURL, "")
	if err != nil {
		return newRootURL, fmt.Errorf("Unable to create an HTTP client: %w", err)
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		fmt.Printf("%+v\n", req.Header)
		return newRootURL, fmt.Errorf("HTTP request error: %w", err)
	}
	defer resp.Body.Close()

	// Getting the full URL used for the last query, after following potential redirection
	lastURL := resp.Request.URL.String()

	// Let's try to get the redirected root URL by removing the gitlab API path from the last use URL
	lastRootURL := strings.TrimSuffix(lastURL, gitlabAPICiLintPath)
	// If the result is not empty or unchanged, it means
	if lastRootURL != "" && lastRootURL != lastURL {
		newRootURL = lastRootURL
	}

	if verboseMode {
		fmt.Printf("Url '%s' validated\n", newRootURL)
	}

	return newRootURL, nil
}

// Send the content of a gitlab-ci file to a Gitlab instance lint API to check its validity
// In case of invalid, lint error messages are returned in `msgs`
func lintGitlabCIUsingAPI(rootURL string, ciFileContent string) (status bool, msgs []string, err error) {

	msgs = []string{}
	status = false

	// Prepare the JSON content of the POST request:
	// {
	//   "content": "<ESCAPED CONTENT OF THE GITLAB-CI FILE>"
	// }
	var reqParams = GitlabAPILintRequest{Content: ciFileContent}
	reqBody, _ := json.Marshal(reqParams)

	// Prepare requesting the API
	lintURL := fmt.Sprintf("%s%s?include_merged_yaml=%t", rootURL, gitlabAPICiLintPath, includeMergedYaml)
	if verboseMode {
		fmt.Printf("Querying %s...\n", lintURL)
	}
	httpClient, req, err := initGitlabHTTPClientRequest("POST", lintURL, string(reqBody))
	if err != nil {
		err = fmt.Errorf("Unable to create an HTTP client: %w", err)
		return
	}

	// Make the request to the API
	resp, err := httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("HTTP request error: %w", err)
		return
	}
	defer resp.Body.Close()

	// Get the results
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("Unable to parse response: %w", err)
		return
	}
	var result GitlabAPILintResponse
	err = json.Unmarshal([]byte(body), &result)
	if err != nil {
		err = fmt.Errorf("Unable to parse JSON response: %w", err)
		return
	}

	if result.MergedYaml != "" {
		fmt.Printf("Merged yaml: %s\n", result.MergedYaml)
	}

	// Analyse the results
	if result.Status == "valid" {
		status = true
		err = nil
		return
	}

	if result.Status == "invalid" {
		msgs = result.Errors
	}

	if result.Error != "" {
		err = fmt.Errorf("API respond  %s", result.Error)
	}

	return
}

func guessGitlabAPIFromGitRemoteURL(remoteURL string) (apiRootURL string, err error) {
	httpRemoteURL, err := checkGitlabAPIUrl(httpiseRemoteURL(remoteURL))
	if err != nil {
		return "", err
	}
	if httpRemoteURL != "" {
		apiRootURL = httpRemoteURL
		if verboseMode {
			fmt.Printf("API url found: %s\n", httpRemoteURL)
		}
	} else {
		return "", errors.New("Unknown error occurs")
	}

	return
}

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
	"io/ioutil"
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
const defaultGitlabRootUrl = "https://gitlab.com"

// Path of the Gitlab CI lint API, to be used on the root url
const gitlabApiCiLintPath = "/api/v4/ci/lint"

type GitlabAPILintRequest struct {
	Content string `json:"content"`
}

type GitlabAPILintResponse struct {
	Status string   `json:"status,omitempty"`
	Error  string   `json:"error,omitempty"`
	Errors []string `json:"errors,omitempty"`
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

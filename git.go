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
	"errors"
	"github.com/go-ini/ini"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

// Name of the git repo directory
const gitRepoDirectory = ".git"

// Name of the git repo config file in a git repo directory
const gitRepoConfigFilename = "config"

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
	}

	return findGitRepo(filepath.Dir(directory))
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
func getGitOriginRemoteURL(gitDirectory string) (string, error) {
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
func httpiseRemoteURL(remoteURL string) string {
	re := regexp.MustCompile("^(https?://[^/]*).*$")
	if re.MatchString(remoteURL) { // http remote
		matches := re.FindStringSubmatch(remoteURL)
		if len(matches) >= 2 {
			return matches[1]
		}
	} else { // ssh remote
		re = regexp.MustCompile("^([^@]*@)?([^:]+)")
		matches := re.FindStringSubmatch(remoteURL)
		if len(matches) >= 3 {
			return "https://" + matches[2]
		}
	}

	return ""
}

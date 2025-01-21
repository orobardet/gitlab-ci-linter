package main

import (
	"net/url"
	"os"
	"runtime"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/mitchellh/go-homedir"
)

// Search for a token in netrc
// A gitlab token is used from the account of named machine (NOT the "default" entry), and only if the login is empty
// or not defined.
// e.g.: for gitlab.com, the .netrc entry should be:
//
//	 machine gitlab.com
//	   # possible login and password
//		account MY_PERSONAL_ACCESS_TOKEN
//
// This is because Gitlab does not allow token in basic auth, so token must be sent as HTTP Header.
// To allow someone to use basic auth in other tools using .netrc, without conflicting with access token
func getGitlabTokenFromNetrc(gitlabURL string) (string, error) {
	u, err := url.Parse(gitlabURL)
	if err != nil {
		return "", err
	}
	fqdn := u.Hostname()
	if fqdn == "" {
		return "", nil
	}

	path, err := getNetrcFilePath()
	if err != nil {
		return "", err
	}

	if path == "" {
		return "", nil
	}

	machine, err := netrc.FindMachine(path, fqdn)
	if err != nil {
		return "", nil
	}
	if !machine.IsDefault() && machine.Account != "" {
		return machine.Account, nil
	}

	return "", nil
}

// Returns the .netrc file path to load
// If an empty string is returned with no error, it means
func getNetrcFilePath() (string, error) {
	path := netrcFile
	if path == "" {
		path = os.Getenv("NETRC")
		if path == "" {
			filename := ".netrc"
			if runtime.GOOS == "windows" {
				filename = "_netrc"
			}

			var err error
			path, err = homedir.Expand("~/" + filename)
			if err != nil {
				return "", err
			}
		}
	}

	// If the path is not a file, then do nothing
	if fi, err := os.Stat(path); err != nil {
		// File doesn't exist, do nothing
		if os.IsNotExist(err) {
			return "", nil
		}

		// Some other error!
		return "", err
	} else if fi.IsDir() {
		// File is directory, ignore
		return "", nil
	}

	return path, nil
}

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
	"os"
	"path"
	"path/filepath"
)

const (
	HookError = iota
	HookCreated
	HookAlreadyCreated
	HookAlreadyExists
	HookDeleted
	HookNotExisting
	HookNotMatching
)

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

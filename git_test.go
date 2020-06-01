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
	"testing"
)

var httpiseRemoteURLData = [][]string{
	{"http://gitlab.com", "http://gitlab.com"},
	{"https://gitlab.com", "https://gitlab.com"},
	{"http://gitlab.com/my/project", "http://gitlab.com"},
	{"https://gitlab.com/my/project", "https://gitlab.com"},
	{"http://my.company-forge.com/", "http://my.company-forge.com"},
	{"https://my.company-forge.com", "https://my.company-forge.com"},
	{"http://my.company-forge.com/my/project", "http://my.company-forge.com"},
	{"https://my.company-forge.com/my/project", "https://my.company-forge.com"},
	{"git@gitlab.com", "https://gitlab.com"},
	{"git@gitlab.com:my/project", "https://gitlab.com"},
	{"git@my.company-forge.com", "https://my.company-forge.com"},
	{"git@my.company-forge.com:my/project", "https://my.company-forge.com"},
	{"gitlab.com", "https://gitlab.com"},
	{"gitlab.com:my/project", "https://gitlab.com"},
	{"my.company-forge.com", "https://my.company-forge.com"},
	{"my.company-forge.com:my/project", "https://my.company-forge.com"},
	{"", ""},
}

func TestHttpiseRemoteUrl(t *testing.T) {

	for _, testData := range httpiseRemoteURLData {
		params := testData[0]
		expectedResult := testData[1]
		t.Run("url="+params, func(t *testing.T) {
			result := httpiseRemoteURL(params)
			if result != expectedResult {
				t.Errorf("received '%s' while expecting '%s'", result, expectedResult)
			}
		})
	}
}

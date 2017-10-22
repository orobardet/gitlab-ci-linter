package main

import (
	"testing"
)

var httpiseRemoteUrl_Data = [][]string{
	[]string{"http://gitlab.com", "http://gitlab.com"},
	[]string{"https://gitlab.com", "https://gitlab.com"},
	[]string{"http://gitlab.com/my/project", "http://gitlab.com"},
	[]string{"https://gitlab.com/my/project", "https://gitlab.com"},
	[]string{"http://my.company-forge.com/", "http://my.company-forge.com"},
	[]string{"https://my.company-forge.com", "https://my.company-forge.com"},
	[]string{"http://my.company-forge.com/my/project", "http://my.company-forge.com"},
	[]string{"https://my.company-forge.com/my/project", "https://my.company-forge.com"},
	[]string{"git@gitlab.com", "http://gitlab.com"},
	[]string{"git@gitlab.com:my/project", "http://gitlab.com"},
	[]string{"git@my.company-forge.com", "http://my.company-forge.com"},
	[]string{"git@my.company-forge.com:my/project", "http://my.company-forge.com"},
	[]string{"gitlab.com", "http://gitlab.com"},
	[]string{"gitlab.com:my/project", "http://gitlab.com"},
	[]string{"my.company-forge.com", "http://my.company-forge.com"},
	[]string{"my.company-forge.com:my/project", "http://my.company-forge.com"},
}

func TestHttpiseRemoteUrl(t *testing.T) {

	for _, testData := range httpiseRemoteUrl_Data {
		params := testData[0]
		expectedResult := testData[1]
		t.Run("url="+params, func(t *testing.T) {
			result := httpiseRemoteUrl(params)
			if result != expectedResult {
				t.Errorf("received '%s' while expecting '%s'", result, expectedResult)
			}
		})
	}
}

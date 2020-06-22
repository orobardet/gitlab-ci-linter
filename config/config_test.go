package config_test

import (
	"github.com/stretchr/testify/assert"
	"gitlab.com/orobardet/gitlab-ci-linter/config"
	"testing"
)

func Test_APPNAME(t *testing.T) {
	asserter := assert.New(t)
	asserter.NotEmpty(config.APPNAME)
}

func Test_VERSION(t *testing.T) {
	asserter := assert.New(t)
	asserter.NotEmpty(config.VERSION)
}

func Test_REVISION(t *testing.T) {
	asserter := assert.New(t)
	asserter.NotEmpty(config.REVISION)
}

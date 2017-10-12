# This is how we want to name the binary output
SOURCEDIR=.

BINARY?=.build/lint-gitlab-ci

VERSION?=$(shell git describe --tags --always --match=v* 2> /dev/null || cat ${SOURCEDIR}/VERSION 2> /dev/null || echo v0.0.0)-dev
REVISION?=$(shell git rev-parse HEAD)
BUILD_TIME?=`date +%FT%T%z`

LDFLAGS=--X main.VERSION=${VERSION} -X main.REVISION=${REVISION} -X main.BUILD_TIME=${BUILD_TIME}

SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(SOURCES)
	go build -ldflags "${LDFLAGS}" -o ${BINARY}

.PHONY: install
install:
	go install -ldflags "${LDFLAGS}" ./...

.PHONY: clean
clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
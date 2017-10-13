# Targets:
#
# - all: default, just match 'build' target
# - build: go build the program
# - clean: remove the binary if it exists
# - rebuild: force the rebuild by running clean and then build
# - run: run the builded binary (build it if needed). The environment variable `RUNARGS` can be used to pas arguments to the binary
# - install: go install the program

SOURCEDIR=.

# Output path and name of the program binary result
BINARY?=.build/lint-gitlab-ci

# Use the environnement variable to pass arguments to the program when using `make run`
# e.g.:
# `RUNARGS="--help install" make run`
# will run (after building if necessary):
# `$(BINARY) --help install`
RUNARGS?=

# Version number to use when building the program
VERSION?=$(shell git describe --tags --always --match=v* 2> /dev/null || cat ${SOURCEDIR}/VERSION 2> /dev/null || echo v0.0.0)-dev
# Revision or VCS hash to use when building the program. Can be long, it may be truncated by the program at
REVISION?=$(shell git rev-parse HEAD)
# Build date&time to use when building the programme. Unlikely needed to be overriden.
BUILD_TIME?=`date +%FT%T%z`


LDFLAGS=--X main.VERSION=${VERSION} -X main.REVISION=${REVISION} -X main.BUILD_TIME=${BUILD_TIME}
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

.DEFAULT_GOAL: all


all: build

build: $(BINARY)

rebuild: clean build

run: $(BINARY)
	$(BINARY) $(RUNARGS)

$(BINARY): $(SOURCES)
	go build -ldflags "${LDFLAGS}" -o ${BINARY}

.PHONY: install
install:
	go install -ldflags "${LDFLAGS}" ./...

.PHONY: clean
clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
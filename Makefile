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
BINARY?=$(if $(filter $(OS),Windows_NT),.build/gitlab-ci-linter.exe,.build/gitlab-ci-linter)

# Use the environnement variable to pass arguments to the program when using `make run`
# e.g.:
# `RUNARGS="-version" make run`
# or
# `make run version`
# will run (after building if necessary):
# `$(BINARY) version`
# If the first argument is "run"...
ifeq (run,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "run"
  RUNARGS?=$(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(RUNARGS):;@:)
endif

DEBUG:=0

# Version number to use when building the program
VERSION?=$(shell git describe --tags --always --match=v* 2> /dev/null || cat ${SOURCEDIR}/VERSION 2> /dev/null || echo v0.0.0)-dev
# Revision or VCS hash to use when building the program. Can be long, it may be truncated by the program at
REVISION?=$(shell git rev-parse HEAD)
# Build date&time to use when building the programme. Unlikely needed to be overriden.
BUILDTIME?=$(shell date +%FT%T%z)


LDFLAGS+=-X main.VERSION=${VERSION} -X main.REVISION=${REVISION} -X main.BUILDTIME=${BUILDTIME}
ifeq ($(DEBUG),0)
  LDFLAGS+=-s -w
endif
SOURCES := $(shell find $(SOURCEDIR) -name '*.go' -not -path "$(SOURCEDIR)/vendor/*" -not -name '*_test.go')
TESTSOURCES := $(shell find $(SOURCEDIR) -name '*_test.go' -not -path "$(SOURCEDIR)/vendor/*")

.DEFAULT_GOAL: all


all: build

build: $(BINARY)

rebuild: clean build

test: $(TESTSOURCES)
	go test

.PHONY: run
run: $(BINARY)
	$(BINARY) $(RUNARGS)

$(BINARY): $(SOURCES)
	go build -ldflags "${LDFLAGS}" -o ${BINARY}

.PHONY: install
install:
	go install -ldflags "${LDFLAGS}" ./...

.PHONY: clean
clean:
	go clean
	if [ -f "$(BINARY)" ] ; then rm $(BINARY) ; fi

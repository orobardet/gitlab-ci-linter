# Olivier Robardet's Go Makefile v1.0.0
#
# Targets:
#
# - setup: install dev dependencies (needs a valid and workin Go environment)
# - imports: list all direct imports for all the packages of the application
# - deps: list all dependencies (imports recursively) for all the packages of the application
# - all: default, just match 'build' target
# - build: go build the program
# - clean: remove the binary if it exists
# - rebuild: force the rebuild by running clean and then build
# - run: run the builded binary (build it if needed). The environment variable `RUNARGS` can be used to pas arguments to the binary
# - install: go install the program
# - vet: go vet of all source files
# - fmt: go fmt check (no modification) of all source files
# - checkstyle: golint to stylistic lint all souce files
# - cyclo: gocyclo all source files
# - secucheck: gosecu all source files
# - test: run tests with coverage.
#         Coverage reports are generated in a directory that can be overrided using the `COVERAGEREPORTDIR` envvar
# - html-cover: run tests and generate html reports for test coverage.
#               HTML coverage reports are generated in the same directory as plain coverage reports
# - check: vet + fmt + checkstyle + cyclo + secucheck

export GO111MODULE?=on

SOURCEDIR=.
BUILDDIR?=.build
DOCBUILDDIR?=$(BUILDDIR)/godoc
# Program name. Defaults to the name of the local package any, or the directory name
PROGRAM_NAME?=$(shell basename `go list . 2> /dev/null` || echo "`basename $(shell pwd)`")

# Output path and name of the program binary result
BINARY?=$(if $(filter $(OS),Windows_NT),$(BUILDDIR)/$(PROGRAM_NAME).exe,$(BUILDDIR)/$(PROGRAM_NAME))

# Use the environnement variable to pass arguments to the program when using `make run`
# e.g.:
# `RUNARGS="-version" make run`
# or
# `make run version`
# will run (after building if necessary):
# `$(BINARY) version`
# If the first argument is "run"...
#ifeq (run,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "run"
#  RUNARGS?=$(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
#  $(eval $(RUNARGS):;@:)
#endif

DEBUG:=0

# Version number to use when building the program
VERSION?=$(shell git describe --tags --match=v* 2> /dev/null || cat ${SOURCEDIR}/VERSION 2> /dev/null || echo v0.0.0)+dev
# Revision or VCS hash to use when building the program. Can be long, it may be truncated by the program at
REVISION?=$(shell git rev-parse HEAD || echo "")
# Build date&time to use when building the programme. Unlikely needed to be overriden.
BUILDTIME?=$(shell date +%FT%T%z)

MAIN_PACKAGE_PATH=$(shell go list . 2> /dev/null)/
LDFLAGS+=-X $(MAIN_PACKAGE_PATH)config.VERSION=${VERSION} -X $(MAIN_PACKAGE_PATH)config.REVISION=${REVISION} -X $(MAIN_PACKAGE_PATH)config.BUILDTIME=${BUILDTIME}
ifeq ($(DEBUG),0)
  LDFLAGS+=-s -w
endif
SOURCES:=$(shell find $(SOURCEDIR) -name '*.go' -not -path "$(SOURCEDIR)/vendor/*" 2>/dev/null)
PACKAGES:=$(shell go list ./...)
PACKAGEPATHS:=$(shell go list -f "{{.Dir}}" ./...)
CYCLOTHRESHOLD?=15

GOTESTCMD?=go test
GOTESTFLAGS?=
COVERAGEREPORTDIR?=$(BUILDDIR)/coverage
COVERAGEMODE?=atomic
COVERAGEGLOBALFILE?=all.cover

SOURCES:=$(shell find $(SOURCEDIR) -name '*.go' -not -path "$(SOURCEDIR)/vendor/*" 2>/dev/null)
PACKAGES:=$(shell go list ./...)
PACKAGEPATHS:=$(shell go list -f "{{.Dir}}" ./...)

.DEFAULT_GOAL: all

.PHONY: all
all: build

.PHONY: has-depends _godoc_binary _golint_binary _gocyclo_binary _gosec_binary _upx_binary

_godoc_binary:
	$(eval GODOC:=$(shell command -v godoc 2> /dev/null))
	@test -n "$(GODOC)" || echo "No godoc binary found, please install it: go get -u golang.org/x/tools/cmd/godoc"
_golint_binary:
	$(eval GOLINT:=$(shell command -v golint 2> /dev/null))
	@test -n "$(GOLINT)" || echo "No golint binary found, please install it: go get -u golang.org/x/lint/golint"
_gocyclo_binary:
	$(eval GOCYCLO:=$(shell command -v gocyclo 2> /dev/null))
	@test -n "$(GOCYCLO)" || echo "No gocyclo binary found, please install it: go get -u github.com/alecthomas/gocyclo"
_gosec_binary:
	$(eval GOSEC:=$(shell command -v gosec 2> /dev/null))
	@test -n "$(GOSEC)" || echo "No gosec binary found, please install it: go get -u github.com/securego/gosec/cmd/gosec/..."
_upx_binary:
	$(eval UPX:=$(shell command -v upx 2> /dev/null))
	@test -n "$(UPX)" || echo "No upx binary found, please install it: https://upx.github.io/"

has-depends: _godoc_binary _golint_binary _gocyclo_binary _gosec_binary _upx_binary

build: $(BINARY)

.PHONY: release

release: _upx_binary build
	$(UPX) -9f --color $(BINARY)

rebuild: clean build

.PHONY: setup
setup:
	cd $(GOPATH)
	go get -u golang.org/x/sys golang.org/x/text golang.org/x/lint/golint github.com/alecthomas/gocyclo github.com/securego/gosec/cmd/gosec

.PHONY: imports
imports:
	@go list -f '{{ join .Imports "\n" }}' ./... | sort -u

.PHONY: deps
deps:
	@go list -f '{{ join .Deps "\n" }}' ./... | sort -u

.PHONY: check checks
check: fmt vet checkstyle cyclo secucheck test
checks: check

.PHONY: fmt dofmt
dofmt: $(SOURCES)
	gofmt -s -l -e -w $(SOURCES)

.PHONY: fmt
fmt: $(SOURCES)
	@echo gofmt -s -l -e -d $(SOURCES)
	@gofmtoutput=$$(gofmt -s -l -e -d $(SOURCES)) && echo "$$gofmtoutput" && [ -z "$$gofmtoutput" ]

.PHONY: fmt
vet: $(SOURCES)
	go vet $(PACKAGES)

.PHONY: checkstyle
checkstyle: $(SOURCES) | _golint_binary
	$(GOLINT) -set_exit_status $(PACKAGEPATHS)

.PHONY: cyclo
cyclo: $(SOURCES) | _gocyclo_binary
	$(GOCYCLO) -avg -over $(CYCLOTHRESHOLD) $(PACKAGEPATHS)

.PHONY: secucheck
secucheck: $(SOURCES) | _gosec_binary
	$(GOSEC) -fmt golint -quiet -log /dev/stdout ./...

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
	rm -fr $(COVERAGEREPORTDIR)
	rm -fr $(DOCBUILDDIR)
	-rmdir $(BUILDDIR) 2>/dev/null || true

$(COVERAGEREPORTDIR):
	mkdir -p $(COVERAGEREPORTDIR)

.PHONY: test _test
test: _test | $(COVERAGEREPORTDIR)
_test: $(addprefix test-package/,$(PACKAGES))
	@echo "Merging all coverage reports"
	@echo "mode: $(COVERAGEMODE)" > $(COVERAGEREPORTDIR)/$(COVERAGEGLOBALFILE)
	@find $(COVERAGEREPORTDIR) -name "*.cover" ! -name "$(COVERAGEGLOBALFILE)" -exec cat {} \; 2>/dev/null | grep -v "^mode:" >> $(COVERAGEREPORTDIR)/$(COVERAGEGLOBALFILE)
	go tool cover -func $(COVERAGEREPORTDIR)/$(COVERAGEGLOBALFILE)

test-package/%: pkg_path=$*
test-package/%: pkg_name=$(shell go list -f '{{.Name}}' $(pkg_path))
test-package/%: | $(COVERAGEREPORTDIR)
	$(GOTESTCMD) -cover -covermode=$(COVERAGEMODE) -coverprofile=$(COVERAGEREPORTDIR)/pkg-$(pkg_name).cover $(GOTESTFLAGS) $(pkg_path)

.PHONY: html-cover _html-cover
.SECONDEXPANSION:
COVERFILES=$(shell find $(COVERAGEREPORTDIR) -name '*.cover' -printf "%f\n" 2>/dev/null)
html-cover: _html-cover | $(COVERAGEREPORTDIR)
_html-cover: $(addprefix html-cover-package/,$(COVERFILES))

html-cover-package/%: | $(COVERAGEREPORTDIR)
	go tool cover -html=$(COVERAGEREPORTDIR)/$* -o $(COVERAGEREPORTDIR)/$(*:.cover=.html)

.PHONY: godoc
godoc: _godoc_binary
	ci/make-godoc.sh

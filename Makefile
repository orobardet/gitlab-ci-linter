# This is how we want to name the binary output
BINARY?=.build/lint-gitlab-ci

VERSION?=`cat VERSION`
REVISION?=`git rev-parse HEAD`
BUILD_TIME?=`date +%FT%T%z`

LDFLAGS=--X main.VERSION=${VERSION} -X main.REVISION=${REVISION} -X main.BUILD_TIME=${BUILD_TIME}

all:
	go build -ldflags "${LDFLAGS}" -o ${BINARY}
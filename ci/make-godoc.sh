#!/usr/bin/env bash

set -e

SOURCE_DIR=${SOURCE_DIR:=$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)}
DOCBUILDDIR=${DOCBUILDDIR:=$SOURCE_DIR/.build/godoc}

CRESET='\e[0m' # Reset color
CGREEN='\e[1;32m'
CCYAN='\e[1;36m'

cd "$SOURCE_DIR"

echo "Will generate godoc in $DOCBUILDDIR directory"

MAIN_PACKAGENAME="$(go list .)"
PACKAGEPATH="${PACKAGEPATH:=gitlab.com/orobardet/gitlab-ci-linter}"

rm -fr "$DOCBUILDDIR"

mkdir -p "$DOCBUILDDIR"
cd "$DOCBUILDDIR"
echo -e "${CCYAN}Starting godoc server on port 8989...${CRESET}"
godoc -http "127.0.0.1:8989" &
GODOCPID=$!
echo -e "${CCYAN}Give some time to godoc server to be ready...${CRESET}"
sleep 10
echo -e "${CCYAN}Crawling $PACKAGEPATH doc file...${CRESET}"
set +e
wget \
    --convert-links \
    --recursive \
    --page-requisites \
    --no-parent \
    --no-host-directories \
    --adjust-extension \
    --exclude-directories '/src/'$PACKAGEPATH'/.git,/src/'$PACKAGEPATH'/.*,/src/'$PACKAGEPATH'/vendor,/src/'$PACKAGEPATH'/coverage,/src/'$PACKAGEPATH'/doc,/src/'$PACKAGEPATH'/tools' \
    -e robots=off \
    -nv \
    -U mozilla \
    http://localhost:8989/src/$PACKAGEPATH \
    http://localhost:8989/pkg/$PACKAGEPATH
kill $GODOCPID
set -e

echo -e "${CCYAN}Post processing godoc...${CRESET}"

# Replace the package's directory index file by the package file
mv "pkg/$PACKAGEPATH.html" "pkg/$(dirname "$PACKAGEPATH")/index.html"

# Ugly hack to hide menu and footer in godocs, that points to
echo "div#menu,div#footer { display: none; }" >> "lib/godoc/style.css"

# Make an index file at the root of the generated doc tree, that redirect the the package's index
echo "<!DOCTYPE html><html><head><meta http-equiv=\"refresh\" content=\"0;pkg/$(dirname "$PACKAGEPATH")/index.html\"></head></html>" > index.html

echo -e "\n\t${CGREEN}Package documentation successfully generated in $DOCBUILDDIR ($(du -sh "." | cut -f1))${CRESET}\n"
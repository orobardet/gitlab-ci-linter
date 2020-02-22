#!/usr/bin/env bash

set -e

CHANGELOG_FILE=${CHANGELOG_FILE:=CHANGELOG.md}
VERSION=${VERSION##v}
AF_BINARY_URL=${AF_BINARY_URL:=https://api.bintray.com}
AF_BINARY_DL_URL=${AF_BINARY_DL_URL:=https://dl.bintray.com}

MARKDOWN=""

if [[ -z "$AF_BINARY_URL" ]] ; then
    >&2 echo "No Artifactory/bintray API url found (\$AF_BINARY_URL)."
    exit 1
fi
if [[ -z "$AF_BINARY_DL_URL" ]] ; then
    >&2 echo "No Artifactory/bintray download url found (\$AF_BINARY_DL_URL)."
    exit 1
fi
if [[ -z "$AF_BINARY_SUBJECT" ]] ; then
    >&2 echo "No Artifactory/bintray subject found (\$AF_BINARY_SUBJECT)."
    exit 1
fi
if [[ -z "$AF_BINARY_REPO" ]] ; then
    >&2 echo "No Artifactory/bintray repository name found (\$AF_BINARY_REPO)."
    exit 1
fi
if [[ -z "$AF_BINARY_PACKAGE" ]] ; then
    >&2 echo "No Artifactory/bintray package name found (\$AF_BINARY_PACKAGE)."
    exit 1
fi
if [[ -z "$AF_API_USER" ]] ; then
    >&2 echo "No Artifactory/bintray API user found (\$AF_API_USER)."
    exit 1
fi
if [[ -z "$AF_API_KEY" ]] ; then
    >&2 echo "No Artifactory/bintray API key found (\$AF_API_KEY)."
    exit 1
fi

if [[ -z "$CI_GITLAB_URL" ]] ; then
    >&2 echo "No Gitlab URL found (\$CI_GITLAB_URL)."
    exit 1
fi
if [[ -z "$CI_PROJECT_PATH" ]] ; then
    >&2 echo "No project path found (\$CI_PROJECT_PATH)."
    exit 1
fi
if [[ -z "$CI_JOB_TOKEN" ]] ; then
    >&2 echo "No job token found (\$CI_JOB_TOKEN)."
    exit 1
fi

if [[ ! -f "$CHANGELOG_FILE" ]] ; then
    >&2 echo "No changelog file '$CHANGELOG_FILE' found (\$CHANGELOG_FILE)."
    exit 1
fi

if [[ -z "$CHANGELOG_FILE" ]] ; then
    >&2 echo "No version found (\$VERSION)."
    exit 1
fi

function escapeForJSON() {
    local l_string="$1" ; shift

    l_string=${l_string//\\/\\\\} # \
    l_string=${l_string//\"/\\\"} # "
    l_string=${l_string//$'\t'/\\t} # \t (tab)
    l_string=${l_string//$'\n'/\\n} # \n (newline)
    l_string=${l_string//$'\r'/\\r} # \r (carriage return)

    echo "$l_string"
}

rawurlencode() {
  local string="${1}"
  local strlen=${#string}
  local encoded=""
  local pos c o

  for (( pos=0 ; pos<strlen ; pos++ )); do
     c=${string:$pos:1}
     case "$c" in
        [-_.~a-zA-Z0-9]) o="${c}" ;;
        * ) printf -v o '%%%02x' "'$c"
     esac
     encoded+="${o}"
  done
  echo "${encoded}"    # You can either set a return variable (FASTER)
}

echo "Updating release note for $AF_BINARY_PACKAGE v$VERSION..."

echo "  Extracting changelog..."

# sed -e <Version Line> -e <Next version line> CHANGELOG.md | sed -e <remove starting and tailing empty lines>
MARKDOWN=$(sed -e '1,/^# v'$VERSION'/d' -e '/^#/,$d' CHANGELOG.md | sed -e :a -e '/./,$!d;/^\n*$/{$d;N;};/\n$/ba' )

MARKDOWN="$MARKDOWN"$'\n\n'

echo "  Retrieving binary files list from bintray..."

BINTRAY_FILES=$(curl -s "$AF_BINARY_URL/packages/$AF_BINARY_SUBJECT/$AF_BINARY_REPO/$AF_BINARY_PACKAGE/versions/$VERSION/files?include_unpublished=0" | jq -r 'sort_by(.name)[] | .name+":"+.path')

if [[ -n "$BINTRAY_FILES" ]] ; then
    MARKDOWN="$MARKDOWN**Download binaries:**"$'\n\n'
    for file in $BINTRAY_FILES ; do
        IFS=':' read name path <<< "$file"
        MARKDOWN="$MARKDOWN- [$name]($AF_BINARY_DL_URL/$AF_BINARY_SUBJECT/$AF_BINARY_PACKAGE/$path)"$'\n'
    done
fi

echo "  Generated release note:"
echo ">>RELEASE_NOTE_START>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
echo -e "$MARKDOWN"
echo "<<RELEASE_NOTE_END<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"

escapedMARKDOWN=$(escapeForJSON "$MARKDOWN")
printf -v l_jsonBody '{ "description":"%s" }' "$escapedMARKDOWN"
l_APIUrl="${CI_GITLAB_URL}/api/v4/projects/$(rawurlencode "$CI_PROJECT_PATH")/repository/tags/v$VERSION/release"
echo "  Updating Gitlab release ($l_APIUrl)..."
curl -s --header "PRIVATE-TOKEN: $CI_JOB_TOKEN" --header "Content-Type: application/json" -XPUT --data "$l_jsonBody" $l_APIUrl > /dev/null

echo "Release note for $AF_BINARY_PACKAGE v$VERSION updated."
#!/bin/bash

GITLAB_CI_YML_PATH=${GITLAB_CI_YML_PATH:=.gitlab-ci.yml}

RESC="\\033[0;0m"
C_F_CYAN="\\033[1;36m"
C_F_RED="\\033[1;31m"
C_F_GREEN="\\033[1;32m"
C_F_YELLOW="\\033[1;33m"

GITLAB_LINTER_ERROR=""

function checkIsGitRepo() {
    if ! GIT_ROOT_PATH=$(git rev-parse --show-toplevel 2>/dev/null) ; then
        >&2 echo -e "${C_F_RED}*** Not in a valid git repository${RESC}"
        exit 1
    fi
}

function checkIsGitlabCIYAML() {
    if [[ ! -f "${GIT_ROOT_PATH}/${GITLAB_CI_YML_PATH}" ]] ; then
        >&2 echo -e "${C_F_YELLOW}*** No Gitlab-CI file found (${GITLAB_CI_YML_PATH})${RESC}"
        exit 1
    fi
}

function escapeForJSON() {
    local l_string="$1" ; shift

    l_string=${l_string//\\/\\\\} # \
    l_string=${l_string//\"/\\\"} # "
    l_string=${l_string//$'\t'/\\t} # \t (tab)
    l_string=${l_string//$'\n'/\\n} # \n (newline)
    l_string=${l_string//$'\r'/\\r} # \r (carriage return)

    echo "$l_string"
}

function checkCILinterAPI() {
    local l_url="${1%%/*}" ; shift
    curl -L --header "Content-Type: application/json" -XPOST \
        "$l_url" \
        --data '{ "content" : "test:\n  script: test" }'
}

function guessGitlabAPIUrl() {
    local l_gitRemote=$(git remote get-url --push origin)
    local l_gitRootUrl=""

    if [[ "$l_gitRemote" =~ ^http:// || "$l_gitRemote" =~ ^https:// ]] ; then
        l_gitRootUrl=$(echo "$l_gitRemote" | sed -r 's|^(https?://[^/]*).*$|\1|')
    else
        l_gitRootUrl=${l_gitRemote#*@}
        l_gitRootUrl=${l_gitRootUrl%%:*}
        l_gitRootUrl="http://$l_gitRootUrl"
    fi

    echo $l_gitRootUrl
}

function validateGitlabCIYAML() {
    local l_gitlabUrlRoot=$1 ; shift
    local l_gitlabci=$1 ; shift

    local l_curlFlags=(-XPOST --header "Content-Type: application/json" -sw "\nHTTP-STATUS:%{response_code}")

    local l_ciContent=$(escapeForJSON "$(cat $l_gitlabci)")

    l_curlResult=$(curl -L "${l_curlFlags[@]}" \
        "$l_gitlabUrlRoot/api/v3/ci/lint" \
        --data "{\"content\" : \"$l_ciContent\"}" )

    l_lintAnswer="${l_curlResult%?HTTP-STATUS:*}"

    l_httpStatusCode=${l_curlResult##*HTTP-STATUS:}

    if [[ $l_httpStatusCode -ne 200 ]] ; then
        if [[ -n "$l_lintAnswer" ]] ; then
            GITLAB_LINTER_ERROR="$l_lintAnswer"
        else
            GITLAB_LINTER_ERROR="HTTP code $l_httpStatusCode"
        fi
        return 1
    fi

    l_linterStatus=$(echo "$l_lintAnswer" | jq -r .status)

    if [[ "$l_linterStatus" != "valid" ]] ; then
        GITLAB_LINTER_ERROR=$(echo "$l_lintAnswer" | jq -r  'if .errors then .errors | join("\n") else "Unknown error" end' | sed 's/^/- /')
        return 1
    fi

    return 0
}


checkIsGitRepo

checkIsGitlabCIYAML

guessGitlabAPIUrl
exit

echo -en "Validating ${GITLAB_CI_YML_PATH}..."
if validateGitlabCIYAML "https://code.search.orangeportails.net" "${GIT_ROOT_PATH}/${GITLAB_CI_YML_PATH}" ; then
    echo -e " ${C_F_GREEN}OK${RESC}"
else
    echo -e " ${C_F_RED}KO${RESC}"
    if [[ -n "$GITLAB_LINTER_ERROR" ]] ; then
        >&2 echo -e "${C_F_RED}$GITLAB_LINTER_ERROR${RESC}"
    else
        >&2 echo -e "${C_F_RED}Unknown error1${RESC}"
    fi
fi
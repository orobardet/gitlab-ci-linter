#!/bin/bash

# v1.0

GITLAB_CI_YML_PATH=${GITLAB_CI_YML_PATH:=.gitlab-ci.yml}
GITLAB_API_PATH_CI_LINT=${GITLAB_API_PATH_CI_LINT:=/api/v4/ci/lint}

ncolors=$(tput colors)
if test -n "$ncolors" && test $ncolors -ge 8; then
    RESC="\\033[0;0m"
    C_F_CYAN="\\033[1;36m"
    C_F_RED="\\033[1;31m"
    C_F_GREEN="\\033[1;32m"
    C_F_YELLOW="\\033[1;33m"
fi

GITLAB_LINTER_ERROR=""
SCRIPTPATH=$(cd $(dirname $0) ; pwd)/${0##*/}

function usage() {
    cat <<EOT

Check .gitlab-ci.yml syntax using Gitlab API.

Usage:
 ${0##*/} [--help|--install|--uninstall]

  -h, --help      : show this help
  -i, --install   : install as git pre-commit hook for the current repository
  -u, --uninstall : remove from git pre-commit hook for the current repository

Without any options, it will check the .gitlab-ci.yml of the current local repository.

The current working directory needs to be within a git repository, but not necessary at the repository root.
The git repository needs to have a remote 'origin' set to an instance of gitlab (ssh or http remote).

The script will try to validate only if a .gitlab-ci.yml file exists in the root of the repository.

--install will install the hook as a symbolic link the the current script. The install will failed if a pre-commit hook
already exists (and is not already a link to the script). In that case, you will have to install it manually.
Manual installation only require to add a call to the current script in your existing pre-commit hook.

--uninstall will only remove the pre-commit hook if it is a symbolic link to the current script. Otherwise, you will have
to uninstall it manually.
EOT
}

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
    local l_gitlabAPIUrl=""

    if [[ "$l_gitRemote" =~ ^http:// || "$l_gitRemote" =~ ^https:// ]] ; then
        # Remote is already using https
        l_gitRootUrl=$(echo "$l_gitRemote" | sed -r 's|^(https?://[^/]*).*$|\1|')
    else
        # Remote is using ssh, extract the FQDN
        l_gitRootUrl=${l_gitRemote#*@}
        l_gitRootUrl=${l_gitRootUrl%%:*}
        l_gitRootUrl="http://$l_gitRootUrl"
    fi

    # get the API URL after redirection
    if ! l_gitlabAPIUrl=$(curl -w "%{url_effective}\n" -I -L -s -S "${l_gitRootUrl}${GITLAB_API_PATH_CI_LINT}" -o /dev/null) ; then
        >&2 echo -e "${C_F_YELLOW}*** Unable to guess gitlab API from remote ${l_gitRemote}${RESC}"
        return 1
    fi

    if [[ -z "$l_gitlabAPIUrl" || ! "$l_gitlabAPIUrl" =~ ^http || ! "$l_gitlabAPIUrl" =~ ${GITLAB_API_PATH_CI_LINT}$ ]] ; then
        >&2 echo -e "${C_F_YELLOW}*** Unable to guess gitlab API from remote ${l_gitRemote}${RESC}"
        return 1
    fi

    echo $l_gitlabAPIUrl | sed -r 's|^(https?://[^/]*).*$|\1|'
}

function validateGitlabCIYAML() {
    local l_gitlabUrlRoot=$1 ; shift
    local l_gitlabci=$1 ; shift

    local l_curlFlags=(-XPOST --header "Content-Type: application/json" -sw "\nHTTP-STATUS:%{response_code}")

    local l_ciContent=$(escapeForJSON "$(cat $l_gitlabci)")

    l_curlResult=$(curl -L "${l_curlFlags[@]}" \
        "${l_gitlabUrlRoot}${GITLAB_API_PATH_CI_LINT}" \
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

function installGitHook() {
    checkIsGitRepo

    local l_hookPath="${GIT_ROOT_PATH}/.git/hooks/pre-commit";

    if guessGitlabAPIUrl > /dev/null ; then
        if [[ -e "$l_hookPath" ]] ; then
            if [[ -L "$l_hookPath" && $(readlink "$l_hookPath") = "$SCRIPTPATH" ]] ; then
                echo -e "${C_F_CYAN}Already installed.${RESC}"
                exit 0
            fi

            >&2 echo -e "${C_F_YELLOW}*** A pre-commit hook already exists${RESC} \nPlease install manually by adding '$SCRIPTPATH' in your pre-commit script."
            exit 4
        fi

        if [[ ! -d $(dirname $l_hookPath) ]] ; then
            mkdir -p $(dirname $l_hookPath)
        fi

        if ln -s "$SCRIPTPATH" "$l_hookPath" ; then
            find "$l_hookPath" -prune \( -type l -printf '%p -> %l\n' -o -printf '%p\n' \)
            echo -e "${C_F_GREEN}Git pre-commit hook installed!${RESC}"
            exit 0
        fi
    else
        exit 3
    fi
}

function uninstallGitHook() {
    checkIsGitRepo

    local l_hookPath="${GIT_ROOT_PATH}/.git/hooks/pre-commit";

    if guessGitlabAPIUrl > /dev/null ; then
        if [[ -e "$l_hookPath" ]] ; then
            if [[ -L "$l_hookPath" && $(readlink "$l_hookPath") = "$SCRIPTPATH" ]] ; then
                if rm "$l_hookPath" ; then
                    echo -e "${C_F_GREEN}Git pre-commit hook uinstalled!${RESC}"
                    exit 0
                fi
            fi

            >&2 echo -e "${C_F_RED}*** Unknown pre-commit hook${RESC} \nPlease uninstall manually."
            exit 4
        fi

        >&2 echo -e "${C_F_YELLOW}*** No pre-commit hook found${RESC}."
        exit 4
    else
        exit 3
    fi
}

function readOpts() {
    case "$1" in
        -h|--help) usage ; shift ; exit 0 ;;
        -i|--install) installGitHook ; shift ; exit 0 ;;
        -u|--uninstall) uninstallGitHook ; shift ; exit 0 ;;
    esac
    return 0
}

readOpts "$@"

checkIsGitRepo

checkIsGitlabCIYAML

if ! _GITLAB_API_ROOT_URL=$(guessGitlabAPIUrl) ; then
    exit 3
fi

echo -en "Validating ${GITLAB_CI_YML_PATH}..."
if validateGitlabCIYAML "$_GITLAB_API_ROOT_URL" "${GIT_ROOT_PATH}/${GITLAB_CI_YML_PATH}" ; then
    echo -e " ${C_F_GREEN}OK${RESC}"
else
    echo -e " ${C_F_RED}KO${RESC}"
    if [[ -n "$GITLAB_LINTER_ERROR" ]] ; then
        >&2 echo -e "${C_F_RED}$GITLAB_LINTER_ERROR${RESC}"
    else
        >&2 echo -e "${C_F_RED}Unknown error1${RESC}"
    fi
    exit 10
fi

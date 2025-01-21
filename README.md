# .gitlab-ci.yml lint helper tool

> Goodbye "yaml invalid" pipeline errors, and don't come back!

This tool use the [Gitlab API](https://docs.gitlab.com/ce/api/lint.html) to validate your local `.gitlab-ci.yml`.  
It can be installed as a git pre-commit hook, preventing commit (and so push) of an invalid `.gitlab-ci.yml`. 

**The tool itself does not lint anything: it uses the lint API of a Gitlab instance => it needs to be run somewhere with an access to the Gitlab instance where your project come from.**

# Installation

**Download the tool from the [releases page](https://gitlab.com/orobardet/gitlab-ci-linter/-/releases).**

The tool is made in [Go](https://golang.org/). So it's cross platform and can be run in Linux, Windows, Mac or any other 
operating system supported by Go.

It is currently tested on Linux x64 (Ubuntu, WSL) and Windows x64 (7 and 10). 

To install, just [download the binary](https://gitlab.com/orobardet/gitlab-ci-linter/-/releases) matching you system and put it somewhere (preferably in your `$PATH`).  
Upgrade is just overriding the binary with a new one.

> For now, releases only build binaries for some common platforms, not all supported by Go.  
> If yours is not available, you can try building it by yourself and check if it works (it should, but never tested).  
> Feedbacks are welcome :)

## Requirements

None.
  
You don't even need a Git client.

The only thing required is a network connection to the Gitlab instance you are using in the repository you want to check.    
And a git repository to check of course, having an `origin` remote corresponding to a Gitlab instance and a `.gitlab-ci.yml` file. 

## Migrating from old bash script version

If you don't want to/can't update your existing repositories with a pre-commit hook to the old bash script, the best 
way is to replace the script with a symlink to the new binary. It's a drop-in replacement.

But it would be better to remove (manually) the previous pre-commit hook link, and then install the new go version:

```shell
# check if the current pre-commit hook is a link to the old bash script
ls -lsa .git/hooks/pre-commit
# if so, remove it
rm .git/hooks/pre-commit
# and then install the new version normally
gitlab-ci-linter install
```  

# Quick start

Once installed.

## Setup

To do once per computer/environment you install the tool.

1. Generate a new [personal access token on your Gitlab.com account](https://gitlab.com/-/user_settings/personal_access_tokens), with `api` scope.  
   Then edit your `~/.netrc` (if you are creating it, **don't forget to make it only readable and writable by _you_ and no one else**), and add:
   ```shell
   machine gitlab.com account <YOUR_PERSONAL_ACCESS_TOKEN>
   ```
   You only need to this the first time you 
2. Add `export GCL_NETRC=1` to your environment (`.bashrc`, `.zshrc`, powershell profile...) and restart your shell.
3. Optionally, repeat step 1 for any private Gitlab instance you may use.

## Use
 
1. `cd` to a git repository having gitlab.com as origin remote (with https or ssh).
2. Run `gitlab-ci-lint` to check the validity of your `.gitlab-ci-lint`
3. Optionally, run `gitlab-ci-lint install` to install it as a pre-commit hook: git will launch the check each time you commit things.

## Tips

Declare an alias `gcl` in your shell to invoke the tool even quicker.

# Usage

Once installed, it can be used as a simple standalone program, by launching it from any directory 
inside a git repository clone.

Let's say you have a Gitlab project cloned in `~/dev/my-super-project`:

```shell
cd ~/dev/my-super-project
gitlab-ci-linter
```
If the `.gitlab-ci.yml` is valid:

![.gitlab-ci.yml is valid!](doc/screen-standalone-ok.png)

You don't need to be in the root of the git repository:
```shell
cd ~/dev/my-super-project/src/public
gitlab-ci-linter
```

If the `.gitlab-ci.yml` is invalid:

![Arg! An error!](doc/screen-standalone-ko.png)

## As git pre-commit hook

The tool can be used as a git pre-commit hook. It means it will be run by git automatically each time you ask for a 
commit, and git will stop if your `.gitlab-ci.yml` is invalid:

![Thanks alerting me!](doc/screen-hook-ko.png)

The tool can install (and uninstall) itself as a pre-commit hook, using the commands `install` and `uninstall`.

```shell
# Inside a git repository tree, install the pre-commit hook:
gitlab-ci-linter install
# Uninstall the pre-commit hook:
gitlab-ci-linter uninstall
```

The self installation is pretty simple: it will just create a `.git/hooks/pre-commit` file as a symbolic link to itself.

> It means updating the tool to a newer version will update all the hooks installed in all your repo => Good!  
> But moving the executable will broke commit in these repo until you manually remove the hook and reinstall it => Not so good...  
> Conclusion: install the tool in a safe and viable place :)

> **Note for Windows users:** Windows require administrator privileges to create symbolic links. So the `gitlab-ci-linter install` 
> command will only work if run with administrator privileges. 

It won't be able to self install if a `.git/hooks/pre-commit` already exists (and is not a link to itself).
 
Self uninstall will only works if `.git/hooks/pre-commit` is a link to itself.

If you are already using a pre-commit hook, you'll have to install manually: simply add a call to the tool in your 
existing pre-commit script.

### Integration with the `pre-commit` project

There is also native support for using gitlab-ci-linter as a pre-commit-hook in
the [pre-commit project](https://pre-commit.com/). If you're using pre-commit,
include this tool in your `.pre-commit-config.yaml` like this:

```
  - repo: https://gitlab.com/orobardet/gitlab-ci-linter/
    rev: < you define a git revision or tag here >
    hooks:
      - id: gitlab-ci-linter
```

Note: this supposes you have a working Go toolchain in a valid version.

## Things to know

- If no `.gitlab-ci.yml` is detected in the git repository root, the tool does nothing (if installed as pre-commit hook, it will not prevent the commit).
- This tool works (or should) with any instance of Gitlab: gitlab.com or private instance.
- It uses the url of the remote `origin` to guess the url of the Gitlab to use, and the project path (also works if the remote is ssh, as soon as the Gitlab respond on HTTP using the same FQDN as ssh)
- If the `projects/:project_path_or_id/ci/lint` API is not publicly accessible (or 2FA is enforced), you can specify a personal access token using `--personal-access-token|-p` option or `GCL_PERSONAL_ACCESS_TOKEN` environment variable. The token must have the `api` scope.
- You can also use the flag `--netrc|-n` to try getting the token from the [`.netrc` file](https://www.gnu.org/software/inetutils/manual/html_node/The-_002enetrc-file.html) (by default `~/.netrc` on *nix, `$HOME/_netrc` on Windows), but not the token must be set
   on the `account` field, not `password` (to prevent conflict with basic auth). `login` is not used.
   e.g.: for gitlab.com, the .netrc entry should be:
   ```shell
   machine gitlab.com
        # possible login and password definition
        account MY_PERSONAL_ACCESS_TOKEN
   ```
  Also, the `default` entry of .netrc is _ignored_.
- Original `/ci/lint` API endpoint was [deprecated](https://docs.gitlab.com/ee/update/deprecations.html?removal_milestone=16.0#post-cilint-api-endpoint-deprecated) in v15.7 and removed in v16.0. Now, `projects/:project_path_or_id/ci/lint` is used instead. 
  The tool will try by default to guess the project path from your remote, but you can specify:
  - The project PATH using `--project-path|-P` option or `CI_PROJECT_ID` environment variable (predefined in Gitlab CI).
  - The project ID using `--project-id|-I` option or `CI_PROJECT_PATH` environment variable (predefined in Gitlab CI).
  `--project-id` has precedence over `--project-path`.

## --help 

A bunch of options are available to configure the tool. All options can be also set using environment variables.  
Option's value on the command line have precedence over environment variables. 

```
Usage:
   gitlab-ci-linter [global options] [command [command options]] [PATH]

   The used Gitlab API is tied to a Gitlab project. Thus, the tools needs to know which Gitlab project (on which Gitlab instance) it has to target.
   By default, it will try to autodetect from the 'origin' remote configured in the git repository (if any), by extracting the FQDN as the root URL,
   and the project path. Works for 'http'' or 'ssh' remotes. e.g.: a remote "https://gitlab.com/orobardet/gitlab-ci-linter.git" or
   "git@gitlab.com:orobardet/gitlab-ci-linter.git" will target the API of the project "orobardet/gitlab-ci-linter" on "https://gitlab.com".

   In case the auto-detection does not work, or you don't have a compatible remote, or you want to target another project, you can specify the Gitlab
   root URL using '-gitlab-url|-u' flag, and the project using '--project-path|-P' or '--project-id|-I' flags. '--project-id' has precedence over '--project-path'.

   If your gitlab instance or project needs an authentification (which is the case on gitlab.com), you have to specify a personal access token with '--personal-access-token|-p'.
   You can also use the flag '--netrc|-n' to try getting the token from the .netrc file (by default ~/.netrc on *nix, $HOME/_netrc on Windows), but not the token must be set
   on the 'account' field, not 'password' (to prevent conflict with basic auth). Also, the 'default' entry of .netrc is ignored.
   e.g.: for gitlab.com, the .netrc entry should be:
      machine gitlab.com
        # possible login and password definition
        account MY_PERSONAL_ACCESS_TOKEN

Global options:
   --gitlab-url URL, -u URL             root URL of the Gitlab instance to use API (default: auto-detect from remote origin, else "https://gitlab.com") [$GCL_GITLAB_URL]
   --ci-file FILE, -f FILE              FILE is the relative or absolute path to the gitlab-ci file [$GCL_GITLAB_CI_FILE]
   --directory DIR, -d DIR              DIR is the directory from where to search for gitlab-ci file and git repository (default: ".") [$GCL_DIRECTORY]
   --personal-access-token TOK, -p TOK  personal access token TOK for accessing repositories when you have 2FA enabled. Has precedence over .netrc usage [$GCL_PERSONAL_ACCESS_TOKEN]
   --netrc, -n                          Try to get personal access token as 'account' from .netrc file (default: false) [$GCL_NETRC]
   --netrc-file value                   Path of .netrc file to use. By default, try to detect it. [$GCL_NETRC_FILE]
   --project-path PATH, -P PATH         PATH of the GitLab project that is used in the API for Gitlab >=13.6. Has precedence over path guessing from remote [$CI_PROJECT_PATH, $GCL_PROJECT_PATH]
   --project-id ID, -I ID               ID of the GitLab project that is used in the API for Gitlab >=13.6. Has precedence over --project-path [$CI_PROJECT_ID, $GCL_PROJECT_ID]
   --timeout value, -t value            timeout in second after which http request to Gitlab API will timeout (and the program will fails) (default: 15) [$GCL_TIMEOUT]
   --no-color                           don't color output. By defaults the output is colorized if a compatible terminal is detected. (default: false) [$GCL_NOCOLOR]
   --verbose, -v                        verbose mode (default: false) [$GCL_VERBOSE]
   --merged-yaml, -m                    include merged yaml in response (default: false) [$GCL_INCLUDE_MERGED_YAML]
   --help, -h                           show help
   --version                            print the version information (default: false)

Arguments:
   If PATH if given, it will depending of its type on filesystem:
    - if a file, it will be used as the gitlab-ci file to check (similar to global --ci-file option)
    - if a directory, it will be used as the folder from where to search for a ci file and a git repository (similar to global --directory option)
   PATH have precedence over --ci-file and --directory options.

Commands:
   check, c      Check the .gitlab-ci.yml (default command if none is given)
   install, i    install as git pre-commit hook
   uninstall, u  uninstall the git pre-commit hook
   version, v    Print the version information
   help, h       Shows a list of commands or help for one command

   If no command is given, 'check 'is used by default
```

## Usage examples

Check the `.gitlab-ci.yml` of the git repository containing the current working directory:

```shell
gitlab-ci-lint
# or
gitlab-ci-lint check
```

Check the `.gitlab-ci.yml` of another git repository directory:

```shell
gitlab-ci-lint /path/to/another/git
# or 
gitlab-ci-lint check /path/to/another/git
# or
gitlab-ci-lint --directory /path/to/another/git check
```

Check a specific CI file (not at the root of the git repository, or not name `.gitlab-ci-.yml`):

```shell
gitlab-ci-lint /path/to/ci-file.yml
# or 
gitlab-ci-lint check /path/to/ci-file.yml
# or
gitlab-ci-lint --ci-file /path/to/ci-file.yml check
```

Install a pre-commit hook in the current git repository:

```shell
gitlab-ci-lint install
```

Install a pre-commit hook in another git repository:

```shell
gitlab-ci-lint -d /path/to/another/git install
```

You don't want https://gitlab.com to be the default Gitlab URL to use? There is no origin remote configured in your repository?
Or you don't want to use this gitlab?

```shell
gitlab-ci-lint --gitlab-url https://gitlab.my.org check
```

But you may prefer to use define the environment variable `GCL_GITLAB_URL=https://gitlab.my.org`, possibly in your shell 
init script, to configure this globally and also for pre-commit hooks.


# Contributing

This tool was my very first Go development, while learning the language.

It was done to fit my personal needs in context of my own day work.

So there is should be plenty room for improvement. Do not hesitate to propose bugfix, new features, or code improvements.  
Open issues.

As I may not have a lot of time to implement propositions, the best way to have a request landing quickly is to come with a merge request :) 


# Development

## Clone the package

```shell
git clokd git@gitlab.com:orobardet/gitlab-ci-linter
```

## Dependencies and module

This software uses go module to handle dependencies. Just ensure to use a recent and compatible version of Go (see go.mod for exact version needed).

## Compilation

A [Makefile](Makefile) is provided, to build the executable you can simply run:

```shell
make
```

The Makefile accept the following targets (but not limited to):

- `setup`: install some Go tooling (goreleaser, golangci-lint, ...)
- `build`
- `clean` 
- `test`: runs tests with code coverage
- `html-cover`: generate an html report of tests coverage
- `check`: runs some checks (fmt, vet, lint, security, cyclo, ...)
- `rebuild`: force the rebuild from scratch (simply runs `clean` followed by `build`)
- `release-snapshot`: launch goreleaser for a local release: builds all binaries and packages in `.build/dist/`

The Makefile also accept the following environment variables:

- `BINARY`: the name and path of the binary to build (by default `.build/gitlab-ci-linter`)
- `VERSION`: the version number to include in the program (by default use the last git tag if any, else the short commit hash, both suffixed by `-dev`)
- `REVISION`: the revision string to include in the program, typically the VCS commit hash (by default the git full commit hash) 
- `BUILDTIME`: the build date and time (by default the current ones, of course)
- `DEBUG`: binaries a build without debug symbols to reduce their size (`-s -w` link options) ; setting `DEBUG` to a non-zero value (0 by default) will build binary with debug symbols

Other targets exists, look directly in the [Makefile](Makefile)'s comments. 

## Dev workflow

- Change the code
- Run with:
```shell
# The following commands are equivalent:
go run . [--some-gitlab-ci-linter-options or arguments]
``` 

Once you are happy with your changes, and before commiting/pushing something, check the code quality and run tests with `make checks`.
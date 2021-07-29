# vNext

- Added a `--merged-yaml` option to allow merged yaml to be returned in response from gitlab API

# v2.2.0

- Add a `--personal-access-token` option specify a personal access token (e.g. when 2FA is enabled). Thanks @fhitche1 (!13)
- Fixes short command line options that where not working since upgrade to urfave/cli v2 (#12)
- Option `--gitlab-url` now has precedence over detecting URL from the origin remote(#13)
- Code refactoring, CI and build tooling improvement (!15)
- Validate value of `--gitlab-url` (#14)
- Better error when unable to contact the gitlab API (#9)

**Breaking changes:**

- When no scheme (http or https) is explicitly given or guessable for a Gitlab URL, https:// is now used by default.

# v2.1.0

- Upgrade dependencies to latest versions (#10)
- Upgrade to go 1.13 and go modules. Thanks @sascha-andres (!9)
- Introduce support for use with pre-commit.com
- Add support for git repos without a remote named "origin". Thanks @rubensayshi (!11)
- Compress release binaries with UPX to reduce size (#4)
- Binary releases are uploaded to bintray (#2)

# v2.0.0

- Full rewrite in Go

# v1.0.0

- First version as a pure bash script

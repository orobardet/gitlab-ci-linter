FROM alpine

COPY gitlab-ci-linter /usr/bin/gitlab-ci-linter

CMD [ "gitlab-ci-linter" ]
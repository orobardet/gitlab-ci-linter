FROM golang:1.23

# Install jq
ADD https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 /usr/local/bin/jq
RUN chmod +x /usr/local/bin/jq

# Set a local go user and use it
ENV GOUSER=go
RUN adduser --gecos "" --disabled-password $GOUSER
USER $GOUSER

# Install some go tools
RUN go install golang.org/x/tools/cmd/godoc@latest && \
    go install golang.org/x/lint/golint@latest && \
    go install github.com/alecthomas/gocyclo@latest && \
    go install github.com/securego/gosec/v2/cmd/gosec@latest && \
    go clean && rm -fr /tmp/*

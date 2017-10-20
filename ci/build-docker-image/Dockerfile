FROM golang:1.9

ENV GOUSER=go

RUN adduser --gecos "" --disabled-password $GOUSER

USER $GOUSER

RUN go get -u github.com/golang/dep/cmd/dep
RUN ln -s /go/bin/dep /go/bin/godep
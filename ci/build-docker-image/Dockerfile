FROM golang:1.9 AS upxbuilder

RUN apt-get update -y && \
    apt-get install -y --no-install-recommends xz-utils && \
    mkdir /upxtmp && \
    cd /upxtmp && \
    wget -O upx.tar.xz https://github.com/upx/upx/releases/download/v3.94/upx-3.94-amd64_linux.tar.xz && \
    tar --strip-components=1 -Jxvf upx.tar.xz

FROM golang:1.9

ADD  https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 /usr/local/bin/jq
RUN chmod +x /usr/local/bin/jq

COPY --from=upxbuilder /upxtmp/upx /usr/local/bin/upx
RUN chmod +x /usr/local/bin/upx

ENV GOUSER=go

RUN adduser --gecos "" --disabled-password $GOUSER

USER $GOUSER

RUN go get -u github.com/golang/dep/cmd/dep
RUN ln -s /go/bin/dep /go/bin/godep


FROM golang:1.13 AS upxbuilder

# UPX compilation
RUN apt-get update -y && \
    apt-get install -y --no-install-recommends xz-utils && \
    mkdir /upxtmp && \
    cd /upxtmp && \
    wget -O upx.tar.xz https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz && \
    tar --strip-components=1 -Jxvf upx.tar.xz

FROM golang:1.13

# Install jq
ADD https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 /usr/local/bin/jq
RUN chmod +x /usr/local/bin/jq

# Install UPX from previous stage
COPY --from=upxbuilder /upxtmp/upx /usr/local/bin/upx
RUN chmod +x /usr/local/bin/upx

# Install Snyk
RUN curl -sL https://deb.nodesource.com/setup_10.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    npm install --global snyk && \
    npm cache clean --force && \
    apt-get clean && rm -fr /tmp/* /var/lib/apt/*

# Set a local go user and use it
ENV GOUSER=go
RUN adduser --gecos "" --disabled-password $GOUSER
USER $GOUSER

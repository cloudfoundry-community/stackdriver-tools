FROM ubuntu:16.04
MAINTAINER Jeff Johnson <jrjohnson@google.com>

# Update base image
RUN apt-get update && apt-get -y upgrade

# Setup language/locale
RUN apt-get install -y locales && apt-get clean
ENV LANGUAGE en_US.UTF-8
ENV LANG en_US.UTF-8
ENV LC_ALL en_US.UTF-8
ENV LC_CTYPE en_US.UTF-8
RUN locale-gen en_US.UTF-8; dpkg-reconfigure locales

# Build dependencies
WORKDIR /tmp/docker-build
RUN apt-get -y install build-essential git-core curl wget jq sudo make gcc; apt-get clean

# Golang + Test dependencies
ENV GO_VERSION=1.9.1
ENV GO_SHA256SUM=07d81c6b6b4c2dcf1b5ef7c27aaebd3691cdb40548500941f92b221147c5d9c7

RUN curl -LO https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz && \
    echo "${GO_SHA256SUM}  go${GO_VERSION}.linux-amd64.tar.gz" > go_${GO_VERSION}_SHA256SUM && \
    sha256sum -cw --status go_${GO_VERSION}_SHA256SUM
RUN tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
ENV GOPATH /root/go
RUN mkdir -p /root/go/bin
ENV PATH $PATH:/usr/local/go/bin:$GOPATH/bin
RUN go get github.com/onsi/ginkgo
RUN go install github.com/onsi/ginkgo/...
RUN go get github.com/golang/lint/golint


# Google SDK
ENV GCLOUD_VERSION=157.0.0
ENV GCLOUD_SHA1SUM=383522491db5feb9f03053f29aaf6a1cf778e070

RUN apt-get -y install python-software-properties; apt-get clean
RUN wget https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${GCLOUD_VERSION}-linux-x86_64.tar.gz \
    -O gcloud_${GCLOUD_VERSION}_linux_amd64.tar.gz && \
    echo "${GCLOUD_SHA1SUM}  gcloud_${GCLOUD_VERSION}_linux_amd64.tar.gz" > gcloud_${GCLOUD_VERSION}_SHA1SUM && \
    sha1sum -cw --status gcloud_${GCLOUD_VERSION}_SHA1SUM && \
    tar xvf gcloud_${GCLOUD_VERSION}_linux_amd64.tar.gz && \
    mv google-cloud-sdk / && cd /google-cloud-sdk  && ./install.sh

ENV PATH=$PATH:/google-cloud-sdk/bin

# Ruby
RUN apt-get -y install ruby2.3 ruby2.3-dev; apt-get clean

# Install Bundler and BOSH CLI
RUN /bin/bash -l -c "gem install bundler bosh_cli --no-ri --no-rdoc"

# Install BOSH2 CLI
ENV BOSH2_VERSION=2.0.42
ENV BOSH2_SHA1SUM=a97b085e45a989abfcd88de77770bb6db0c8a29e

RUN curl -LO https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-${BOSH2_VERSION}-linux-amd64 && \
    echo "${BOSH2_SHA1SUM}  bosh-cli-${BOSH2_VERSION}-linux-amd64" > bosh-cli-${BOSH2_VERSION}-SHA1SUM && \
    sha1sum -cw --status bosh-cli-${BOSH2_VERSION}-SHA1SUM && \
    mv bosh-cli-${BOSH2_VERSION}-linux-amd64 /usr/local/bin/bosh2 && \
    chmod a+x /usr/local/bin/bosh2

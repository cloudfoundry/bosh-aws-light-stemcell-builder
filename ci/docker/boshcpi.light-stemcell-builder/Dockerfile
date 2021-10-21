FROM ubuntu:16.04

ENV BOSH_CLI_VERSION 6.2.1-linux-amd64
ENV METALINK_VERSION 0.3.0-linux-amd64
ENV GO_VERSION go1.17.2.linux-amd64

RUN apt update; apt -y upgrade; apt-get clean
RUN apt install -y \
  git tar curl wget make ruby unzip qemu-utils python python-pip jq && \
  apt-get clean
RUN pip install awscli

RUN cd /tmp && \
    wget -nv "https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-${BOSH_CLI_VERSION}" && \
    ( \
        echo "ca7580008abfd4942dcb1dd6218bde04d35f727717a7d08a2bc9f7d346bce0f6 bosh-cli-${BOSH_CLI_VERSION}" | \
        sha256sum -c - \
    ) && \
    mv /tmp/bosh-cli-* /usr/local/bin/bosh && \
    chmod +x /usr/local/bin/bosh

RUN cd /tmp && \
    wget -nv "https://github.com/dpb587/metalink/releases/download/v0.3.0/meta4-${METALINK_VERSION}" && \
    ( \
        echo "e276ccff74cbe9a85eda17e5955df0b0b06ee68fcdb67d860f691d41b34f65cf meta4-${METALINK_VERSION}" | \
        sha256sum -c - \
    ) && \
    mv /tmp/meta4-* /usr/local/bin/meta4 && \
    chmod +x /usr/local/bin/meta4

RUN cd /tmp && \
    wget -nv "https://dl.google.com/go/${GO_VERSION}.tar.gz" && \
    ( \
        echo "8a4cbc9f2b95d114c38f6cbe94a45372d48c604b707db2057c787398dfbf8e7f ${GO_VERSION}.tar.gz" | \
        sha256sum -c - \
    ) && \
    tar -C /usr/local -xzf go*.tar.gz && \
    rm go*.tar.gz

ENV GOPATH /opt/go

ENV PATH /usr/local/go/bin:/opt/go/bin:$PATH

RUN mkdir /opt/go && go get github.com/onsi/ginkgo/ginkgo

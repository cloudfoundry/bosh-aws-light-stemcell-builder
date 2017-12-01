FROM ubuntu:16.04

ENV BOSH_CLI_VERSION 2.0.45-linux-amd64
ENV METALINK_VERSION 0.2.0-linux-amd64

RUN apt update; apt -y upgrade; apt-get clean
RUN apt install -y \
  git tar curl wget make ruby unzip qemu-utils python python-pip jq && \
  apt-get clean
RUN pip install awscli

RUN cd /tmp && \
    wget -nv "https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-${BOSH_CLI_VERSION}" && \
    ( \
        echo "bf04be72daa7da0c9bbeda16fda7fc7b2b8af51e bosh-cli-${BOSH_CLI_VERSION}" | \
        sha1sum -c - \
    ) && \
    mv /tmp/bosh-cli-* /usr/local/bin/bosh && \
    chmod +x /usr/local/bin/bosh

RUN cd /tmp && \
    wget -nv "https://github.com/dpb587/metalink/releases/download/v0.2.0/meta4-${METALINK_VERSION}" && \
    ( \
        echo "81a592eaf647358563f296aced845ac60d9061a45b30b852d1c3f3674720fe19 meta4-${METALINK_VERSION}" | \
        sha256sum -c - \
    ) && \
    mv /tmp/meta4-* /usr/local/bin/meta4 && \
    chmod +x /usr/local/bin/meta4

RUN cd /tmp && \
    wget -nv https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz && \
    ( \
        echo '1862f4c3d3907e59b04a757cfda0ea7aa9ef39274af99a784f5be843c80c6772 go1.8.3.linux-amd64.tar.gz' | \
        sha256sum -c - \
    ) && \
    tar -C /usr/local -xzf go*.tar.gz && \
    rm go*.tar.gz

ENV GOPATH /opt/go

ENV PATH /usr/local/go/bin:/opt/go/bin:$PATH

RUN mkdir /opt/go && go get github.com/onsi/ginkgo/ginkgo

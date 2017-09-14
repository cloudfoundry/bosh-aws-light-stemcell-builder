FROM ubuntu:15.10

RUN locale-gen en_US.UTF-8
RUN dpkg-reconfigure locales
ENV LANG en_US.UTF-8
ENV LC_ALL en_US.UTF-8

RUN apt-get update; apt-get -y upgrade; apt-get clean
RUN apt-get install -y \
  git tar curl wget make ruby unzip qemu-utils python python-pip jq && \
  apt-get clean
RUN pip install awscli

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

FROM ubuntu:15.10

RUN locale-gen en_US.UTF-8
RUN dpkg-reconfigure locales
ENV LANG en_US.UTF-8
ENV LC_ALL en_US.UTF-8

RUN apt-get update; apt-get -y upgrade; apt-get clean
RUN apt-get install -y git tar curl wget make unzip; apt-get clean

RUN cd /tmp && \
    curl -O -L https://storage.googleapis.com/golang/go1.6.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go*.tar.gz && \
    rm go*.tar.gz

ENV PATH /usr/local/go/bin:$PATH

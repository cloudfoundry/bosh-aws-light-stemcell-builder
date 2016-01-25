FROM ubuntu:15.10

RUN locale-gen en_US.UTF-8
RUN dpkg-reconfigure locales
ENV LANG en_US.UTF-8
ENV LC_ALL en_US.UTF-8

RUN apt-get update; apt-get -y upgrade; apt-get clean
RUN apt-get install -y git tar curl wget make python unzip python-pip openjdk-8-jre jq uuid-runtime; apt-get clean
RUN apt-get install -y libxslt-dev libxml2-dev zlib1g-dev; apt-get clean;
RUN apt-get install -y vagrant; apt-get clean; vagrant plugin install vagrant-aws
RUN pip install awscli pyyaml

RUN /var/lib/dpkg/info/ca-certificates-java.postinst configure

RUN mkdir /usr/local/ec2;
RUN cd /tmp; curl -O http://s3.amazonaws.com/ec2-downloads/ec2-api-tools.zip; unzip ec2-api-tools.zip -d /usr/local/ec2; cd -

# Utility python scripts for converting YAML <-> JSON
COPY json2yaml yaml2json /usr/bin/

RUN cd /tmp && \
    curl -O -L https://storage.googleapis.com/golang/go1.5.1.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go*.tar.gz && \
    rm go*.tar.gz

ENV JAVA_HOME /usr/lib/jvm/java-8-openjdk-amd64/jre
ENV EC2_HOME /usr/local/ec2/ec2-api-tools-1.7.5.1
ENV PATH /usr/local/go/bin:$EC2_HOME/bin:$PATH

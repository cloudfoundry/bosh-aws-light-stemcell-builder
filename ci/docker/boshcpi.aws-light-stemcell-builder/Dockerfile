FROM ubuntu:15.04

RUN locale-gen en_US.UTF-8
RUN dpkg-reconfigure locales
ENV LANG en_US.UTF-8
ENV LC_ALL en_US.UTF-8

RUN apt-get update; apt-get -y upgrade; apt-get clean
RUN apt-get install -y git tar curl wget make python unzip python-pip openjdk-7-jre jq; apt-get clean
RUN apt-get install -y libxslt-dev libxml2-dev zlib1g-dev; apt-get clean;
RUN apt-get install -y vagrant; apt-get clean; vagrant plugin install vagrant-aws
RUN pip install awscli

RUN mkdir /usr/local/ec2;
RUN cd /tmp; curl -O http://s3.amazonaws.com/ec2-downloads/ec2-api-tools.zip; unzip ec2-api-tools.zip -d /usr/local/ec2; cd -

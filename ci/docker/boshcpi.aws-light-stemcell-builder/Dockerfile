FROM ubuntu:latest

RUN locale-gen en_US.UTF-8
RUN dpkg-reconfigure locales
ENV LANG en_US.UTF-8
ENV LC_ALL en_US.UTF-8

RUN apt-get update; apt-get -y upgrade; apt-get clean
RUN apt-get install -y git curl python unzip python-pip openjdk-7-jre; apt-get clean
RUN pip install awscli

RUN mkdir /usr/local/ec2;
RUN cd /tmp; curl -O http://s3.amazonaws.com/ec2-downloads/ec2-api-tools.zip; unzip ec2-api-tools.zip -d /usr/local/ec2; cd -

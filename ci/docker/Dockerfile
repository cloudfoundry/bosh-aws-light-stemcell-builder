FROM bosh/golang-release

ARG BOSH_CLI_VERSION
ARG METALINK_VERSION

RUN apt update; apt -y upgrade; apt-get clean
RUN apt install -y \
  git tar curl wget make ruby unzip qemu-utils python3 python3-pip jq && \
  apt-get clean
RUN pip3 install awscli

RUN curl -fsL https://github.com/cloudfoundry/bosh-cli/releases/download/v${BOSH_CLI_VERSION}/bosh-cli-${BOSH_CLI_VERSION}-linux-amd64 \
    > /usr/local/bin/bosh \
    && chmod +x /usr/local/bin/bosh

RUN curl -fsL https://github.com/dpb587/metalink/releases/download/v${METALINK_VERSION}/meta4-${METALINK_VERSION}-linux-amd64 \
    > /usr/local/bin/meta4 \
    && chmod +x /usr/local/bin/meta4

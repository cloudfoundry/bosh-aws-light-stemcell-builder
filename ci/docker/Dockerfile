FROM ghcr.io/cloudfoundry/bosh/golang-release

ARG BOSH_CLI_VERSION
ARG METALINK_VERSION

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
      git \
      tar \
      curl \
      wget \
      make \
      ruby \
      unzip \
      qemu-utils \
      python3 \
      python3-pip \
      jq
RUN pip3 install awscli

RUN curl -fsL https://github.com/cloudfoundry/bosh-cli/releases/download/v${BOSH_CLI_VERSION}/bosh-cli-${BOSH_CLI_VERSION}-linux-amd64 \
      > /usr/local/bin/bosh \
    && chmod +x /usr/local/bin/bosh

RUN curl -fsL https://github.com/dpb587/metalink/releases/download/v${METALINK_VERSION}/meta4-${METALINK_VERSION}-linux-amd64 \
      > /usr/local/bin/meta4 \
    && chmod +x /usr/local/bin/meta4

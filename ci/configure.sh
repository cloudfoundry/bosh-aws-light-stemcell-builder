#!/usr/bin/env bash

project_dir="$( cd $(dirname $0)/.. && pwd )"

fly -t bosh-ecosystem sp -p bosh-aws-light-stemcell-builder -c ${project_dir}/ci/pipeline.yml

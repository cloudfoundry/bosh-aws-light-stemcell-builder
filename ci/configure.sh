#!/usr/bin/env bash
set -eu -o pipefail

if [[ -n "${DEBUG:-}" ]]; then
  set -x
fi

REPO_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"

concourse_target="${CONCOURSE_TARGET:-stemcell}"
fly="${FLY_CLI:-fly}"

pipeline_name="bosh-aws-light-stemcell-builder"
pipeline_config="${REPO_ROOT}/ci/pipeline.yml"

echo "Validating..."
"${fly}" validate-pipeline --strict --config "${pipeline_config}"
echo ""

"${fly}" -t "${concourse_target}" \
  set-pipeline \
    --pipeline "${pipeline_name}" \
    --config "${pipeline_config}"

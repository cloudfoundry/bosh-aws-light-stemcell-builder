#!/bin/bash
set -eu -o pipefail

cat <<EOF > settings.json
{
  "params": {
    "rename": "{{.Version}}/stemcells.aws.meta4",
    "files": ["light-stemcell/*.tgz"],
    "version": "us-input-stemcell/.resource/version"
  },
  "source": {
    "uri": "$uri",
    "version": "$version"
    "mirror_files": [
      {
        "destination": "s3://s3.amazonaws.com/${bucket_name}/{{.Name}}",
      }
    ],
    "url_handlers": [
      {
        "type": "s3",
        include:"(s3|https)://.*",
        "options": {
          "access_key": "$access_key",
          secret_key: "$secret_key"
        }
      }
    ],
    "options": {
      "private_key": "$(echo "$git_private_key" | tr '\n' '#'|  sed 's/#/\\n/g')"
    }
  }
}
EOF

cat settings.json | /opt/resource/out $PWD

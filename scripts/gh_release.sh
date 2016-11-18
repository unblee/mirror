#!/bin/sh
set -eu

# This script is used only in containers.

apk add --no-cache openssl curl git

echo ""
wget -O /tmp/ghr.zip https://github.com/tcnksm/ghr/releases/download/v0.5.3/ghr_v0.5.3_linux_amd64.zip
unzip -d /go/bin /tmp/ghr.zip
ghr=/go/bin/ghr
ghr_cmd="${ghr} -u ${USERNAME} -r ${REPONAME} ${VERSION} ${RELEASE_ARCHIVE_DIR}"
echo ${ghr_cmd}
eval ${ghr_cmd}

echo ""
echo Release to GitHub is completed!
echo ""
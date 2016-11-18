#!/bin/sh
set -eu

# This script is used only in containers.

rm -fr ${RELEASE_BIN_DIR}
out_dir=${RELEASE_BIN_DIR}/${BINNAME}-${VERSION}-linux-amd64
mkdir ${RELEASE_BIN_DIR} && chmod 777 ${RELEASE_BIN_DIR}
mkdir ${out_dir} && chmod 777 ${out_dir}

build_cmd="GOOS=linux GOARCH=amd64 go build -a -tags netgo -installsuffix netgo -ldflags \"${LDFLAGS}\" -o ${out_dir}/${BINNAME}"
echo ${build_cmd}
eval ${build_cmd}

echo ""
echo build finished!
echo ""
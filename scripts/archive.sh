#!/bin/sh
set -eu

# This script is used only in containers.

apk add --no-cache zip

rm -fr ${RELEASE_ARCHIVE_DIR}
mkdir ${RELEASE_ARCHIVE_DIR} && chmod 777 ${RELEASE_ARCHIVE_DIR}

echo ""
for target in `ls ${RELEASE_BIN_DIR}`
do
  {
    cd ${RELEASE_BIN_DIR}
    archive_cmd="zip -r ${RELEASE_ARCHIVE_DIR}/${target}.zip ${target}"
    echo ${archive_cmd}
    eval ${archive_cmd}
    archive_cmd="tar -zcvf ${RELEASE_ARCHIVE_DIR}/${target}.tar.gz ${target}"
    echo ${archive_cmd}
    eval ${archive_cmd}
  }
done

echo ""
echo archive finished!
echo ""
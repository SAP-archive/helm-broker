#!/usr/bin/env bash

if [ "$#" -ne 1 ]; then
    echo "Some parameters [VERSION] were not provided"
    return 1
fi

VERSION=$1

echo "Changing the version of links to docs to: ${VERSION}"

sed -i "" "s#helm-broker/blob/master/docs#helm-broker/blob/${VERSION}/docs#g" README.md
sed -i "" "s/__RELEASE_VERSION__/${VERSION}/g" docs/release/release-base.md

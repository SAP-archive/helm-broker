#!/usr/bin/env bash

if [ "$#" -ne 3 ]; then
    echo "Some parameters [GIT_TAG, REPO_NAME, REPO_OWNER] were not provided"
    return 1
fi

GIT_TAG=$1
REPO_NAME=$2
REPO_OWNER=$3

CHANGELOG_FLAGS=
LAST_TAG=$(git describe --tags $(git rev-list --tags --max-count=1 --skip=1 --no-walk))

if [[ "${GIT_TAG}" == "${LAST_TAG}" ]]; then
    # needed when we create first release
    LAST_TAG=
fi

if [[ -n "${LAST_TAG}" ]]; then
    CHANGELOG_FLAGS="$CHANGELOG_FLAGS --since-tag $LAST_TAG"
fi

if [[ -n "${GIT_TAG}" ]]; then
    CHANGELOG_FLAGS="$CHANGELOG_FLAGS --future-release $GIT_TAG"
fi

set -e

docker run --rm -v $(pwd):/usr/local/src/your-app ferrarimarco/github-changelog-generator -u ${REPO_OWNER} -p ${REPO_NAME} -t ${GITHUB_TOKEN} ${CHANGELOG_FLAGS} > /dev/null

mv CHANGELOG.md toCopy/
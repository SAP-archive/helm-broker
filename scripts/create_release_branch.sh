#!/usr/bin/env bash
set -e

if [ "$#" -ne 2 ]; then
    echo "Some parameters [GIT_TAG, GIT_REPO] were not provided"
    exit
fi

GIT_TAG=$1
GIT_REPO=$2

# exclude 'v' from release branch name
TAG=$(echo ${GIT_TAG} | cut -d "v" -f 2)

MAJOR=$(echo ${TAG} | cut -d. -f1)
MINOR=$(echo ${TAG} | cut -d. -f2)
REVISION=$(echo ${TAG} | cut -d. -f3)

if [[ ${REVISION} = "0" ]]; then
  GIT_BRANCH=release-${MAJOR}.${MINOR}
  echo "Creating branch ${GIT_BRANCH}"
  git checkout -b ${GIT_BRANCH}
  git push https://${GITHUB_TOKEN}@github.com/${GIT_REPO} ${GIT_BRANCH}
fi
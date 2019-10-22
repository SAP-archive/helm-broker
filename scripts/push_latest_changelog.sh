#!/usr/bin/env bash
set -e

if [ "$#" -ne 2 ]; then
    echo "Some parameters [GIT_TAG, GIT_REPO] were not provided"
    exit
fi

GIT_TAG=$1
GIT_REPO=$2
GIT_BRANCH=master

git remote add origin https://github.com/kyma-project/helm-broker.git || true
git fetch origin
git checkout ${GIT_BRANCH}

git add ./CHANGELOG.md
git status
git commit -m "Update Changelog"

git push https://${GITHUB_TOKEN}@github.com/${GIT_REPO} ${GIT_BRANCH}
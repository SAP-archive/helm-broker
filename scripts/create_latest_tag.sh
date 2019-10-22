#!/usr/bin/env bash
set -e

GIT_REPO=$1
export GIT_TAG=latest
git tag ${GIT_TAG} -f -a -m "Generated tag from ProwCI"
git push https://${GITHUB_TOKEN}@github.com/${GIT_REPO} ${GIT_TAG} -f
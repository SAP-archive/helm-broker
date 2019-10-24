#!/usr/bin/env bash
set -e

if [ "$#" -ne 2 ]; then
    echo "Some parameters [GIT_TAG, GIT_REPO] were not provided"
    exit
fi

GIT_TAG=$1
GIT_REPO=$2

CHANGELOG=./CHANGELOG.md
CHART=./helm-broker-chart.tar.gz

body="$(cat CHANGELOG.md)"

# Overwrite CHANGELOG.md with JSON data for GitHub API
jq -n \
  --arg body "$body" \
  --arg name "${GIT_TAG}" \
  --arg tag_name "${GIT_TAG}" \
  --arg target_commitish "master" \
  '{
    body: $body,
    name: $name,
    tag_name: $tag_name,
    target_commitish: $target_commitish,
    draft: true,
    prerelease: false
  }' > CHANGELOG.md

echo "Create release ${GIT_TAG} for repo: ${GIT_REPO}, branch: ${GIT_TAG}"
RESPONSE=$(curl -H "Authorization: token ${GITHUB_TOKEN}" --data @CHANGELOG.md "https://api.github.com/repos/${GIT_REPO}/releases")
ASSET_UPLOAD_URL=$(echo "$RESPONSE" | jq -r .upload_url | cut -d '{' -f1)
if [ -z "$ASSET_UPLOAD_URL" ]; then
    echo ${RESPONSE}
    exit 1
fi

echo "Uploading CHANGELOG to url: $ASSET_UPLOAD_URL?name=${CHANGELOG}"
curl -s --data-binary @${CHANGELOG} -H "Content-Type: application/octet-stream" -X POST "$ASSET_UPLOAD_URL?name=$(basename ${CHANGELOG})&access_token=${GITHUB_TOKEN}" > /dev/null

echo "Uploading CHART to url: $ASSET_UPLOAD_URL?name=${CHART}"
curl -s --data-binary @${CHART} -H "Content-Type: application/octet-stream" -X POST "$ASSET_UPLOAD_URL?name=$(basename ${CHART})&access_token=${GITHUB_TOKEN}" > /dev/null


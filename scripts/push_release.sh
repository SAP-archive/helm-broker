#!/usr/bin/env bash
set -e

if [ "$#" -ne 2 ]; then
    echo "Some parameters [GIT_TAG, GIT_REPO] were not provided"
    exit
fi

GIT_TAG=$1
GIT_REPO=$2

body="$(cat toCopy/CHANGELOG.md)"

# Overwrite CHANGELOG.md with JSON data for GitHub API
jq -n \
  --arg body "$body" \
  --arg name "${GIT_TAG}" \
  --arg tag_name "${GIT_TAG}" \
  --arg target_commitish "$(git rev-parse HEAD)" \
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

for FILE in toCopy/*; do
    echo "Uploading asset: $FILE to url: $ASSET_UPLOAD_URL?name=${FILE}"
    curl -s --data-binary @${FILE} -H "Content-Type: application/octet-stream" -X POST "$ASSET_UPLOAD_URL?name=$(basename ${FILE})&access_token=${GITHUB_TOKEN}" > /dev/null
done

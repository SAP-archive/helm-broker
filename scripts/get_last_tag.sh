#!/usr/bin/env bash
set -e

function findLatestTag() {
    TAG_LIST_STRING=$(git describe --tags $(git rev-list --tags) --always | grep -F . | grep -v "-")
    TAG_LIST=($(echo $TAG_LIST_STRING | tr " " "\n"))
    TAG=$1

    for i in "${!TAG_LIST[@]}"
    do
       :
       if [[ $TAG == ${TAG_LIST[$i]} ]]
        then PENULTIMATE=${TAG_LIST[$i+1]}
       fi
    done
}

findLatestTag $1

if [ "$PENULTIMATE" != "" ]; then
    echo $PENULTIMATE
else
    echo $(git rev-list --max-parents=0 HEAD)
fi
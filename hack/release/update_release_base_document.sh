#!/usr/bin/env bash

if [ "$#" -ne 1 ]; then
    echo "Some parameters [VERSION] were not provided"
    return 1
fi

VERSION=$1

sed docs/release/release-base.md
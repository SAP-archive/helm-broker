#!/usr/bin/env bash

YAML_EDITOR=

if command -v yq.v2 >/dev/null 2>&1 ; then
    YAML_EDITOR=$(which yq.v2)
fi

if command -v yq >/dev/null 2>&1 ; then
    YAML_EDITOR=$(which yq)
fi

if [[ -z ${YAML_EDITOR} ]]; then
    go get gopkg.in/mikefarah/yq.v2
    YAML_EDITOR=$(which yq.v2)
fi

echo "${YAML_EDITOR}"
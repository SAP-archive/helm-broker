#!/usr/bin/env bash

YAML_EDITOR=
if [[ -z $(which yq.v2) ]]; then
    YAML_EDITOR=$(which yq.v2)
fi

if [[ -z $(which yq) ]]; then
    YAML_EDITOR=$(which yq)
fi

if [[ -z ${YAML_EDITOR} ]]; then
    go get gopkg.in/mikefarah/yq.v2
    YAML_EDITOR=$(which yq.v2)
fi

echo "${YAML_EDITOR}"
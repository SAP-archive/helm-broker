#!/usr/bin/env bash

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

export GO111MODULE=on

##
# GO BUILD
##
binaries=("broker" "controller" "indexbuilder" "targz" "webhook")
buildEnv=""
if [ "$1" == "$CI_FLAG" ]; then
	# build binary statically for linux architecture
	buildEnv="env CGO_ENABLED=0 GOOS=linux GOARCH=amd64"
fi

for binary in "${binaries[@]}"; do
	${buildEnv} go build -o ${binary} ./cmd/${binary}
	goBuildResult=$?
	if [ ${goBuildResult} != 0 ]; then
		echo -e "${RED}✗ go build ${binary} ${NC}\n$goBuildResult${NC}"
		exit 1
	else echo -e "${GREEN}√ go build ${binary} ${NC}"
	fi
done

echo "? compile chart tests"
${buildEnv} go test -v -c -o hb_chart_test ./test/charts/helm_broker_test.go
goBuildResult=$?
if [[ ${goBuildResult} != 0 ]]; then
    echo -e "${RED}✗ go test -c ./test/charts/helm_broker_test.go ${NC}\n$goBuildResult${NC}"
    exit 1
else echo -e "${GREEN}√ go test -c ./test/charts/helm_broker_test.go ${NC}"
fi

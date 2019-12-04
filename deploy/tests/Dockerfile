FROM alpine:3.10

RUN apk --no-cache add ca-certificates

COPY ./hb_chart_test /usr/local/bin/hb_chart_test

LABEL source=git@github.com:kyma-project/helm-broker.git

ENTRYPOINT ["hb_chart_test"]

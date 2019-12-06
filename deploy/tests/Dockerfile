FROM alpine:3.10 as certs
RUN apk --no-cache add ca-certificates

FROM scratch

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ./hb_chart_test /usr/local/bin/hb_chart_test

LABEL source=git@github.com:kyma-project/helm-broker.git

ENTRYPOINT ["hb_chart_test"]

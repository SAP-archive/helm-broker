FROM alpine:3.10 AS builder

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache curl

FROM scratch

COPY ./hb_chart_test /usr/local/bin/hb_chart_test
COPY --from=builder /usr/bin/curl /bin/curl
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

LABEL source=git@github.com:kyma-project/helm-broker.git

ENTRYPOINT ["hb_chart_test"]

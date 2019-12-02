FROM alpine:3.10 as certs
RUN apk update && apk --no-cache add ca-certificates

FROM eu.gcr.io/kyma-project/health-check-proxy:PR-6466 as health

FROM alpine:3.10
RUN apk add --no-cache git openssh mercurial

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller
COPY --from=health /health-check /health-check
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/start.sh"]

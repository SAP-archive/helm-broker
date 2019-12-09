FROM alpine:latest as certs
RUN apk --no-cache add ca-certificates

FROM scratch

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ./helm-broker /root/helm-broker

LABEL source=git@github.com:kyma-project/helm-broker.git

ENTRYPOINT ["/root/helm-broker"]

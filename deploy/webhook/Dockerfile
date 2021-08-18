FROM alpine:latest as certs
RUN apk --no-cache add ca-certificates

FROM scratch

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ./webhook /root/webhook

LABEL source=git@github.com:kyma-project/helm-broker.git

ENTRYPOINT ["/root/webhook"]
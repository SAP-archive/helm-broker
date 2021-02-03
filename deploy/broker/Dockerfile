FROM alpine:latest as certs
RUN apk --no-cache add ca-certificates

FROM scratch

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app
COPY ./helm-broker /app/helm-broker

LABEL source=git@github.com:kyma-project/helm-broker.git

ENTRYPOINT ["/app/helm-broker"]

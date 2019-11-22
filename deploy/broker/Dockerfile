FROM alpine:3.10 as certs
RUN apk --no-cache add ca-certificates

FROM health-check

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ./helm-broker /root/helm-broker

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/helm-broker"]
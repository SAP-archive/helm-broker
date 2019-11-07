FROM alpine:3.10 as builder
RUN apk add --no-cache curl git openssh mercurial

FROM alpine:3.10 as certs
RUN apk --no-cache add ca-certificates

FROM scratch

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller
COPY --from=builder /usr/bin/git /bin/git
COPY --from=builder /usr/bin/ssh /bin/ssh
COPY --from=builder /usr/bin/hg /bin/hg
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/start.sh"]

FROM alpine:3.12

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache curl=7.69.1-r1 git=2.26.2-r0 openssh=8.3_p1-r0 mercurial=5.4-r0

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/start.sh"]

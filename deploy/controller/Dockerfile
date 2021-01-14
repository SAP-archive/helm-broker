FROM alpine:3.12.3

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache git=2.26.2-r0 openssh=8.3_p1-r1 mercurial=5.4-r0
RUN apk --no-cache add --update curl --repository=http://dl-cdn.alpinelinux.org/alpine/edge/main

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/start.sh"]

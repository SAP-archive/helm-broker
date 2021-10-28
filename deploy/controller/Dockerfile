FROM alpine:3.14.2

RUN apk add --no-cache ca-certificates curl
RUN apk add --no-cache mercurial>=5.7-r0 --repository=http://dl-cdn.alpinelinux.org/alpine/edge/main
RUN apk add --no-cache git>=2.33.1-r0 --repository=http://dl-cdn.alpinelinux.org/alpine/edge/main
RUN apk add --no-cache openssh=8.8_p1-r0 --repository=http://dl-cdn.alpinelinux.org/alpine/edge/main

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/start.sh"]

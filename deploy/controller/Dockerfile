FROM alpine:3.14.3

RUN apk add --no-cache ca-certificates curl
RUN apk add --no-cache mercurial>=5.9.3-r1 --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community
RUN apk add --no-cache git>=2.34.1-r1 --repository=http://dl-cdn.alpinelinux.org/alpine/edge/main
RUN apk add --no-cache openssh>=8.8_p1-r3 --repository=http://dl-cdn.alpinelinux.org/alpine/edge/main

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/start.sh"]

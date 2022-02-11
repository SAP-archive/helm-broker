FROM alpine:3.15.0

RUN apk add --no-cache --update git openssh expat git sqlite openssl openssl-dev --repository=https://dl-cdn.alpinelinux.org/alpine/edge/main
RUN apk add --no-cache mercurial --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community
RUN apk add --no-cache ca-certificates curl

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/start.sh"]

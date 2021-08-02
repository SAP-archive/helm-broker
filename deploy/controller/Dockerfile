FROM alpine:3.14.0

RUN apk add --no-cache mercurial openssh
RUN apk add --no-cache ca-certificates curl git

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/start.sh"]

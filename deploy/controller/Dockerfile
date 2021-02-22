FROM alpine:3.13.2

RUN apk add --no-cache ca-certificates curl git openssh mercurial

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/start.sh"]

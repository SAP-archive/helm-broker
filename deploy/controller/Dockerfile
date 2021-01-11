FROM alpine:3.12

RUN apk --no-cache add ca-certificates curl git openssh mercurial

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/start.sh"]

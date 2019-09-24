FROM alpine:3.8

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache curl git openssh mercurial

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/root/start.sh"]

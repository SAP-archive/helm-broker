FROM alpine:3.8

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache curl
RUN apk add --no-cache curl git openssh

COPY ./start.sh /root/start.sh
COPY ./controller /root/controller

LABEL source=git@github.com:kyma-project/kyma.git

CMD ["/root/start.sh"]

FROM alpine:3.13

RUN apk add --no-cache ca-certificates curl git openssh mercurial

WORKDIR /app
COPY ./start.sh /app/start.sh
COPY ./controller /app/controller

LABEL source=git@github.com:kyma-project/helm-broker.git

CMD ["/app/start.sh"]

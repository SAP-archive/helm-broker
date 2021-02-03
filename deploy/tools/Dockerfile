FROM scratch

COPY ./targz /usr/local/bin/targz
COPY ./indexbuilder /usr/local/bin/indexbuilder

LABEL source=git@github.com:kyma-project/helm-broker.git

WORKDIR /app
ENTRYPOINT ["tail", "-f", "/dev/null"]


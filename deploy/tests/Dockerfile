FROM alpine:latest as builder
RUN apk --no-cache add ca-certificates

# creates a non-root user to give him write permissions to tmp folder
# needs for logger which saves logs under tmp dir
RUN mkdir /user && \
    echo 'appuser:x:2000:2000:appuser:/:' > /user/passwd && \
    echo 'appuser:x:2000:' > /user/group
RUN mkdir -p tmp

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ./hb_chart_test /usr/local/bin/hb_chart_test

COPY --from=builder /user/group /user/passwd /etc/

USER appuser:appuser

# appuser must be an owner of the tmp dir to write there
COPY --from=builder --chown=appuser /tmp /tmp

LABEL source=git@github.com:kyma-project/helm-broker.git

ENTRYPOINT ["hb_chart_test"]

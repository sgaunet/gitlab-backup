FROM alpine:3.22.1 AS alpine

FROM scratch AS final
LABEL maintainer="Sylvain Gaunet <sgaunet@gmail.com>"
WORKDIR /usr/local/bin
COPY gitlab-backup .
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY "resources" /
USER gitlab-backup

VOLUME [ "/data", "/tmp" ]

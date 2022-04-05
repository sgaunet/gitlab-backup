
FROM scratch AS final
LABEL maintainer="Sylvain Gaunet <sgaunet@gmail.com>"
WORKDIR /usr/local/bin
COPY gitlab-backup .
COPY "resources" /
USER gitlab-backup

VOLUME [ "/data" ]

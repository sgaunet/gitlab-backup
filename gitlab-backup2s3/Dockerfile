FROM sgaunet/gitlab-backup:0.2 AS gitlab-backup

FROM alpine:3.13.3
LABEL description="Backup gitlab projects to a S3"

RUN addgroup -S bkpgrp -g 1000 && adduser -S bkp -G bkpgrp --uid 1000

RUN apk update && apk add --no-cache bash aws-cli openssl \
    && mkdir /data \
    && chown bkp:bkpgrp /data

COPY --chown=bkp:bkpgrp src/* /app/
COPY --from=gitlab-backup /usr/bin/gitlab-backup /usr/bin/gitlab-backup
WORKDIR /app

USER bkp
CMD ["/app/gitlab-backup.sh"]
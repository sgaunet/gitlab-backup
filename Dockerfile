FROM golang:1.16.4-alpine AS builder
LABEL stage=builder

# ARG GITLAB_TOKEN
# ARG USERNAME

RUN apk add --no-cache git upx xz

#ENV GOPRIVATE="gitlab.com/.../.../libs/golibexample/golibexample"

# RUN echo "machine gitlab.com" > ~/.netrc
# RUN echo "    login ${USERNAME}" >> ~/.netrc
# RUN echo "    password ${GITLAB_TOKEN}" >> ~/.netrc

COPY src/  /go/src/gitlab-backup/src/
COPY src/go.mod /go/src/gitlab-backup/

WORKDIR /go/src/gitlab-backup/src
RUN go get && \
    CGO_ENABLED=0 GOOS=linux go build #-a gitlab-backup

RUN upx /go/src/gitlab-backup/src/gitlab-backup
##################################################################


FROM alpine:3.13.5

RUN apk add --no-cache bash
COPY --from=builder /go/src/gitlab-backup/src/gitlab-backup            /usr/bin/gitlab-backup
RUN chmod +x /usr/bin/gitlab-backup

VOLUME [ "/data" ]

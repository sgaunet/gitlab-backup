#!/usr/bin/env bash

docker build . -t sgaunet/gitlab-backup2s3:latest
rc="$?"

if [ "$rc" != "0" ]
then
    echo "FAILED !"
    exit 1
fi

docker push sgaunet/gitlab-backup2s3:latest
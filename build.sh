#!/usr/bin/env bash

docker build . -t sgaunet/gitlab-backup:latest
rc=$?

if [ "$rc" != "0" ]
then
    echo "Build Failed"
    exit 1
fi

docker push sgaunet/gitlab-backup:latest
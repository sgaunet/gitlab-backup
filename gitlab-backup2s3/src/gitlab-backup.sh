#!/bin/sh
##############################################################################################
# Crée le 23-04-2021 par SG
#
# Ce script copie les fichiers dans un S3
##############################################################################################
#

function copy2s3
{
    filetocopy="$1"
    s3endpoint="$2"
    filename=$(basename "$filetocopy")
    echo "Copy ${filetocopy} => s3://$s3endpoint/$filename"
    aws s3 cp "${filetocopy}" "s3://$s3endpoint/$filename"
}

function delete_if_copy_ok
{
    rc_copy="$1"
    fic="$2"
    
    if [ "$rc" = "0" ]
    then
        \rm "${fic}"
    fi
}

export TMPFILE=/tmp/tmp.$$

if [ "$MDP" = "" ]
then
    echo "Please set \$MDP"
    exit 1
fi

gitlab-backup -gid $GID -o /data -p 2

find /data -type f | while read i
do
    # Encrypt file
    openssl enc -e -aes-256-cbc -md sha512 -pbkdf2 -iter 100000 -salt -in "$i" -out "$i.enc" -pass pass:$MDP
    # decrypt
    # openssl enc -d -aes-256-cbc -md sha512 -pbkdf2 -iter 100000 -salt -in titi -out new -pass pass:YOYO
    /bin/rm "$i"

    copy2s3 "$i.enc" "$S3ENDPOINT"
done

rm /data/*

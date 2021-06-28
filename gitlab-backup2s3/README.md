
# gitlab-backup2s3

gitlab-backup2s3 is an enhanced docker image to export gitlab projects, encrypt the archive and save them in a S3.

## Configuration

Need to setup some environment variables :

```
GID: "8888"                                # GroupID of the gitlab projects to save (The root parent)
S3ENDPOINT: "aws-s3-endpoint/sources"      # S3 to save archives
GITLAB_TOKEN: "..."
MDP: "..."                                 # Token used to encrypt archive
```

## Example of deployment

In the deploy folder, you will find manifests to deploy a cronjob in kubernetes.
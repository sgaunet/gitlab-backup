# gitlab-backup

This project is for backup gitlab projects. Projects are actually saved one by one. 
I'm an ops, not a dev, the code will be improved soon.

# Usage


To download every projects of a group :

```
export GITLAB_TOKEN="...."
export GITLAB_URI="https://your-gitlab-uri"  # Optional, default https://gitlab.com
gitlab-backup -gid <main_group_id> [-o <path_to_save_archives>]
```

To download a single backup porject :

```
export GITLAB_TOKEN="...."
export GITLAB_URI="https://your-gitlab-uri"  # Optional, default https://gitlab.com
gitlab-backup -pid <project_id> [-o <path_to_save_archives>]
```


# Build

```
cd src
go build . -o gitlab-backup
```

# Test

Not yet.

# Gitlab API examples

Get the list of subgroups:

```
curl --header "PRIVATE-TOKEN: $GITLAB_TOKEN" https://gitlab.com/api/v4/groups/**id_of_group**/subgroups
```

Get the list of projects of a group:

```
curl --header "PRIVATE-TOKEN: $GITLAB_TOKEN" https://gitlab.com/api/v4/groups/**id_of_group**/projects
```

Ask an export:

```
curl --request POST --header "PRIVATE-TOKEN: $GITLAB_TOKEN" https://gitlab.com/api/v4/projects/**id_of_project**/export
```

Get the status of the export:

```
curl --header "PRIVATE-TOKEN: $GITLAB_TOKEN" https://gitlab.com/api/v4/projects/**id_of_project**/export
```

When the status is finished, download the backup:

```
curl --header "PRIVATE-TOKEN: $GITLAB_TOKEN" --remote-header-name --remote-name https://gitlab.com/api/v4/projects/**id_of_project**/export/download
```

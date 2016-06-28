## Making a dex Release

Creating a Dex release

* Creating a git tag.
* Building and pushing Docker images to Quay.
* Writing up release notes.

First, make sure you've [uploaded your GPG key](https://github.com/settings/keys)
and [configured git](https://git-scm.com/book/en/v2/Git-Tools-Signing-Your-Work)
to use that key. Note that the email of the key must also match the email you
use for git (`git config [--global] user.email "your_email@example.com"`).

Create a signed git tag in Dex corresponding to your release then push to the
CoreOS repo. The tag comment can just be the tag version.

```
$ git tag -s v0.4.0 ea4c04fde83bd6c48f4d43862c406deb4ea9dba2
$ git push 'git@github.com:coreos/dex.git' v0.4.0
```

Next checkout the tag and build the Docker image for the release and push to
Quay. This assumes you've logged in and have the correct credentials to write
to the repo.

```
$ git checkout tags/v0.4.0
$ ./build-image # use sudo if you don't have sudoless docker configured
$ docker push quay.io/coreos/dex:v0.4.0
$ docker push quay.io/coreos/dex:latest
```

Finally draft a release on GitHub summarizing the changes since the prevous
release. See existing release notes for formatting.

https://github.com/coreos/dex/releases

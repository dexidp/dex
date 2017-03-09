# Releases

Making a dex release involves:

* Tagging a git commit and pushing the tag to GitHub.
* Building and pushing a Docker image.

This requires the following tools.

* Docker

And the following permissions.

* Push access to the github.com/coreos/dex git repo.
* Push access to the quay.io/coreos/dex Docker repo.

## Tagging the release

Make sure you've [uploaded your GPG key](https://github.com/settings/keys) and
configured git to [use that signing key](
https://git-scm.com/book/en/v2/Git-Tools-Signing-Your-Work) either globally or
for the Dex repo. Note that the email the key is issued for must be the email
you use for git.

```
git config [--global] user.signingkey "{{ GPG key ID }}"
git config [--global] user.email "{{ Email associated with key }}"
```

Create a signed tag at the commit you wish to release. This action will prompt
you to enter a tag message, which can just be the release version.

```
git tag -s v2.0.0 ea4c04fde83bd6c48f4d43862c406deb4ea9dba2
```

Push that tag to the CoreOS repo.

```
git push git@github.com:coreos/dex.git v2.0.0
```

Draft releases on GitHub and summarize the changes since the last release. See
previous releases for the expected format.

https://github.com/coreos/dex/releases

## Minor releases - create a branch

If the release is a minor release (2.1.0, 2.2.0, etc.) create a branch for future patch releases.

```bash
git checkout -b v2.1.x tags/v2.1.0
git push git@github.com:coreos/dex.git v2.1.x
```

## Patch releases - cherry pick required commits

If the release is a patch release (2.0.1, 2.0.2, etc.) checkout the desired release branch and cherry pick specific commits. A patch release is only meant for urgent bug or security fixes.

```bash
RELEASE_BRANCH="v2.0.x"
git checkout $RELEASE_BRANCH
git checkout -b "cherry-picked-change"
git cherry-pick (SHA of change)
git push origin "cherry-picked-change"
```

Open a PR onto $RELEASE_BRANCH to get the changes approved.

## Building the Docker image

Build the Docker image and push to Quay.

```bash
# checkout the tag
git checkout tags/v2.1.0
# will prompt for sudo password
make docker-image
sudo docker push quay.io/coreos/dex:v2.1.0
```

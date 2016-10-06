# Releases

Making a dex release involves:

* Tagging a git commit and pushing the tag to GitHub.
* Building and pushing a Docker image.
* Building, signing, and hosting an ACI.

This requires the following tools.

* rkt
* Docker
* [docker2aci](https://github.com/appc/docker2aci)
* [acbuild](https://github.com/containers/build) (must be in your sudo user's PATH)

And the following permissions.

* Push access to the github.com/coreos/dex git repo.
* Push access to the quay.io/coreos/dex Docker repo.
* Access to the CoreOS application signing key.

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
git tag -s v2.1.0-alpha ea4c04fde83bd6c48f4d43862c406deb4ea9dba2
```

Push that tag to the CoreOS repo.

```
git push git@github.com:coreos/dex.git v2.1.0-alpha
```

Draft releases on GitHub and summarize the changes since the last release. See
previous releases for the expected format.

https://github.com/coreos/dex/releases

## Building the Docker image

Build the Docker image and push to Quay.

```bash
# checkout the tag
git checkout tags/v2.1.0-alpha
# rkt doesn't play nice with SELinux, see https://github.com/coreos/rkt/issues/1727
sudo setenforce Permissive
# will prompt for sudo password
make docker-image
sudo docker push quay.io/coreos/dex:v2.1.0-alpha
```

## Building the ACI

```bash
# checkout the tag
git checkout tags/v2.1.0-alpha
# rkt doesn't play nice with SELinux, see https://github.com/coreos/rkt/issues/1727
sudo setenforce Permissive
# will prompt for sudo password
make aci
# aci will be built at _output/image/dex.aci
```

Sign the ACI using the CoreOS application signing key. Upload the ACI and
signature to the GitHub release.

# Making a dex Release

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
git tag -s v0.4.0 ea4c04fde83bd6c48f4d43862c406deb4ea9dba2
```

Push that tag to the CoreOS repo.

```
git push git@github.com:coreos/dex.git v0.4.0
```

Draft releases on GitHub and summarize the changes since the last release. See
previous releases for the expected format.

https://github.com/coreos/dex/releases

Finally create an image tag on Quay corresponding to the release. Log into
Quay, navigate to the `quay.io/coreos/dex` repo, find the correct commit, and
add an additional tag to that image for the release (click the gear on the
image tag's row and then "Add New Tag").

https://quay.io/repository/coreos/dex?tag=latest&tab=tags

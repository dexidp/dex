# dex - A federated OpenID Connect provider

![logo](docs/logos/dex-horizontal-color.png)

This repository is a fork of [dexidp/dex](https://github.com/dexidp/dex).

[Giant Swarm](https://www.giantswarm.io/) uses Dex for [authentication to our Platform API](https://docs.giantswarm.io/overview/architecture/authentication/) and offer it as part of [auth-bundle managed app](https://github.com/giantswarm/auth-bundle) for our customer to enable authentication capabilities in a Giant Swarm cluster.

## Release Process

We follow same [release process as upstream](https://dexidp.io/docs/development/releases/), so please follow the same.

Upon completing these release steps, the final tagged image will be available:

```sh
docker pull quay.io/giantswarm/dex:vX.Y.Z-gsN
```

### Post-Release Actions

After publishing a release:

- A Dependabot PR should be automatically created in the dex-app repository to bump the newly released version in the Dockerfile. If not, you can also trigger a check for updates from the Dependency graph from [giantswarm/dex-app/network/updates](https://github.com/giantswarm/dex-app/network/updates)


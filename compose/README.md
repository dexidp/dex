# Build Status

[![CircleCI](https://circleci.com/gh/circleci/circle-dex/tree/master.svg?style=svg&circle-token=349726b0dd98728f82d7c186b2c3fffa0d58f19b)](https://circleci.com/gh/circleci/circle-dex/tree/master)

# CircleCI's Dex-fu

Being a place to store docker configs and other Dex miscellania

## Run

First, add `127.0.0.1 dex.com` to your hosts file. (TODO: rename this to something less likely to conflict with real domains, like `circle-dex-dev.com`, and change redirect URIs)

This is necessary because Google only allows OAuth redirects to URLs with externally accessible TLDs like `.com`.

And then bring up the services here using:

```
$ docker-compose up -d
```

If you're modifying the `Dockerfile`, run this instead to have docker-compose rebuild images before bringing them up:

```
$ docker-compose up --build
```

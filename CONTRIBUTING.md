# Contributing to Dex

Dex is [Apache 2.0 licensed](LICENSE) and accepts contributions via GitHub pull requests.
This document outlines how to contribute to the project.

- [Code of Conduct](#code-of-conduct)
- [Finding something to work on](#finding-something-to-work-on)
- [Setting up a development environment](#setting-up-a-development-environment)
- [Making changes](#making-changes)
- [Running the example app](#running-the-example-app)
- [Committing your changes](#committing-your-changes)
- [Submitting a pull request](#submitting-a-pull-request)
- [Enhancement proposals](#enhancement-proposals)
- [Getting help](#getting-help)

## Code of Conduct

This project follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

## Finding something to work on

If you have a bug fix or a small improvement, go ahead and open a pull request.

For larger changes, please open a [discussion](https://github.com/dexidp/dex/discussions/new?category=Ideas) first
to align with the community and avoid unnecessary work. Major features or significant architectural
changes should go through the [Enhancement Proposal](#enhancement-proposals) process.

If you're looking for something to work on, check:

- Issues labeled with [good first issue](https://github.com/dexidp/dex/labels/good%20first%20issue)
- Issues labeled with [help wanted](https://github.com/dexidp/dex/labels/help%20wanted)

Please comment on the issue to claim it before starting work to avoid duplicated efforts.

## Setting up a development environment

For the best developer experience, install [Nix](https://builtwithnix.org/) and [direnv](https://direnv.net/).
This will automatically set up all required tools (Go, golangci-lint, protobuf compiler, kind, etc.).

Alternatively, you can set up the environment manually:

1. Install [Go](https://go.dev/doc/install) (see the version in [go.mod](go.mod)).
2. Install [Docker](https://docs.docker.com/get-started/).
3. Install development dependencies:

```shell
make deps
```

This installs `golangci-lint`, `gotestsum`, `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, and `kind`.

You can also use [GitHub Codespaces](https://github.com/codespaces/new?repo=dexidp/dex) for a ready-to-code cloud environment.

## Making changes

Run `make help` to see all available commands. The key ones for contributors:

```shell
make deps        # Install development dependencies
make build       # Build Dex binaries
make testall     # Run all tests (includes race detection)
make lint        # Run linter
make generate    # Regenerate protobuf, ent, and go mod tidy
```

## Running the example app

To test the login flow locally, run Dex with the dev config and the example OIDC client app.

**Terminal 1** — start Dex:

```shell
make build
./bin/dex serve config.dev.yaml
```

**Terminal 2** — start the example app:

```shell
make examples
./bin/example-app
```

Open http://127.0.0.1:5555 in your browser, click "Login", and authenticate with:

- **Email:** `admin@example.com`
- **Password:** `password`

After successful login, the example app displays the ID token claims returned by Dex.

## Committing your changes

The project follows [Conventional Commits](https://www.conventionalcommits.org/) style:

```
<type>[optional scope]: <description>

[optional body]

Signed-off-by: First Last <email@example.com>
```

Common types: `feat`, `fix`, `build`, `chore`, `docs`, `refactor`, `test`.

Examples from the project:

```
feat: use protobuf for session cookie (#4675)
fix: non-constant format string in call to newRedirectedErr (#4671)
build(deps): bump github/codeql-action from 4.33.0 to 4.34.1 (#4679)
```

### Developer Certificate of Origin

As a CNCF project, Dex requires all contributors to sign the [Developer Certificate of Origin (DCO)](https://developercertificate.org/).
This certifies that you have the right to submit your contribution under the project's open source license.

You must add a `Signed-off-by` line to every commit. Use git's `-s` flag to do this automatically:

```shell
git commit -s -m "feat: add new feature"
```

The DCO check will fail on pull requests with unsigned commits.

## Submitting a pull request

1. Fork the repository and create your branch from `master`.
2. Make your changes, following the guidelines above.
3. Ensure all tests pass (`make testall`) and the linter is clean (`make lint`).
4. Ensure generated code is up to date (`make generate`).
5. Push your branch and open a pull request.

When opening a pull request:

- Fill in the [pull request template](.github/PULL_REQUEST_TEMPLATE.md) with an overview and explanation of the change.
- After opening a PR, a maintainer will add least one [release note label](https://github.com/dexidp/dex/labels?q=release-note) to the PR.
  Valid labels include: `kind/feature`, `kind/enhancement`, `kind/bug`, `release-note/new-feature`,
  `release-note/enhancement`, `release-note/bug-fix`, `release-note/breaking-change`,
  `release-note/deprecation`, `release-note/ignore`, `area/dependencies`, `release-note/dependency-update`.
- If the PR is still in progress, use GitHub's [Draft PR](https://github.blog/2019-02-14-introducing-draft-pull-requests/) feature.

All CI checks (tests, linting, DCO, release label) must pass before the PR can be merged.

## Enhancement proposals

Significant features or architectural changes require a [Dex Enhancement Proposal (DEP)](docs/enhancements/README.md).

The process:

1. Search existing [issues](https://github.com/dexidp/dex/issues), [discussions](https://github.com/dexidp/dex/discussions), and [DEPs](https://github.com/dexidp/dex/tree/master/docs/enhancements).
2. Open a [discussion](https://github.com/dexidp/dex/discussions/new?category=Ideas) to get initial feedback.
3. Fork the repo and copy the [DEP template](docs/enhancements/_title-YYYY-MM-DD-#issue.md) with an appropriate name.
4. Fill in all sections and submit a PR for review.

## Getting help

- For bugs and feature requests, file an [issue](https://github.com/dexidp/dex/issues).
- For general discussion, open a [discussion](https://github.com/dexidp/dex/discussions) or join [#dexidp](https://cloud-native.slack.com/messages/dexidp) on the CNCF Slack.
- Mailing list (as a backup): [dex-dev](https://groups.google.com/forum/#!forum/dex-dev).
- For security vulnerabilities, see the [security policy](.github/SECURITY.md).

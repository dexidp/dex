# How to Contribute

CoreOS projects are [Apache 2.0 licensed](LICENSE) and accept contributions via
GitHub pull requests.  This document outlines some of the conventions on
development workflow, commit message formatting, contact points and other
resources to make it easier to get your contribution accepted.

# Certificate of Origin

By contributing to this project you agree to the Developer Certificate of
Origin (DCO). This document was created by the Linux Kernel community and is a
simple statement that you, as a contributor, have the legal right to make the
contribution. See the [DCO](DCO) file for details.

# Email and Chat

The project currently uses the general CoreOS email list and IRC channel:
- Email: [coreos-dev](https://groups.google.com/forum/#!forum/coreos-dev)
- IRC: #[coreos](irc://irc.freenode.org:6667/#coreos) IRC channel on freenode.org

Please avoid emailing maintainers found in the MAINTAINERS file directly. They
are very busy and read the mailing lists.

## Getting Started

- Fork the repository on GitHub
- Read the [README](README.md) for build and test instructions
- Play with the project, submit bugs, submit patches!

## Contribution Flow

This is a rough outline of what a contributor's workflow looks like:

- Create a proposal (if neccessary - see below) and get an LGTM by someone in [maintainer](MAINTAINERS).
- Create a topic branch from where you want to base your work (usually master).
- Make commits of logical units.
- Make sure your commit messages are in the proper format (see below).
- Push your changes to a topic branch in your fork of the repository.
- Make sure the tests pass, and add any new tests as appropriate.
- Submit a pull request to the original repository.

Thanks for your contributions!

**Note**: When editing documentation you should follow the [CoreOS Docs Style Guide][coreos-docs-style].

## Proposals

For very simple contributions - bug fixes, documentation tweaks, small optimizations, etc. - a proposal is not neccesary. Otherwise, it's necessary to discuss your proposal with other members of the community and get approval from the maintainers. 

To create a proposal, copy the markdown from the [proposal template](PROPOSAL_TEMPLATE.md) and put it in the relevant issue. Once you've gotten an LGTM from a [maintainer](MAINTAINERS), you can proceed with putting together a PR.

Don't worry if you're proposal is not accepted right away; you'll probably need to make some changes, depending on the scope of the proposal.

Here's a link which creates a new issue populated with the proposal link:

[Create a Proposal](https://github.com/coreos/dex/issues/new?body=Proposal%0A%3D%3D%3D%0A%0A%28Feel%20free%20to%20change%20headings%20here%2C%20remove%20sections%20that%20are%20not%20relevant%2C%20or%20add%20other%20sections%29%0A%0A%23%23%20Background%0A%0ADescribe%20what%20problem%20is%20being%20solved%20here%2C%20and%20%28briefly%29%20how%20this%20proposal%20solves%20it.%0A%0A%23%23%20Data%20Model%0A%0ADescribe%20the%20logical%20data%20model%20that%20your%20proposal%20adds.%0A%0A%23%23%20Data%20Storage%0A%0ADescribe%20how%20the%20data%20will%20be%20persisted.%0A%0A%23%23%20API%0A%0ADescribe%20the%20methods%20that%20the%20API%20will%20expose.%20If%20there%20are%20any%20breaking%20changes%20be%20sure%20to%20call%20them%20out%20here.%0A%0A%23%23%20UI/UX%0A%0AIs%20there%20a%20front-end%20component%20to%20this%20work%3F%0A%0A%23%23%20Implementation%0A%0AHere%20is%20where%20you%20can%20go%20into%20detail%20about%20implementation%20details%20like%20data%20structures%2C%20algorithms%2C%20etc.%0A%0A%23%23%20Security%0A%0AWhat%20are%20the%20security%20implications%20of%20this%20proposal%3F%20How%20are%20API%20requests%20authenticated%3F%20Who%20can%20make%20API%20calls%3F%0A%0A%23%23%20OIDC/OAUTH2%0A%0ADoes%20this%20feature%20relate%20to%20any%20spec%3F%20%0A%0A%23%23%20Risks/Alternatives%20Considered%0A%0AWhat%20are%20the%20downsides%20to%20this%20implementation%3F%20What%20other%20alternatives%20were%20considered%3F%0A%0A%23%23%20References%0A%0AIf%20there%27s%20any%20references%20or%20prior%20art%2C%20put%20that%20here.)

### Format of the Commit Message

We follow a rough convention for commit messages that is designed to answer two
questions: what changed and why. The subject line should feature the what and
the body of the commit should describe the why.

```
scripts: add the test-cluster command

this uses tmux to setup a test cluster that you can easily kill and
start for debugging.

Fixes #38
```

The format can be described more formally as follows:

```
<subsystem>: <what changed>
<BLANK LINE>
<why this change was made>
<BLANK LINE>
<footer>
```

The first line is the subject and should be no longer than 70 characters, the
second line is always blank, and other lines should be wrapped at 80 characters.
This allows the message to be easier to read on GitHub as well as in various
git tools.

[coreos-docs-style]: https://github.com/coreos/docs/blob/master/STYLE.md

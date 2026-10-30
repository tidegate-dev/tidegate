# AGENTS.md

This file is for AI coding tools that work on tidegate. `CLAUDE.md` is a symlink to it.

## What tidegate is

tidegate is an open source privileged access management tool. It decides who may reach production systems, for how long and with which permissions. It's in early development. A design mistake here becomes a security hole later, so when something is unclear, ask instead of guessing.

## Where decisions are made

Significant changes are decided in a Tidegate Enhancement Proposal (TEP), a design document in the separate repository [tidegate-dev/enhancements](https://github.com/tidegate-dev/enhancements). That covers new user-facing features and components, changes to how tidegate authenticates, authorizes or handles credentials, breaking changes to an API or a format, and new dependencies that tidegate's security relies on. If an issue says a decision belongs in a TEP, or a change fits this list and no TEP covers it, don't make the decision in code. Point it out and ask.

## Checking your work

`just ci` runs every check that CI runs. Run it before you call a change done. Its integration tests need Postgres, so start `just dev-deps` first.

If `just` isn't on your PATH, the tools from the Nix dev shell are missing. Run the command as `nix develop --command just ci`. If that fails too, ask instead of installing tools.

The dev shell installs a pre-commit hook that runs lint and the tests. If it refuses a commit, fix the cause and commit again. Never skip it with `--no-verify`.

## Commits and pull requests

[CONTRIBUTING.md](CONTRIBUTING.md) explains how commits are signed off, how pull requests are titled and how to disclose that an AI tool helped. Follow it. In short: never add a `Signed-off-by` line, because only a person can certify the Developer Certificate of Origin, and give every commit you helped write an `Assisted-by` trailer with your name and exact model version, in the format CONTRIBUTING.md shows.

A maintainer reviews and merges every pull request. Don't merge one yourself.

Don't add license or copyright headers to files. `REUSE.toml` sets the license for every file in the repository.

## Security

tidegate guards access to infrastructure, so a vulnerability in it is worse than in most projects. If you find one, don't describe it in a public issue, pull request or commit message. Tell the person you're working with and point them to [SECURITY.md](SECURITY.md), which explains how to report it privately.

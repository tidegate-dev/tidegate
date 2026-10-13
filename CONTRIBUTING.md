# Contributing to tidegate

Thanks for your interest in contributing. This guide explains how to propose a change and what we look for in a pull request.

Please follow our [Code of Conduct](CODE_OF_CONDUCT.md) in all project spaces.

## Before you start

For bug fixes and small changes, open a pull request directly. For larger changes or new features, open an issue first and describe what you want to do. This way we can agree on the approach before you spend time on it.

## Making a change

1. Fork the repository and create a branch for your change.
2. Make your change and sign off every commit (see [Developer Certificate of Origin](#developer-certificate-of-origin)).
3. Open a pull request against `main`.
4. A maintainer reviews the pull request and may ask for changes.
5. Once it is approved, a maintainer squash merges it.

## Pull request titles

Pull requests are squash merged, so the pull request title becomes the commit message on `main`. Titles must follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<optional scope>): <description>
```

For example:

- `feat: add SSH session recording`
- `fix(proxy): close idle connections`
- `docs: explain how to run the tests`

Allowed types are `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style` and `test`. For a breaking change, add `!` after the type or scope, for example `feat!: remove the v1 API`.

A check on every pull request validates the title. The commit messages inside your branch do not have to follow this format.

## Developer Certificate of Origin

All contributions must be signed off under the [Developer Certificate of Origin](https://developercertificate.org/) (DCO). By signing off, you confirm that you wrote the change or have the right to submit it under the project's license.

To sign off a commit, add `-s` to `git commit`:

```sh
git commit -s -m "fix: close idle connections"
```

This adds a line like this to the commit message:

```
Signed-off-by: Jane Doe <jane@example.com>
```

The email in this line must match the commit author. A check on every pull request makes sure all commits are signed off.

If you forgot to sign off, you can add the sign-off to all commits on your branch and update the pull request:

```sh
git rebase --signoff main
git push --force-with-lease
```

## Using AI tools

You can use any tools for your contribution, including AI tools. You stay responsible for the result:

- Check everything an AI tool wrote before anyone else has to look at it.
- Be ready to answer questions about your change in the review.
- A tool may help you write the pull request description, but check that it says what you mean. The description explains why you made the change, how it works and what's still open.
- If a tool wrote a substantial part of the change, say so in the pull request description, and name the tool and what it did. For example: "The tests were written with an AI assistant." This tells reviewers which parts to read more closely.
- Don't paste review comments into an AI tool and push whatever it returns without reading it.

The same applies to issues and to comments on pull requests and issues. Read what the tool wrote before you post it, and if a tool wrote a substantial part, say so.

If you report a security vulnerability with the help of an AI tool, reproduce the problem yourself first and describe how you did it.

You're responsible for everything an AI tool does in your name, such as the pull requests and comments it posts.

Every change costs reviewers time, and AI output that you haven't checked yourself pushes that work onto them. A maintainer may ask for changes to a pull request, issue or comment that doesn't follow these rules, or close or hide it.

If you're new to tidegate, start with small changes that you fully understand.

Your sign-off under the [DCO](#developer-certificate-of-origin) still means you have the right to submit the change. If an AI tool reproduces someone else's code, that doesn't give you that right, so check for it.

These rules are adapted from the [LLVM AI Tool Use Policy](https://llvm.org/docs/AIToolPolicy.html), which builds on a policy of the Fedora project.

## License

By contributing, you agree that your contributions are licensed under the [Apache License 2.0](LICENSE).

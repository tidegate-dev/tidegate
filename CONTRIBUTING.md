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

You may use AI tools to write code and text for tidegate. A change made with an AI tool follows the same rules as any other change.

An AI tool can't certify the [Developer Certificate of Origin](#developer-certificate-of-origin), so it must not add a `Signed-off-by` line. Before you sign off, review the change, make sure its license fits the project and take full responsibility for it.

If an AI tool helped write a commit, add an `Assisted-by` line to the commit message:

```
Assisted-by: AGENT_NAME:MODEL_VERSION [TOOL1] [TOOL2]
```

`AGENT_NAME` is the AI tool and `MODEL_VERSION` is the exact model it used. After them you may list analysis tools that helped, like a linter or a static analyzer. Don't list basic tools like git, the compiler or your editor. For example:

```
Assisted-by: Claude:claude-opus-5-5
```

The squash merge keeps the commit messages from your branch, so this line ends up on `main`.

## License

By contributing, you agree that your contributions are licensed under the [Apache License 2.0](LICENSE).

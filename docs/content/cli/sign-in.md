# Signing in and out

A command needs a session at a phemera installation. `phemera login` creates the session and saves it, so you don't pass a URL and a password to every command:

```sh
phemera login https://phemera.example
```

`phemera login` reads your user name and password from the terminal and requests a session token. It saves the URL and the token as a connection. Later commands read the connection:

```sh
phemera project list
```

`phemera logout` revokes the session token and removes it from the connection. The URL stays saved, so `phemera login` without a URL signs in again.

The URL must use `https`. `http` is accepted only for `localhost`, `127.0.0.1` and `::1`. If the session has expired or was revoked, a command fails and prints the `phemera login` command to run.

## Several installations

```sh
phemera login https://phemera.example --connection production
phemera login https://staging.phemera.example --connection staging
```

`--connection` sets the name of the connection. Without it, the name is the URL as given. A name maps to one URL: `phemera login <url> --connection <name>` fails if the name is saved for a different URL. Two connections may have the same URL, for example for two accounts.

If one connection is saved, every command uses it. If several are saved, select one with `--connection` or with `PHEMERA_CONNECTION`:

```sh
phemera project list --connection staging
export PHEMERA_CONNECTION=production
```

The flag takes precedence over the variable. Without either, the command fails and lists the saved names. The selection isn't stored, so it doesn't carry over from one shell to another.

`phemera login --connection staging` signs in to a saved connection again and revokes the session it replaces.

## Listing and removing connections

```sh
phemera connections list
phemera connections current
phemera connections remove staging
```

`list` prints the name and the URL of every connection and whether it holds a session token. It doesn't print the token. `current` prints the connection that a command in this shell selects. Neither calls the authority.

`remove` revokes the session token and deletes the connection. If the authority can't be reached, `remove` deletes the connection and warns that the session stays valid until it expires. `logout` fails in that case and keeps the token.

## Scripts

```sh
export PHEMERA_SERVER=https://phemera.example
export PHEMERA_TOKEN=$(phemera login "$PHEMERA_SERVER" --print-token < credentials.txt)

phemera project list
phemera logout
```

`--print-token` writes the session token to stdout and saves no connection. If stdin isn't a terminal, `phemera login` reads the user name from the first line of stdin and the password from the second.

If `PHEMERA_SERVER` or `PHEMERA_TOKEN` is set, a command uses these two and doesn't read the saved connections. `--connection` together with either is an error. `phemera logout` revokes the token in `PHEMERA_TOKEN`.

[Reference](reference.md) says where the connections are saved and which file permissions phemera requires.

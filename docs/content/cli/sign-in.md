# Signing in and out

A command needs a session at a tidegate installation. `tidegate login` creates the session and saves it, so you don't pass a URL and a password to every command:

```sh
tidegate login https://tidegate.example
```

`tidegate login` reads your user name and password from the terminal and requests a session token. It saves the URL and the token as a connection. Later commands read the connection:

```sh
tidegate ping
```

`tidegate logout` revokes the session token and removes it from the connection. The URL stays saved, so `tidegate login` without a URL signs in again.

The URL must use `https`. `http` is accepted only for `localhost`, `127.0.0.1` and `::1`. If the session has expired or was revoked, a command exits with status 1 and prints the `tidegate login` command to run.

## Several installations

```sh
tidegate login https://tidegate.example --connection production
tidegate login https://staging.tidegate.example --connection staging
```

`--connection` sets the name of the connection. Without it, the name is the URL as given. A name maps to one URL: `tidegate login <url> --connection <name>` fails if the name is saved for a different URL. Two connections may have the same URL, for example for two accounts.

If one connection is saved, every command uses it. If several are saved, select one with `--connection` or with `TIDEGATE_CONNECTION`:

```sh
tidegate ping --connection staging
export TIDEGATE_CONNECTION=production
```

The flag takes precedence over the variable. Without either, the command fails and lists the saved names. The selection isn't stored, so it doesn't carry over from one shell to another.

`tidegate login --connection staging` signs in to a saved connection again and revokes the session it replaces.

## Listing and removing connections

```sh
tidegate connections list
tidegate connections current
tidegate connections remove staging
```

`list` prints the name and the URL of every connection and whether it holds a session token. It doesn't print the token. `current` prints the connection that a command in this shell selects. Neither calls the authority.

`remove` revokes the session token and deletes the connection. If the authority can't be reached, `remove` deletes the connection and warns that the session stays valid until it expires. `logout` fails in that case and keeps the token.

## Scripts

```sh
export TIDEGATE_SERVER=https://tidegate.example
export TIDEGATE_TOKEN=$(tidegate login "$TIDEGATE_SERVER" --print-token < credentials.txt)

tidegate ping
tidegate logout
```

`--print-token` writes the session token to stdout and saves no connection. If stdin isn't a terminal, `tidegate login` reads the user name from the first line of stdin and the password from the second.

If `TIDEGATE_SERVER` or `TIDEGATE_TOKEN` is set, a command uses these two and doesn't read the saved connections. `--connection` together with either is an error. `tidegate logout` revokes the token in `TIDEGATE_TOKEN`.

[Reference](reference.md) says where the connections are saved and which file permissions tidegate requires.

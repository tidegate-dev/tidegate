# Reference

```sh
phemera [--connection <name>] [--output <format>] <command>
```

Global flags work before or after the command. A command that fails prints `Error:` and a message to stderr and exits with status 1. `phemera connect` is the exception.

`phemera connect` exits with the status that the shell ended with, like `ssh` does, and prints no message for it. If the host reports a status above 255, `phemera connect` exits with status 255. If `phemera connect` itself fails, for example because it can't reach the ssh-proxy, it prints `Error:` and a message and exits with status 255. Only a mistake in a flag exits with status 1, like a flag that it doesn't know or a flag without its value.

## Commands

| Command | What it does |
|---|---|
| `phemera login <url>` | Signs in at the URL and saves the session as a connection |
| `phemera login` | Signs in to a saved connection again and revokes the session it replaces |
| `phemera logout` | Revokes the session token and removes it from the connection |
| `phemera connections list` | Prints the name and the URL of every saved connection and whether it holds a session token |
| `phemera connections current` | Prints the saved connection that a command selects |
| `phemera connections remove <name>` | Revokes the session token and deletes the connection |
| `phemera project create <name>` | Creates a project and prints its ID and name |
| `phemera project list` | Prints the ID and the name of every project |
| `phemera role create --project-name <project> <name>` | Creates a role in the project and prints its ID and name |
| `phemera asset create ssh --project-name <project> --address <address> --host-key <key> --user-name <user> (--password \| --private-key-file <file>) <name>` | Adds an SSH host to the project as an asset and prints the asset's ID and name |
| `phemera asset list --project-name <project>` | Prints the ID, the name, the address and the labels of every asset of the project |
| `phemera connect <asset-id>` | Requests a ticket for the asset and opens a shell through the ssh-proxy on `127.0.0.1:2022`. Issuance currently assumes the user has a binding whose role permits access. The ssh-proxy is a prototype that connects every session to the test SSH host of the development environment, whatever the asset is. |
| `phemera binding create --project-name <project> --role-name <role> --user-name <user>` | Gives the role of the project to the user and prints the binding's ID, the role and the user |
| `phemera binding list --project-name <project>` | Prints the ID, the role and the user of every binding of the project |

## Flags

| Flag | Commands | Values |
|---|---|---|
| `--connection <name>`, `-c <name>` | All | The name of a saved connection. With `phemera login <url>`, the name to save the connection under. |
| `--output <format>`, `-o <format>` | All | `text`, `json` or `yaml`. The default is `text`. The `text` format escapes some characters for terminal display. To recover the original text you can use the `json` format and decode the text from there. |
| `--print-token` | `login <url>` | Writes the session token to stdout and saves no connection. Can't be combined with `--connection`. |
| `--project-name <project>` | `role create`, `asset create ssh`, `asset list`, `binding create`, `binding list` | The name of the project that gets the role, the asset or the binding, or whose assets or bindings are listed. Required. |
| `--create-roles` | `role create` | The holders of the role may create roles in the project. |
| `--escalate` | `role create` | The holders of the role may create roles with permissions that they don't have themselves. |
| `--create-bindings` | `role create` | The holders of the role may create bindings in the project. |
| `--bind` | `role create` | The holders of the role may bind roles with permissions that they don't have themselves. |
| `--address <address>` | `asset create ssh` | The host's DNS name or IP address, as `host` or `host:port`. Without a port, it's 22. An IPv6 address with a port needs brackets, like `[2001:db8::7]:2222`. Required. |
| `--host-key <key>` | `asset create ssh` | The host's public key, like the content of `/etc/ssh/ssh_host_ed25519_key.pub`. A comment after the key is dropped. Required. |
| `--label <key>=<value>` | `asset create ssh` | A label that roles select the asset by, like `env=prod`. Can be given several times. |
| `--user-name <user>` | `asset create ssh` | The user that phemera logs in as on the host. Required. |
| `--password` | `asset create ssh` | The user logs in with a password. The command asks for it on a terminal, or reads it from the first line of stdin. The command needs this flag or `--private-key-file`, and not both. |
| `--private-key-file <file>` | `asset create ssh` | The user logs in with the private key in the file. The key has no passphrase. The command needs this flag or `--password`, and not both. |
| `--role-name <role>` | `binding create` | The name of the role that the user gets. The role must belong to the project. Required. |
| `--user-name <user>` | `binding create` | The name of the user who gets the role. Uppercase and lowercase letters count as the same. Required. |

## Environment variables

| Variable | Values |
|---|---|
| `PHEMERA_CONNECTION` | The name of a saved connection, used when `--connection` isn't given. `phemera login <url>` ignores it. |
| `PHEMERA_SERVER` | The URL of an installation. If it is set, commands use it with `PHEMERA_TOKEN` and no saved connection. |
| `PHEMERA_TOKEN` | A session token from `phemera login <url> --print-token`. Requires `PHEMERA_SERVER`. |

`phemera login` ignores `PHEMERA_SERVER` and `PHEMERA_TOKEN`.

## Which connection a command uses

A command takes the first of these that applies:

1. `PHEMERA_SERVER` and `PHEMERA_TOKEN`, if either is set. `--connection` is then an error.
2. The saved connection named by `--connection`.
3. The saved connection named by `PHEMERA_CONNECTION`.
4. The only saved connection.

If none applies, the command fails. With several saved connections, the error lists their names.

`phemera login <url>` saves the connection under the first of these names:

1. The name from `--connection`. If that name is saved for a different URL, the command fails.
2. The name of the only saved connection that has this URL.
3. The URL as given.

If several saved connections have this URL and no name is given, the command fails and lists them.

## Output

| Format | Result |
|---|---|
| `text` | The result's fields, separated by tabs, like the ID and the name that `project create` prints. `connections list`, `project list`, `asset list` and `binding list` print a table. |
| `json` | An object with the result's fields, like `{"id": "…", "name": "payments"}` |
| `yaml` | A mapping with the result's fields, like `name: payments` |

`phemera login` and `phemera logout` print nothing to stdout, unless `--print-token` is given. Prompts and warnings go to stderr.

## Saved connections

phemera saves the connections in `phemera/connections.json` in your configuration directory:

| System | Configuration directory |
|---|---|
| Linux | `$XDG_CONFIG_HOME`, or `~/.config` if it isn't set |
| macOS | `~/Library/Application Support` |
| Windows | `%AppData%` |

The file holds session tokens. phemera creates the directory with mode `0700` and the file with mode `0600`. On Linux and macOS, it refuses to read or write them if any of these is true, and the error gives the `chmod` command that fixes the mode:

- The directory or the file has a permission for the group or for others.
- The directory or the file is a symbolic link.
- The directory or the file belongs to another user.

phemera changes the file while it holds the lock file `connections.json.lock` in the same directory. A second phemera process waits up to 10 seconds for the lock and then fails.

If there is no configuration directory, like in a container that sets no `HOME`, commands that use a saved connection fail. Set `PHEMERA_SERVER` and `PHEMERA_TOKEN` there.

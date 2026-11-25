# ssh-proxy

The ssh-proxy carries an SSH session from a user to a host. The user never holds the host's password, because the ssh-proxy logs in to the host itself and relays the session.

Note that the ssh-proxy is a prototype. It doesn't call the authority yet. It connects every session to the test SSH host of the development environment, and it accepts tickets that nobody signed. Only run it for development.

## Open a session

First, start tidegate with `just dev`. Then create a key and open a session with it:

```sh
ssh-keygen -q -t ed25519 -N '' -f /tmp/tidegate-session-key
ticket=$(ssh-keygen -lf /tmp/tidegate-session-key.pub | cut -d ' ' -f 2)

ssh -i /tmp/tidegate-session-key -o IdentitiesOnly=yes \
    -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -o ConnectTimeout=5 \
    -o ProxyCommand="sh -c '{ echo \"tidegate-session-v1 $ticket\"; cat; } | nc 127.0.0.1 2022'" \
    session@ssh-proxy
```

This opens a shell on the test SSH host as its user `tidegate`. To run a command instead, add it at the end, like with any `ssh` call.

A connection to the ssh-proxy starts with one line that names a ticket: `tidegate-session-v1`, a space, the ticket and a line break. The SSH connection follows on the same connection. In the example, `ProxyCommand` sends that line. A ticket belongs to one SSH key, and the ssh-proxy refuses a client that doesn't hold this key. For now, a ticket is the SHA256 fingerprint of its key, as `ssh-keygen -lf` prints it.

The ssh-proxy ignores the user name that the client sends, which is `session` in the example. It creates a new host key every time it starts, so the example turns off the host key check of `ssh`.

## What the ssh-proxy does with a session

First, the ssh-proxy reads the ticket and checks that the client holds the ticket's key. Then it connects to the host and compares the key that the host presents with the host key it expects. Only if the two match, it fetches the password and logs in. A host with another key never gets the password. Finally, it relays the session between the client and the host.

The ssh-proxy relays shells and commands, with a terminal, changes of the terminal's size, and the variables `LANG` and `LC_*`. It refuses port forwarding in both directions, agent forwarding, X11 forwarding, subsystems like SFTP, and every other variable.

[TEP-0005](https://github.com/tidegate-dev/enhancements/tree/main/teps/0005-gateway-sessions) describes the whole path of a session. It calls the ssh-proxy the SSH worker.

## Settings

You configure the ssh-proxy with environment variables:

```sh
TIDEGATE_SSH_PROXY__LISTEN_ADDRESS=127.0.0.1:2200 \
TIDEGATE_SSH_PROXY__LOG__FORMAT=text \
ssh-proxy
```

This makes the ssh-proxy listen on port 2200 of the local machine and write logs that are easy to read.

Every variable starts with `TIDEGATE_SSH_PROXY__`. Two underscores separate the parts of a name. After the prefix, upper and lower case don't matter in names, or in values like `debug`. A setting keeps its default when its variable isn't set or is empty. If a value is invalid, or a variable starts with `TIDEGATE_SSH_PROXY__` but isn't a setting, the ssh-proxy stops and lists each problem.

| Variable | Default | Values |
|---|---|---|
| `TIDEGATE_SSH_PROXY__LISTEN_ADDRESS` | `127.0.0.1:2022` | A host and port, like `:2022` or `[::1]:2022` |
| `TIDEGATE_SSH_PROXY__LOG__LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `TIDEGATE_SSH_PROXY__LOG__FORMAT` | `json` | `json` for log collectors, or `text`, which is easier to read locally |

bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
}

@test "login refuses a plain HTTP server that isn't on this machine" {
    run --separate-stderr phemera login http://phemera.example < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"refusing to send a password or a token over plain HTTP"* ]]
}

@test "login refuses an argument that isn't a URL" {
    run --separate-stderr phemera login production < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"invalid server URL"* ]]
    [ ! -e "$saved" ]
}

@test "a call with a session token refuses a plain HTTP server that isn't on this machine" {
    PHEMERA_SERVER=http://phemera.example PHEMERA_TOKEN=secret run --separate-stderr phemera project create payments
    [ "$status" -eq 1 ]
    [[ $stderr == *"refusing to send a password or a token over plain HTTP"* ]]
}

@test "a saved connection to a plain HTTP server that isn't on this machine is refused" {
    save '{"connections": {"plain": {"server": "http://phemera.example", "token": "secret"}}}'

    run --separate-stderr phemera project create payments
    [ "$status" -eq 1 ]
    [[ $stderr == *"refusing to send a password or a token over plain HTTP"* ]]
}

@test "an authority in PHEMERA_SERVER that can't be reached is named in the error" {
    PHEMERA_SERVER=https://127.0.0.1:9 run --separate-stderr phemera project list
    [ "$status" -eq 1 ]
    [[ $stderr == *"can't reach the authority at https://127.0.0.1:9"* ]]
}

@test "an authority from a saved connection that can't be reached is named in the error" {
    save_unreachable

    run --separate-stderr phemera project list
    [ "$status" -eq 1 ]
    [[ $stderr == *"can't reach the authority at https://127.0.0.1:9"* ]]
}

@test "saved connections that can't be read are named in the error" {
    save_broken

    run --separate-stderr phemera project list
    [ "$status" -eq 1 ]
    [[ $stderr == *"$saved"* ]]
}

@test "saved connections that can't be read are ignored when PHEMERA_SERVER is set" {
    save_broken

    PHEMERA_SERVER=https://127.0.0.1:9 PHEMERA_TOKEN=secret run --separate-stderr phemera project list
    [ "$status" -eq 1 ]
    [[ $stderr == *"can't reach the authority at https://127.0.0.1:9"* ]]
}

@test "a missing configuration directory is ignored when PHEMERA_SERVER is set" {
    run --separate-stderr env -u HOME -u XDG_CONFIG_HOME PHEMERA_SERVER=https://127.0.0.1:9 phemera project list
    [ "$status" -eq 1 ]
    [[ $stderr == *"can't reach the authority at https://127.0.0.1:9"* ]]
}

@test "login --print-token works without a configuration directory" {
    run --separate-stderr env -u HOME -u XDG_CONFIG_HOME phemera login https://127.0.0.1:9 --print-token < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"no user name on stdin"* ]]
}

@test "login fails before it asks for a password when it can't save the session" {
    run --separate-stderr env -u HOME -u XDG_CONFIG_HOME phemera login https://127.0.0.1:9 < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"saved connections"* ]]
    [[ $stderr != *"Signing in"* ]]
}

@test "a command that reads the saved connections fails without a configuration directory" {
    run --separate-stderr env -u HOME -u XDG_CONFIG_HOME phemera --connection first project list
    [ "$status" -eq 1 ]
    [[ $stderr == *"finding the directory for the saved connections"* ]]
}

@test "saved connections that other users can read are refused" {
    save_unreachable
    chmod 644 "$saved"

    run --separate-stderr phemera project list
    [ "$status" -eq 1 ]
    [[ $stderr == *"chmod 600 $saved"* ]]
}

@test "a directory for saved connections that other users can use is refused" {
    save_unreachable
    chmod 755 "$XDG_CONFIG_HOME/phemera"

    run --separate-stderr phemera connections list
    [ "$status" -eq 1 ]
    [[ $stderr == *"chmod 700 $XDG_CONFIG_HOME/phemera"* ]]

    run --separate-stderr phemera login https://127.0.0.1:9 < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"chmod 700 $XDG_CONFIG_HOME/phemera"* ]]
    [[ $stderr != *"Signing in"* ]]
}

@test "a HOME that isn't an absolute path is refused" {
    run --separate-stderr env -u XDG_CONFIG_HOME HOME=. phemera connections list
    [ "$status" -eq 1 ]
    [[ $stderr == *"absolute path"* ]]
}

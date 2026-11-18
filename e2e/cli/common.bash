# Gives the test its own configuration directory and clears the variables
# that choose an authority.
isolate() {
    export SSL_CERT_FILE=$TIDEGATE_TEST_CA_FILE
    export HOME=$BATS_TEST_TMPDIR/home
    export XDG_CONFIG_HOME=$BATS_TEST_TMPDIR/config
    unset TIDEGATE_CONNECTION TIDEGATE_SERVER TIDEGATE_TOKEN
    saved=$XDG_CONFIG_HOME/tidegate/connections.json
}

# Runs tidegate with the arguments and the admin's user name and password on stdin.
sign_in() {
    printf '%s\n%s\n' "$TIDEGATE_TEST_USER" "$TIDEGATE_TEST_PASSWORD" | tidegate "$@"
}

# Prints a new session token and saves nothing.
print_token() {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --print-token 2>/dev/null
}

# Writes its argument as the file of saved connections, private to the user.
save() {
    mkdir -p "$XDG_CONFIG_HOME"
    mkdir -m 700 "$XDG_CONFIG_HOME/tidegate"
    (umask 077 && echo "$1" > "$saved")
}

# Saves the connection "gone" to a server that nothing listens on.
save_unreachable() {
    save '{"connections": {"gone": {"server": "https://127.0.0.1:9", "token": "secret"}}}'
}

# Saves a file that isn't JSON in place of the connections.
save_broken() {
    save 'not json'
}

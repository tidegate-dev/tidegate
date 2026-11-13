bats_require_minimum_version 1.5.0

setup() {
    load common
    isolate
}

# Signing in

@test "after login a command works without TIDEGATE_TOKEN" {
    run --separate-stderr sign_in login "$TIDEGATE_TEST_AUTHORITY"
    [ "$status" -eq 0 ]
    [ -z "$output" ]

    run tidegate dummy write hello
    [ "$status" -eq 0 ]

    run tidegate dummy read "$output"
    [ "$status" -eq 0 ]
    [ "$output" = hello ]
}

@test "login says which server it signs in to" {
    run --separate-stderr sign_in login "$TIDEGATE_TEST_AUTHORITY"
    [ "$status" -eq 0 ]
    [[ $stderr == *"Signing in at $TIDEGATE_TEST_AUTHORITY"* ]]
}

@test "login with a wrong password fails and saves nothing" {
    run --separate-stderr tidegate login "$TIDEGATE_TEST_AUTHORITY" \
        < <(printf '%s\n%s\n' "$TIDEGATE_TEST_USER" 'not the password at all')
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"wrong user name or password"* ]]
    [ ! -e "$saved" ]
}

@test "login with an unknown user fails like a wrong password" {
    run --separate-stderr tidegate login "$TIDEGATE_TEST_AUTHORITY" \
        < <(printf '%s\n%s\n' nobody "$TIDEGATE_TEST_PASSWORD")
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"wrong user name or password"* ]]
}

@test "a connection without a name is saved under the URL as given" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY/"

    [ "$(jq -r '.connections | keys[]' "$saved")" = "$TIDEGATE_TEST_AUTHORITY/" ]
    [ "$(jq -r '.connections[].server' "$saved")" = "$TIDEGATE_TEST_AUTHORITY/" ]
}

@test "login saves the connection under the name from --connection" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection production
    sign_in --connection staging login "$TIDEGATE_TEST_AUTHORITY"

    [ "$(jq -r '.connections | keys | join(",")' "$saved")" = production,staging ]
}

@test "login with a URL ignores TIDEGATE_CONNECTION" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection production

    TIDEGATE_CONNECTION=production run sign_in login "${TIDEGATE_TEST_AUTHORITY/127.0.0.1/localhost}"
    [ "$status" -eq 0 ]
    [ "$(jq -r '.connections.production.server' "$saved")" = "$TIDEGATE_TEST_AUTHORITY" ]
    [ "$(jq '.connections | length' "$saved")" -eq 2 ]
}

@test "login refuses a name that is saved for another server" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection production

    run --separate-stderr tidegate login https://127.0.0.1:9 --connection production < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"tidegate connections remove"* ]]
    [ "$(jq -r '.connections.production.server' "$saved")" = "$TIDEGATE_TEST_AUTHORITY" ]
}

@test "login with two URLs fails" {
    run --separate-stderr tidegate login "$TIDEGATE_TEST_AUTHORITY" "$TIDEGATE_TEST_AUTHORITY" < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"at most one URL"* ]]
}

# Signing in again

@test "login without a URL signs in again to the only saved connection" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection production
    tidegate logout

    run sign_in login
    [ "$status" -eq 0 ]
    [ "$(jq -r '.connections.production.server' "$saved")" = "$TIDEGATE_TEST_AUTHORITY" ]

    run tidegate dummy write hello
    [ "$status" -eq 0 ]
}

@test "login with --connection signs in again to that connection" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection first
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection second
    before=$(jq -r '.connections.first.token' "$saved")

    run sign_in login --connection first
    [ "$status" -eq 0 ]
    [ "$(jq '.connections | length' "$saved")" -eq 2 ]
    [ "$(jq -r '.connections.first.token' "$saved")" != "$before" ]

    run tidegate --connection first dummy write hello
    [ "$status" -eq 0 ]
}

@test "login without a URL follows TIDEGATE_CONNECTION" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection first
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection second
    before=$(jq -r '.connections.second.token' "$saved")

    TIDEGATE_CONNECTION=second run sign_in login
    [ "$status" -eq 0 ]
    [ "$(jq -r '.connections.second.token' "$saved")" != "$before" ]
}

@test "signing in again ends the session it replaces" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    replaced=$(jq -r '.connections[].token' "$saved")

    sign_in login

    run tidegate dummy write hello
    [ "$status" -eq 0 ]

    TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY TIDEGATE_TOKEN=$replaced run --separate-stderr tidegate dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"doesn't accept the session token in TIDEGATE_TOKEN"* ]]
}

@test "login with the URL of a named connection signs in again to that connection" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection production

    run sign_in login "$TIDEGATE_TEST_AUTHORITY"
    [ "$status" -eq 0 ]
    [ "$(jq -r '.connections | keys[]' "$saved")" = production ]
}

@test "login with a URL that several connections use asks for --connection" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection first
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection second

    run --separate-stderr tidegate login "$TIDEGATE_TEST_AUTHORITY" < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *'"first", "second"'* ]]
    [[ $stderr == *"--connection"* ]]
}

@test "login with a name that isn't saved and no URL fails" {
    run --separate-stderr tidegate login --connection production < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *'no saved connection named "production"'* ]]
    [[ $stderr == *"tidegate login <url> --connection"* ]]
}

@test "login without a URL and with nothing saved says to give a URL" {
    run --separate-stderr tidegate login < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"tidegate login <url>"* ]]
}

# Choosing a connection

@test "a command with nothing saved says to run tidegate login" {
    run --separate-stderr tidegate dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"Run \`tidegate login <url>\`"* ]]
}

@test "with several saved connections a command says to choose one" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection first
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection second

    run --separate-stderr tidegate dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *'"first", "second"'* ]]
}

@test "--connection and TIDEGATE_CONNECTION choose a connection, and the flag wins" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection first
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection second
    tidegate --connection second logout

    run tidegate --connection first dummy write hello
    [ "$status" -eq 0 ]

    run tidegate dummy write hello --connection first
    [ "$status" -eq 0 ]

    TIDEGATE_CONNECTION=first run tidegate dummy write hello
    [ "$status" -eq 0 ]

    TIDEGATE_CONNECTION=second run tidegate dummy write hello
    [ "$status" -eq 1 ]

    TIDEGATE_CONNECTION=second run tidegate --connection first dummy write hello
    [ "$status" -eq 0 ]
}

@test "a command with a name that isn't saved fails" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection first

    run --separate-stderr tidegate --connection second dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *'no saved connection named "second"'* ]]
}

# Signing out

@test "after logout a command fails and says to run tidegate login" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    token=$(jq -r '.connections[].token' "$saved")

    run tidegate logout
    [ "$status" -eq 0 ]
    [ -z "$output" ]
    [ "$(jq -r '.connections[].token' "$saved")" = null ]
    [ "$(jq -r '.connections[].server' "$saved")" = "$TIDEGATE_TEST_AUTHORITY" ]

    run --separate-stderr tidegate dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"Run \`tidegate login --connection \"$TIDEGATE_TEST_AUTHORITY\"\`"* ]]

    TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY TIDEGATE_TOKEN=$token run --separate-stderr tidegate dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"doesn't accept the session token in TIDEGATE_TOKEN"* ]]
}

@test "logout of a connection that is signed out says so and calls no authority" {
    save '{"connections": {"gone": {"server": "https://127.0.0.1:9"}}}'

    run --separate-stderr tidegate logout
    [ "$status" -eq 0 ]
    [ -z "$output" ]
    [[ $stderr == *'"gone" is already signed out'* ]]
}

@test "logout removes a saved session that the authority no longer accepts" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY TIDEGATE_TOKEN=$(jq -r '.connections[].token' "$saved") tidegate logout

    run tidegate logout
    [ "$status" -eq 0 ]
    [ "$(jq -r '.connections[].token' "$saved")" = null ]
}

@test "logging out of one connection keeps the session of the other" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection first
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection second

    run tidegate --connection first logout
    [ "$status" -eq 0 ]

    run tidegate --connection second dummy write hello
    [ "$status" -eq 0 ]

    run --separate-stderr tidegate --connection first dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *'Run `tidegate login --connection "first"`'* ]]
}

@test "logout keeps the session and says so when the authority can't be reached" {
    save_unreachable

    run --separate-stderr tidegate logout
    [ "$status" -eq 1 ]
    [[ $stderr == *"can't reach the authority at https://127.0.0.1:9"* ]]
    [[ $stderr == *"still saved"* ]]
    [[ $stderr == *'tidegate logout --connection "gone"'* ]]
    [ "$(jq -r '.connections.gone.token' "$saved")" = secret ]
}

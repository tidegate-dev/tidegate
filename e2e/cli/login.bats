bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
    # The tests share one database, so each one creates projects of its own.
    project=login-$BATS_TEST_NUMBER
}

# Signing in

@test "after login a command works without PHEMERA_TOKEN" {
    run --separate-stderr sign_in login "$PHEMERA_TEST_AUTHORITY"
    [ "$status" -eq 0 ]
    [ -z "$output" ]

    run phemera project create "$project"
    [ "$status" -eq 0 ]

    run phemera project list
    [ "$status" -eq 0 ]
    grep -qE " $project\$" <<<"$output"
}

@test "login says which server it signs in to" {
    run --separate-stderr sign_in login "$PHEMERA_TEST_AUTHORITY"
    [ "$status" -eq 0 ]
    [[ $stderr == *"Signing in at $PHEMERA_TEST_AUTHORITY"* ]]
}

@test "login with a wrong password fails and saves nothing" {
    run --separate-stderr phemera login "$PHEMERA_TEST_AUTHORITY" \
        < <(printf '%s\n%s\n' "$PHEMERA_TEST_USER" 'not the password at all')
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"wrong user name or password"* ]]
    [ ! -e "$saved" ]
}

@test "login with an unknown user fails like a wrong password" {
    run --separate-stderr phemera login "$PHEMERA_TEST_AUTHORITY" \
        < <(printf '%s\n%s\n' nobody "$PHEMERA_TEST_PASSWORD")
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"wrong user name or password"* ]]
}

@test "a connection without a name is saved under the URL as given" {
    sign_in login "$PHEMERA_TEST_AUTHORITY/"

    [ "$(jq -r '.connections | keys[]' "$saved")" = "$PHEMERA_TEST_AUTHORITY/" ]
    [ "$(jq -r '.connections[].server' "$saved")" = "$PHEMERA_TEST_AUTHORITY/" ]
}

@test "login saves the connection under the name from --connection" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection production
    sign_in --connection staging login "$PHEMERA_TEST_AUTHORITY"

    [ "$(jq -r '.connections | keys | join(",")' "$saved")" = production,staging ]
}

@test "login with a URL ignores PHEMERA_CONNECTION" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection production

    PHEMERA_CONNECTION=production run sign_in login "${PHEMERA_TEST_AUTHORITY/127.0.0.1/localhost}"
    [ "$status" -eq 0 ]
    [ "$(jq -r '.connections.production.server' "$saved")" = "$PHEMERA_TEST_AUTHORITY" ]
    [ "$(jq '.connections | length' "$saved")" -eq 2 ]
}

@test "login refuses a name that is saved for another server" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection production

    run --separate-stderr phemera login https://127.0.0.1:9 --connection production < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"phemera connections remove"* ]]
    [ "$(jq -r '.connections.production.server' "$saved")" = "$PHEMERA_TEST_AUTHORITY" ]
}

@test "login with two URLs fails" {
    run --separate-stderr phemera login "$PHEMERA_TEST_AUTHORITY" "$PHEMERA_TEST_AUTHORITY" < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"at most one URL"* ]]
}

# Signing in again

@test "login without a URL signs in again to the only saved connection" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection production
    phemera logout

    run sign_in login
    [ "$status" -eq 0 ]
    [ "$(jq -r '.connections.production.server' "$saved")" = "$PHEMERA_TEST_AUTHORITY" ]

    run phemera project create "$project"
    [ "$status" -eq 0 ]
}

@test "login with --connection signs in again to that connection" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection second
    before=$(jq -r '.connections.first.token' "$saved")

    run sign_in login --connection first
    [ "$status" -eq 0 ]
    [ "$(jq '.connections | length' "$saved")" -eq 2 ]
    [ "$(jq -r '.connections.first.token' "$saved")" != "$before" ]

    run phemera --connection first project create "$project"
    [ "$status" -eq 0 ]
}

@test "login without a URL follows PHEMERA_CONNECTION" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection second
    before=$(jq -r '.connections.second.token' "$saved")

    PHEMERA_CONNECTION=second run sign_in login
    [ "$status" -eq 0 ]
    [ "$(jq -r '.connections.second.token' "$saved")" != "$before" ]
}

@test "signing in again ends the session it replaces" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"
    replaced=$(jq -r '.connections[].token' "$saved")

    sign_in login

    run phemera project create "$project"
    [ "$status" -eq 0 ]

    PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY PHEMERA_TOKEN=$replaced run --separate-stderr phemera project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *"doesn't accept the session token in PHEMERA_TOKEN"* ]]
}

@test "login with the URL of a named connection signs in again to that connection" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection production

    run sign_in login "$PHEMERA_TEST_AUTHORITY"
    [ "$status" -eq 0 ]
    [ "$(jq -r '.connections | keys[]' "$saved")" = production ]
}

@test "login with a URL that several connections use asks for --connection" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection second

    run --separate-stderr phemera login "$PHEMERA_TEST_AUTHORITY" < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *'"first", "second"'* ]]
    [[ $stderr == *"--connection"* ]]
}

@test "login with a name that isn't saved and no URL fails" {
    run --separate-stderr phemera login --connection production < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *'no saved connection named "production"'* ]]
    [[ $stderr == *"phemera login <url> --connection"* ]]
}

@test "login without a URL and with nothing saved says to give a URL" {
    run --separate-stderr phemera login < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"phemera login <url>"* ]]
}

# Choosing a connection

@test "a command with nothing saved says to run phemera login" {
    run --separate-stderr phemera project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *"Run \`phemera login <url>\`"* ]]
}

@test "with several saved connections a command says to choose one" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection second

    run --separate-stderr phemera project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *'"first", "second"'* ]]
}

@test "--connection and PHEMERA_CONNECTION choose a connection, and the flag wins" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection second
    phemera --connection second logout

    run phemera --connection first project create "$project-1"
    [ "$status" -eq 0 ]

    run phemera project create "$project-2" --connection first
    [ "$status" -eq 0 ]

    PHEMERA_CONNECTION=first run phemera project create "$project-3"
    [ "$status" -eq 0 ]

    PHEMERA_CONNECTION=second run phemera project create "$project-4"
    [ "$status" -eq 1 ]

    PHEMERA_CONNECTION=second run phemera --connection first project create "$project-5"
    [ "$status" -eq 0 ]
}

@test "a command with a name that isn't saved fails" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first

    run --separate-stderr phemera --connection second project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *'no saved connection named "second"'* ]]
}

# Signing out

@test "after logout a command fails and says to run phemera login" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"
    token=$(jq -r '.connections[].token' "$saved")

    run phemera logout
    [ "$status" -eq 0 ]
    [ -z "$output" ]
    [ "$(jq -r '.connections[].token' "$saved")" = null ]
    [ "$(jq -r '.connections[].server' "$saved")" = "$PHEMERA_TEST_AUTHORITY" ]

    run --separate-stderr phemera project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *"Run \`phemera login --connection \"$PHEMERA_TEST_AUTHORITY\"\`"* ]]

    PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY PHEMERA_TOKEN=$token run --separate-stderr phemera project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *"doesn't accept the session token in PHEMERA_TOKEN"* ]]
}

@test "logout of a connection that is signed out says so and calls no authority" {
    save '{"connections": {"gone": {"server": "https://127.0.0.1:9"}}}'

    run --separate-stderr phemera logout
    [ "$status" -eq 0 ]
    [ -z "$output" ]
    [[ $stderr == *'"gone" is already signed out'* ]]
}

@test "logout removes a saved session that the authority no longer accepts" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"
    PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY PHEMERA_TOKEN=$(jq -r '.connections[].token' "$saved") phemera logout

    run phemera logout
    [ "$status" -eq 0 ]
    [ "$(jq -r '.connections[].token' "$saved")" = null ]
}

@test "logging out of one connection keeps the session of the other" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection second

    run phemera --connection first logout
    [ "$status" -eq 0 ]

    run phemera --connection second project create "$project"
    [ "$status" -eq 0 ]

    run --separate-stderr phemera --connection first project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *'Run `phemera login --connection "first"`'* ]]
}

@test "logout keeps the session and says so when the authority can't be reached" {
    save_unreachable

    run --separate-stderr phemera logout
    [ "$status" -eq 1 ]
    [[ $stderr == *"can't reach the authority at https://127.0.0.1:9"* ]]
    [[ $stderr == *"still saved"* ]]
    [[ $stderr == *'phemera logout --connection "gone"'* ]]
    [ "$(jq -r '.connections.gone.token' "$saved")" = secret ]
}

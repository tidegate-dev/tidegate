bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
}

@test "connections list prints every saved connection without its token" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection second
    phemera --connection second logout
    token=$(jq -r '.connections.first.token' "$saved")

    run phemera connections list
    [ "$status" -eq 0 ]
    [[ ${lines[0]} == NAME*SERVER*TOKEN ]]
    [[ ${lines[1]} == first*"$PHEMERA_TEST_AUTHORITY"*saved ]]
    [[ ${lines[2]} == second*"$PHEMERA_TEST_AUTHORITY"*none ]]
    [[ $output != *"$token"* ]]
}

@test "connections list prints json" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first

    run phemera -o json connections list
    [ "$status" -eq 0 ]
    [ "$(jq -r '.connections[0].name' <<<"$output")" = first ]
    [ "$(jq -r '.connections[0].server' <<<"$output")" = "$PHEMERA_TEST_AUTHORITY" ]
    [ "$(jq -r '.connections[0].tokenSaved' <<<"$output")" = true ]
    [ "$(jq -r '.connections[0] | has("token")' <<<"$output")" = false ]
}

@test "connections list works when nothing is saved" {
    run phemera connections list
    [ "$status" -eq 0 ]
    [[ $output == *"phemera login <url>"* ]]

    run phemera -o json connections list
    [ "$status" -eq 0 ]
    [ "$(jq '.connections | length' <<<"$output")" -eq 0 ]
}

@test "connections current prints the only saved connection" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first

    run phemera connections current
    [ "$status" -eq 0 ]
    [ "$output" = "first"$'\t'"$PHEMERA_TEST_AUTHORITY"$'\t'"saved" ]
}

@test "connections current follows --connection and PHEMERA_CONNECTION" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection second

    run phemera --connection first -o json connections current
    [ "$status" -eq 0 ]
    [ "$(jq -r .name <<<"$output")" = first ]

    PHEMERA_CONNECTION=second run phemera -o json connections current
    [ "$status" -eq 0 ]
    [ "$(jq -r .name <<<"$output")" = second ]
}

@test "connections current fails when several are saved and none is chosen" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection second

    run --separate-stderr phemera connections current
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *'"first", "second"'* ]]
}

@test "connections current fails when nothing is saved" {
    run --separate-stderr phemera connections current
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"phemera login <url>"* ]]
}

@test "connections current makes no call to the authority" {
    save_unreachable

    run phemera connections current
    [ "$status" -eq 0 ]
    [ "$output" = "gone"$'\t'"https://127.0.0.1:9"$'\t'"saved" ]
}

@test "connections remove revokes the token and removes the connection" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection second
    token=$(jq -r '.connections.first.token' "$saved")

    run phemera connections remove first
    [ "$status" -eq 0 ]
    [ -z "$output" ]
    [ "$(jq -r '.connections | keys[]' "$saved")" = second ]

    PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY PHEMERA_TOKEN=$token run --separate-stderr phemera project create payments
    [ "$status" -eq 1 ]
    [[ $stderr == *"doesn't accept the session token in PHEMERA_TOKEN"* ]]
}

@test "connections remove removes a connection that is signed out" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first
    phemera logout

    run phemera connections remove first
    [ "$status" -eq 0 ]
    [ "$(jq '.connections | length' "$saved")" -eq 0 ]
}

@test "connections remove removes the connection and warns when the authority can't be reached" {
    save_unreachable

    run --separate-stderr phemera connections remove gone
    [ "$status" -eq 0 ]
    [[ $stderr == *"Warning"*"still valid"* ]]
    [ "$(jq '.connections | length' "$saved")" -eq 0 ]
}

@test "connections remove fails for a name that isn't saved" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first

    run --separate-stderr phemera connections remove second
    [ "$status" -eq 1 ]
    [[ $stderr == *'no saved connection named "second"'* ]]
    [ "$(jq '.connections | length' "$saved")" -eq 1 ]
}

@test "connections remove for a name that isn't saved doesn't say to run login" {
    run --separate-stderr phemera connections remove first
    [ "$status" -eq 1 ]
    [[ $stderr == *'no saved connection named "first"'* ]]
    [[ $stderr != *"phemera login"* ]]
}

@test "connections remove wants exactly one name" {
    run --separate-stderr phemera connections remove
    [ "$status" -eq 1 ]
    [[ $stderr == *"exactly one name"* ]]
}

@test "connections current fails when PHEMERA_SERVER is set" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first

    PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY run --separate-stderr phemera connections current
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"no saved connection"* ]]
}

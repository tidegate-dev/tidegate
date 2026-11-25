bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
    # The tests share one database, so each one creates projects of its own.
    project=script-$BATS_TEST_NUMBER
}

@test "login --print-token prints a token and saves nothing" {
    run --separate-stderr sign_in login "$PHEMERA_TEST_AUTHORITY" --print-token
    [ "$status" -eq 0 ]
    [ -n "$output" ]
    [ ! -e "$saved" ]
}

@test "login --print-token without a URL fails" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"

    run --separate-stderr phemera login --print-token < /dev/null
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"phemera login <url> --print-token"* ]]
}

@test "login with a URL, --connection and --print-token fails" {
    run --separate-stderr phemera login "$PHEMERA_TEST_AUTHORITY" --connection production --print-token < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"takes no --connection"* ]]
}

@test "a script with PHEMERA_SERVER and PHEMERA_TOKEN can create a project and list it" {
    export PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY
    PHEMERA_TOKEN=$(print_token)
    export PHEMERA_TOKEN

    run phemera project create "$project"
    [ "$status" -eq 0 ]

    run phemera project list
    [ "$status" -eq 0 ]
    grep -qE " $project\$" <<<"$output"
    [ ! -e "$saved" ]
}

@test "a call with PHEMERA_SERVER and no token says to set PHEMERA_TOKEN" {
    PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY run --separate-stderr phemera project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *"Set PHEMERA_TOKEN"* ]]
}

@test "a saved session isn't sent to the server in PHEMERA_SERVER" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"

    PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY run --separate-stderr phemera project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *"Set PHEMERA_TOKEN"* ]]
}

@test "PHEMERA_SERVER and PHEMERA_TOKEN are used instead of a saved connection" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"

    PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY PHEMERA_TOKEN=not-a-token run --separate-stderr phemera project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *"PHEMERA_TOKEN"* ]]
    [[ $stderr == *"--print-token"* ]]
}

@test "PHEMERA_TOKEN without PHEMERA_SERVER fails" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"

    PHEMERA_TOKEN=$(print_token) run --separate-stderr phemera project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *"PHEMERA_SERVER isn't"* ]]
}

@test "--connection next to PHEMERA_TOKEN fails" {
    sign_in login "$PHEMERA_TEST_AUTHORITY" --connection first

    PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY PHEMERA_TOKEN=$(print_token) \
        run --separate-stderr phemera --connection first project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *"not both"* ]]
}

@test "after logout the token in PHEMERA_TOKEN no longer works" {
    export PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY
    PHEMERA_TOKEN=$(print_token)
    export PHEMERA_TOKEN

    run phemera logout
    [ "$status" -eq 0 ]
    [ -z "$output" ]

    run --separate-stderr phemera project create "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *"PHEMERA_TOKEN"* ]]
}

@test "logging out one token keeps the others" {
    export PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY
    first=$(print_token)
    second=$(print_token)

    PHEMERA_TOKEN=$first run phemera logout
    [ "$status" -eq 0 ]

    PHEMERA_TOKEN=$second run phemera project create "$project"
    [ "$status" -eq 0 ]
}

@test "logout with a token from the environment keeps the saved session" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"

    PHEMERA_SERVER=$PHEMERA_TEST_AUTHORITY PHEMERA_TOKEN=$(print_token) run phemera logout
    [ "$status" -eq 0 ]

    run phemera project create "$project"
    [ "$status" -eq 0 ]
}

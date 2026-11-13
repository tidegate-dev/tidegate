bats_require_minimum_version 1.5.0

setup() {
    load common
    isolate
}

@test "login --print-token prints a token and saves nothing" {
    run --separate-stderr sign_in login "$TIDEGATE_TEST_AUTHORITY" --print-token
    [ "$status" -eq 0 ]
    [ -n "$output" ]
    [ ! -e "$saved" ]
}

@test "login --print-token without a URL fails" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    run --separate-stderr tidegate login --print-token < /dev/null
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"tidegate login <url> --print-token"* ]]
}

@test "login with a URL, --connection and --print-token fails" {
    run --separate-stderr tidegate login "$TIDEGATE_TEST_AUTHORITY" --connection production --print-token < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"takes no --connection"* ]]
}

@test "a script with TIDEGATE_SERVER and TIDEGATE_TOKEN can store a text and read it back" {
    export TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY
    TIDEGATE_TOKEN=$(print_token)
    export TIDEGATE_TOKEN

    run tidegate dummy write hello
    [ "$status" -eq 0 ]

    run tidegate dummy read "$output"
    [ "$status" -eq 0 ]
    [ "$output" = hello ]
    [ ! -e "$saved" ]
}

@test "a call with TIDEGATE_SERVER and no token says to set TIDEGATE_TOKEN" {
    TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY run --separate-stderr tidegate dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"Set TIDEGATE_TOKEN"* ]]
}

@test "a saved session isn't sent to the server in TIDEGATE_SERVER" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY run --separate-stderr tidegate dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"Set TIDEGATE_TOKEN"* ]]
}

@test "TIDEGATE_SERVER and TIDEGATE_TOKEN are used instead of a saved connection" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY TIDEGATE_TOKEN=not-a-token run --separate-stderr tidegate dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"TIDEGATE_TOKEN"* ]]
    [[ $stderr == *"--print-token"* ]]
}

@test "TIDEGATE_TOKEN without TIDEGATE_SERVER fails" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    TIDEGATE_TOKEN=$(print_token) run --separate-stderr tidegate dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"TIDEGATE_SERVER isn't"* ]]
}

@test "--connection next to TIDEGATE_TOKEN fails" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY" --connection first

    TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY TIDEGATE_TOKEN=$(print_token) \
        run --separate-stderr tidegate --connection first dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"not both"* ]]
}

@test "after logout the token in TIDEGATE_TOKEN no longer works" {
    export TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY
    TIDEGATE_TOKEN=$(print_token)
    export TIDEGATE_TOKEN

    run tidegate logout
    [ "$status" -eq 0 ]
    [ -z "$output" ]

    run --separate-stderr tidegate dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"TIDEGATE_TOKEN"* ]]
}

@test "logging out one token keeps the others" {
    export TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY
    first=$(print_token)
    second=$(print_token)

    TIDEGATE_TOKEN=$first run tidegate logout
    [ "$status" -eq 0 ]

    TIDEGATE_TOKEN=$second run tidegate dummy write hello
    [ "$status" -eq 0 ]
}

@test "logout with a token from the environment keeps the saved session" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY TIDEGATE_TOKEN=$(print_token) run tidegate logout
    [ "$status" -eq 0 ]

    run tidegate dummy write hello
    [ "$status" -eq 0 ]
}

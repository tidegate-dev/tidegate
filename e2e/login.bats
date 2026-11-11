bats_require_minimum_version 1.5.0

setup() {
    export SSL_CERT_FILE=$TIDEGATE_TEST_CA_FILE
}

login() {
    printf '%s\n%s\n' "$TIDEGATE_TEST_USER" "$TIDEGATE_TEST_PASSWORD" |
        tidegate --server "$TIDEGATE_TEST_AUTHORITY" login
}

@test "a signed-in user can store a text and read it back" {
    TIDEGATE_TOKEN=$(login)
    export TIDEGATE_TOKEN

    run tidegate --server "$TIDEGATE_TEST_AUTHORITY" dummy write hello
    [ "$status" -eq 0 ]

    run tidegate --server "$TIDEGATE_TEST_AUTHORITY" dummy read "$output"
    [ "$status" -eq 0 ]
    [ "$output" = hello ]
}

@test "login with a wrong password fails" {
    run --separate-stderr tidegate --server "$TIDEGATE_TEST_AUTHORITY" login \
        < <(printf '%s\n%s\n' "$TIDEGATE_TEST_USER" 'not the password at all')
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"wrong user name or password"* ]]
}

@test "login with an unknown user fails like a wrong password" {
    run --separate-stderr tidegate --server "$TIDEGATE_TEST_AUTHORITY" login \
        < <(printf '%s\n%s\n' nobody "$TIDEGATE_TEST_PASSWORD")
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"wrong user name or password"* ]]
}

@test "a call without a session token says to run tidegate login" {
    unset TIDEGATE_TOKEN

    run --separate-stderr tidegate --server "$TIDEGATE_TEST_AUTHORITY" dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"Run \`tidegate login\`"* ]]
}

@test "after logout the session token no longer works" {
    TIDEGATE_TOKEN=$(login)
    export TIDEGATE_TOKEN

    run tidegate --server "$TIDEGATE_TEST_AUTHORITY" logout
    [ "$status" -eq 0 ]
    [ -z "$output" ]

    run --separate-stderr tidegate --server "$TIDEGATE_TEST_AUTHORITY" dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"Run \`tidegate login\`"* ]]
}

@test "logging out one session keeps the others" {
    first=$(login)
    second=$(login)

    TIDEGATE_TOKEN=$first run tidegate --server "$TIDEGATE_TEST_AUTHORITY" logout
    [ "$status" -eq 0 ]

    TIDEGATE_TOKEN=$second run tidegate --server "$TIDEGATE_TEST_AUTHORITY" dummy write hello
    [ "$status" -eq 0 ]
}
